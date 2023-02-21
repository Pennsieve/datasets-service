package handler

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/pennsieve/datasets-service/api/service"
	"github.com/pennsieve/datasets-service/api/store"
	"github.com/pennsieve/pennsieve-go-api/pkg/authorizer"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strconv"
)

type RequestHandler struct {
	request   *events.APIGatewayV2HTTPRequest
	requestID string

	method      string
	path        string
	queryParams map[string]string
	body        string

	logger          *log.Entry
	datasetsService *service.DatasetsService
	claims          *authorizer.Claims
}

func (h *RequestHandler) handle() (*events.APIGatewayV2HTTPResponse, error) {

	switch h.path {
	case "/datasets/trashcan":
		trashcanHandler := TrashcanHandler{*h}
		return trashcanHandler.handle()
	default:
		return h.logAndBuildError("resource not found: "+h.path, http.StatusNotFound), nil
	}
}

func (h *RequestHandler) logAndBuildError(message string, status int) *events.APIGatewayV2HTTPResponse {
	h.logger.Error(message)
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
		h.logger.Errorf("error marshalling body: [%v]: %s", body, err)
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

func newService(claims *authorizer.Claims) (*service.DatasetsService, error) {
	str, err := store.NewDatasetStoreAtOrg(PennsieveDB, claims.OrgClaim.IntId)
	if err != nil {
		return nil, err
	}
	datasetsSvc := service.NewDatasetsService(str)
	return datasetsSvc, nil
}

func NewHandler(request *events.APIGatewayV2HTTPRequest) (*RequestHandler, error) {
	claims := authorizer.ParseClaims(request.RequestContext.Authorizer.Lambda)
	method := request.RequestContext.HTTP.Method
	path := request.RequestContext.HTTP.Path
	reqID := request.RequestContext.RequestID
	logger := log.WithFields(log.Fields{
		"requestID": reqID,
	})
	requestHandler := RequestHandler{
		request:   request,
		requestID: reqID,

		method:      method,
		path:        path,
		queryParams: request.QueryStringParameters,
		body:        request.Body,

		logger: logger,
		claims: claims,
	}
	logger.WithFields(log.Fields{
		"method":      requestHandler.method,
		"path":        requestHandler.path,
		"queryParams": requestHandler.queryParams,
		"requestBody": requestHandler.body,
		"claims":      requestHandler.claims}).Info("creating RequestHandler")

	datasetsService, err := newService(claims)

	if err != nil {
		logger.Error("unable to create DatasetsService", err)
		return nil, err
	}

	requestHandler.datasetsService = datasetsService
	return &requestHandler, nil
}
