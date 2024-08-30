package handler

import (
	"context"
	"github.com/aws/aws-lambda-go/events"
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

	datasetNodeId, ok := h.request.QueryStringParameters["dataset_id"]
	if !ok {
		return h.logAndBuildError("query param 'dataset_id' is required", http.StatusBadRequest), nil
	}

	manifestResult, err := h.datasetsService.GetManifest(ctx, datasetNodeId)
	if err != nil {
		return nil, err
	}

	return h.buildResponse(manifestResult, http.StatusOK)
}
