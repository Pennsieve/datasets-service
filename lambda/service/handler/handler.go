package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/pennsieve/datasets-service/api/logging"
	"github.com/pennsieve/datasets-service/api/models"
	"github.com/pennsieve/datasets-service/api/service"
	"github.com/pennsieve/pennsieve-go-core/pkg/authorizer"
	"log/slog"
	"strconv"
)

var (
	PennsieveDB *sql.DB
	S3Client    *s3.Client
	SNSClient   *sns.Client
	HandlerVars *models.HandlerVars
)

// Logging configuration used to live in an init() here that duplicated one in
// api/store and a third in api/logging. Go runs init() functions in
// dependency/import order, so "whichever ran last" silently decided the
// effective level. main.go now calls logging.SetDefaultFromEnv once instead.

func DatasetsServiceHandler(ctx context.Context, request events.APIGatewayV2HTTPRequest) (*events.APIGatewayV2HTTPResponse, error) {
	claims := authorizer.ParseClaims(request.RequestContext.Authorizer.Lambda)
	handler := NewHandler(&request, claims)
	
	// Route to appropriate service based on path
	path := request.RequestContext.HTTP.Path
	if path == "/shared-datasets" {
		handler = handler.WithCrossWorkspaceService()
	} else {
		handler = handler.WithDefaultService()
	}
	
	return handler.handle(ctx)
}

// RequestHandler wraps the incoming request with a logger and a service.DatasetsService.
// Some request params are pulled out for convenience. Use NewHandler followed by WithDefaultService to have things
// initialized nicely. Use WithService in tests where a specially constructed or mock service.DatasetsService is required.
type RequestHandler struct {
	request   *events.APIGatewayV2HTTPRequest
	requestID string
	traceID   string

	method      string
	path        string
	queryParams map[string]string
	body        string

	logger                        *slog.Logger
	datasetsService               service.DatasetsService
	crossWorkspaceDatasetsService service.CrossWorkspaceDatasetsService
	claims                        *authorizer.Claims
}

// NewHandler creates a RequestHandler whose logger carries, for the life of the
// invocation, both the API Gateway per-hop request id and this service's
// internal trace id. They are deliberately separate fields: the AWS id is minted
// fresh at each hop and only correlates within it, while the trace id is taken
// from an inbound correlation header when the caller supplied one, so it can tie
// together one logical operation across services. This logger is threaded down
// into the service and store layers so DB/S3/SNS failures carry the same context.
func NewHandler(request *events.APIGatewayV2HTTPRequest, claims *authorizer.Claims) *RequestHandler {
	method := request.RequestContext.HTTP.Method
	path := request.RequestContext.HTTP.Path
	reqID := request.RequestContext.RequestID
	trace, traceSource := traceID(request)

	logger := slog.Default().With(
		slog.String(logging.ApiGatewayRequestIdKey, reqID),
		slog.String(logging.TraceIdKey, trace),
		slog.String(logging.TraceIdSourceKey, traceSource),
	)
	requestHandler := RequestHandler{
		request:   request,
		requestID: reqID,
		traceID:   trace,

		method:      method,
		path:        path,
		queryParams: request.QueryStringParameters,
		body:        request.Body,

		logger: logger,
		claims: claims,
	}
	logger.Info("creating RequestHandler",
		slog.String(logging.MethodKey, requestHandler.method),
		slog.String(logging.PathKey, requestHandler.path),
		slog.Any(logging.QueryParamsKey, requestHandler.queryParams),
		slog.String(logging.RequestBodyKey, requestHandler.body),
		slog.Any(logging.ClaimsKey, requestHandler.claims))

	return &requestHandler
}

// WithDefaultService adds a new service.DatasetsService to the RequestHandler that
// has been initialized to use PennsieveDB as the SQL database pointed to the
// workspace in the RequestHandler's OrgClaim.
func (h *RequestHandler) WithDefaultService() *RequestHandler {
	srv := service.NewDatasetsService(PennsieveDB, S3Client, SNSClient, HandlerVars, int(h.claims.OrgClaim.IntId), h.logger)
	h.datasetsService = srv
	return h
}

// WithCrossWorkspaceService adds a new service.CrossWorkspaceDatasetsService to the RequestHandler
// for operations that span multiple workspaces
func (h *RequestHandler) WithCrossWorkspaceService() *RequestHandler {
	srv := service.NewCrossWorkspaceDatasetsService(PennsieveDB, h.logger)
	h.crossWorkspaceDatasetsService = srv
	return h
}

// WithService simply attaches the passed in service.DatasetsService to the RequestHandler. Used for
// tests that do not need to use PennsieveDB.
func (h *RequestHandler) WithService(dsService service.DatasetsService) *RequestHandler {
	h.datasetsService = dsService
	return h
}

func (h *RequestHandler) logAndBuildError(message string, status int) *events.APIGatewayV2HTTPResponse {
	h.logger.Error(message, slog.Int(logging.StatusKey, status))
	errorBody := fmt.Sprintf("{'message': '%s (requestID: %s)'}", message, h.requestID)
	return buildResponseFromString(errorBody, status)
}

func (h *RequestHandler) queryParamAsInt(paramName string, minValue, maxValue, defaultValue int) (int, error) {
	strValue, ok := h.request.QueryStringParameters[paramName]
	if !ok {
		return defaultValue, nil
	}
	v, err := strconv.Atoi(strValue)
	if err != nil {
		return 0, err
	}
	if v < minValue {
		return 0, fmt.Errorf("%d is less than min value %d for %q", v, minValue, paramName)
	}
	if v > maxValue {
		return 0, fmt.Errorf("%d is more than max value %d for %q", v, maxValue, paramName)
	}
	return v, nil
}

func (h *RequestHandler) buildResponse(body any, status int) (*events.APIGatewayV2HTTPResponse, error) {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		h.logger.Error("error marshalling response body",
			slog.Any(logging.ErrorKey, err))
		return nil, err
	}
	return buildResponseFromString(string(bodyBytes), status), nil
}

func buildResponseFromString(body string, status int) *events.APIGatewayV2HTTPResponse {
	response := events.APIGatewayV2HTTPResponse{
		Body:       body,
		StatusCode: status,
	}
	return &response
}
