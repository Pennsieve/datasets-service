package handler

import (
	"context"
	"errors"
	"github.com/aws/aws-lambda-go/events"
	"github.com/pennsieve/datasets-service/api/models"
	"github.com/pennsieve/pennsieve-go-core/pkg/authorizer"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/permissions"
	"net/http"
)

type ManifestHandler struct {
	RequestHandler
}

func (h *ManifestHandler) handle(ctx context.Context) (*events.APIGatewayV2HTTPResponse, error) {
	switch h.method {
	case "GET":
		return h.get(ctx)
	default:
		return h.logAndBuildError("method not allowed: "+h.method, http.StatusMethodNotAllowed), nil
	}

}

func (h *ManifestHandler) get(ctx context.Context) (*events.APIGatewayV2HTTPResponse, error) {
	if authorized := authorizer.HasRole(*h.claims, permissions.ViewFiles); !authorized {
		return h.logAndBuildError("unauthorized", http.StatusUnauthorized), nil
	}

	datasetNodeID, ok := h.request.QueryStringParameters["dataset_id"]
	if !ok {
		return h.logAndBuildError("query param 'dataset_id' is required", http.StatusBadRequest), nil
	}

	// Triggering the worker lambda to create the manifest
	// TODO: heuristically run this directly in the current lambda based on dataset size
	page, err := h.datasetsService.TriggerAsyncGetManifest(ctx, datasetNodeID)

	if err == nil {
		h.logger.Info("OK")
		return h.buildResponse(page, http.StatusOK)
	}

	var datasetNotFoundError models.DatasetNotFoundError
	var packageNotFoundError models.PackageNotFoundError
	var folderNotFoundError models.FolderNotFoundError

	switch {
	case errors.As(err, &datasetNotFoundError):
		return h.logAndBuildError(err.Error(), http.StatusNotFound), nil
	case errors.As(err, &packageNotFoundError):
		return h.logAndBuildError(err.Error(), http.StatusBadRequest), nil
	case errors.As(err, &folderNotFoundError):
		return h.logAndBuildError(err.Error(), http.StatusBadRequest), nil
	default:
		h.logger.Errorf("get trashcan failed: %s", err)
		return nil, err
	}

}
