package handler

import (
	"github.com/aws/aws-lambda-go/events"
	"github.com/pennsieve/datasets-service/api/models"
	"github.com/pennsieve/pennsieve-go-api/pkg/authorizer"
	"github.com/pennsieve/pennsieve-go-api/pkg/models/dataset"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCallGetTrashcan(t *testing.T) {
	expectedDatasetID := "N:Dataset:1234"
	req := newTestRequest("GET",
		"/datasets/trashcan",
		"getTrashcanRequestID",
		map[string]string{"dataset_id": expectedDatasetID},
		"")
	datasetsService := MockDatasetsService{}
	claims := authorizer.Claims{
		DatasetClaim: dataset.Claim{
			Role:   dataset.Viewer,
			NodeId: expectedDatasetID,
			IntId:  1234,
		}}
	handler := newTestHandler(req, &claims, &datasetsService)
	_, err := handler.handle()
	if assert.NoError(t, err) {
		assert.Equal(t, expectedDatasetID, datasetsService.ActualGetTrashcanArgs.DatasetID)
		assert.Equal(t, DefaultLimit, datasetsService.ActualGetTrashcanArgs.Limit)
		assert.Equal(t, DefaultOffset, datasetsService.ActualGetTrashcanArgs.Offset)
	}
}

func TestFail(t *testing.T) {
	assert.True(t, false)
}

func newTestRequest(method string, path string, requestID string, queryParams map[string]string, body string) *events.APIGatewayV2HTTPRequest {
	request := events.APIGatewayV2HTTPRequest{
		QueryStringParameters: queryParams,
		Body:                  body,
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			RequestID: requestID,
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: method,
				Path:   path,
			},
		},
	}
	return &request
}

func newTestHandler(request *events.APIGatewayV2HTTPRequest, claims *authorizer.Claims, mockService *MockDatasetsService) *RequestHandler {
	logger := log.WithFields(log.Fields{
		"requestID": request.RequestContext.RequestID,
	})
	requestHandler := RequestHandler{
		request:   request,
		requestID: request.RequestContext.RequestID,

		method:      request.RequestContext.HTTP.Method,
		path:        request.RequestContext.HTTP.Path,
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

	requestHandler.datasetsService = mockService
	return &requestHandler
}

type GetTrashcanArgs struct {
	DatasetID string
	Limit     int
	Offset    int
}

type MockDatasetsService struct {
	ActualGetTrashcanArgs  GetTrashcanArgs
	GetTrashcanReturnValue *models.TrashcanPage
	GetTrashcanReturnError error
}

func (m *MockDatasetsService) GetTrashcan(datasetID string, limit int, offset int) (*models.TrashcanPage, error) {
	m.ActualGetTrashcanArgs = GetTrashcanArgs{
		DatasetID: datasetID,
		Limit:     limit,
		Offset:    offset,
	}
	if m.GetTrashcanReturnError != nil {
		return &models.TrashcanPage{}, m.GetTrashcanReturnError
	}
	return m.GetTrashcanReturnValue, nil
}
