package handler

import (
	"github.com/aws/aws-lambda-go/events"
	"github.com/pennsieve/pennsieve-go-api/pkg/authorizer"
	"github.com/pennsieve/pennsieve-go-api/pkg/models/permissions"
	"math"
	"net/http"
)

const (
	DefaultLimit  = 10
	DefaultOffset = 0
)

type TrashcanHandler struct {
	RequestHandler
}

func (h *TrashcanHandler) handle() (*events.APIGatewayV2HTTPResponse, error) {
	switch h.method {
	case "GET":
		return h.get()
	default:
		return h.logAndBuildError("method not allowed: "+h.method, http.StatusMethodNotAllowed), nil
	}

}

func (h *TrashcanHandler) get() (*events.APIGatewayV2HTTPResponse, error) {
	if authorized := authorizer.HasRole(*h.claims, permissions.ViewFiles); !authorized {
		return h.logAndBuildError("unauthorized", http.StatusUnauthorized), nil
	}

	datasetID, ok := h.request.QueryStringParameters["dataset_id"]
	if !ok {
		return h.logAndBuildError("query param 'dataset_id' is required", http.StatusBadRequest), nil
	}
	limit, err := h.queryParamAsInt("limit", 0, 100, DefaultLimit)
	if err != nil {
		return h.logAndBuildError(err.Error(), http.StatusBadRequest), nil
	}
	offset, err := h.queryParamAsInt("offset", 0, math.MaxInt, DefaultOffset)
	if err != nil {
		return h.logAndBuildError(err.Error(), http.StatusBadRequest), nil
	}
	page, err := h.datasetsService.GetTrashcan(datasetID, limit, offset)
	if err != nil {
		h.logger.Errorf("get trashcan failed: %s", err)
		return nil, err
	}
	return h.buildResponse(page, http.StatusOK)

}
