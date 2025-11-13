package handler

import (
	"context"
	"github.com/aws/aws-lambda-go/events"
	"github.com/pennsieve/pennsieve-go-core/pkg/authorizer"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/pgdb"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/permissions"
	"math"
	"net/http"
)

type SharedDatasetsHandler struct {
	RequestHandler
}

func (h *SharedDatasetsHandler) handle(ctx context.Context) (*events.APIGatewayV2HTTPResponse, error) {
	switch h.method {
	case "GET":
		return h.get(ctx)
	default:
		return h.logAndBuildError("method not allowed: "+h.method, http.StatusMethodNotAllowed), nil
	}
}

func (h *SharedDatasetsHandler) get(ctx context.Context) (*events.APIGatewayV2HTTPResponse, error) {
	// Check if user has ViewFiles permission
	if authorized := authorizer.HasRole(*h.claims, permissions.ViewFiles); !authorized {
		return h.logAndBuildError("unauthorized", http.StatusUnauthorized), nil
	}

	// Check if the user is a guest in the organization
	if h.claims.OrgClaim.Role != pgdb.Guest {
		return h.logAndBuildError("this endpoint is only available for guest users", http.StatusForbidden), nil
	}

	// Extract pagination params
	limit, err := h.queryParamAsInt("limit", 1, 100, DefaultLimit)
	if err != nil {
		return h.logAndBuildError(err.Error(), http.StatusBadRequest), nil
	}

	offset, err := h.queryParamAsInt("offset", 0, math.MaxInt, DefaultOffset)
	if err != nil {
		return h.logAndBuildError(err.Error(), http.StatusBadRequest), nil
	}

	// Get user ID from claims
	userId := h.claims.UserClaim.Id

	// Call service to get shared datasets
	page, err := h.datasetsService.GetSharedDatasetsForGuest(ctx, userId, limit, offset)
	if err != nil {
		h.logger.Errorf("get shared datasets failed: %s", err)
		return nil, err
	}

	h.logger.Info("OK")
	return h.buildResponse(page, http.StatusOK)
}
