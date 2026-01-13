package handler

import (
    "context"
    "github.com/aws/aws-lambda-go/events"
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
    // Shared datasets endpoint doesn't require specific dataset permissions
    // Just needs authenticated user
    if h.claims == nil || h.claims.UserClaim == nil {
        return h.logAndBuildError("unauthorized", http.StatusUnauthorized), nil
    }

    // Check that we have the cross-workspace service
    if h.crossWorkspaceDatasetsService == nil {
        return h.logAndBuildError("cross-workspace service not configured", http.StatusInternalServerError), nil
    }

    // Get query parameters
    limit, err := h.queryParamAsInt("limit", 0, 100, DefaultLimit)
    if err != nil {
        return h.logAndBuildError(err.Error(), http.StatusBadRequest), nil
    }
    offset, err := h.queryParamAsInt("offset", 0, math.MaxInt, DefaultOffset)
    if err != nil {
        return h.logAndBuildError(err.Error(), http.StatusBadRequest), nil
    }

    // Get user ID from claims
    userId := int(h.claims.UserClaim.Id)

    // Call cross-workspace service to get shared datasets
    page, err := h.crossWorkspaceDatasetsService.GetSharedDatasetsPage(ctx, userId, limit, offset)
    if err != nil {
        h.logger.Errorf("get shared datasets failed: %s", err)
        return nil, err
    }

    h.logger.Info("OK")
    return h.buildResponse(page, http.StatusOK)
}
