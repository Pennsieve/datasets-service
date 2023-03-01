package handler

import (
	"context"
	"github.com/aws/aws-lambda-go/events"
	"github.com/pennsieve/datasets-service/api/models"
	"github.com/pennsieve/pennsieve-go-core/pkg/authorizer"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/dataset"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/packageInfo/packageType"
	"github.com/stretchr/testify/assert"
	"net/http"
	"strconv"
	"testing"
)

type queryParamMap map[string]string

func TestTrashcanRoute(t *testing.T) {
	expectedDatasetID := "N:Dataset:1234"
	for tName, expectedQueryParams := range map[string]queryParamMap{
		"without root_node_id param": {"dataset_id": expectedDatasetID},
		"with root_node_id param":    {"dataset_id": expectedDatasetID, "root_node_id": "N:collection:abcd"},
		"with limit param":           {"dataset_id": expectedDatasetID, "root_node_id": "N:collection:abcd", "limit": "30"},
		"with offset param":          {"dataset_id": expectedDatasetID, "offset": "10"},
	} {
		req := newTestRequest("GET",
			"/datasets/trashcan",
			"getTrashcanRequestID",
			expectedQueryParams,
			"")
		datasetsService := MockDatasetsService{}
		claims := authorizer.Claims{
			DatasetClaim: dataset.Claim{
				Role:   dataset.Viewer,
				NodeId: expectedDatasetID,
				IntId:  1234,
			}}
		expectedLimit := DefaultLimit
		if limit, ok := expectedQueryParams["limit"]; ok {
			var err error
			expectedLimit, err = strconv.Atoi(limit)
			assert.NoError(t, err)
		}
		expectedOffset := DefaultOffset
		if offset, ok := expectedQueryParams["offset"]; ok {
			var err error
			expectedOffset, err = strconv.Atoi(offset)
			assert.NoError(t, err)
		}
		handler, err := NewHandler(req, &claims).WithService(&datasetsService)
		if assert.NoError(t, err) {
			t.Run(tName, func(t *testing.T) {
				_, err := handler.handle(context.Background())
				if assert.NoError(t, err) {
					assert.Equal(t, expectedDatasetID, datasetsService.ActualGetTrashcanArgs.DatasetID)
					assert.Equal(t, expectedQueryParams["root_node_id"], datasetsService.ActualGetTrashcanArgs.RootNodeID)
					assert.Equal(t, expectedLimit, datasetsService.ActualGetTrashcanArgs.Limit)
					assert.Equal(t, expectedOffset, datasetsService.ActualGetTrashcanArgs.Offset)
				}
			})
		}
	}
}

func TestTrashcanRouteHandledErrors(t *testing.T) {
	datasetID := "N:Dataset:1234"
	rootNodeID := "N:collection:abcd"
	for tName, tData := range map[string]struct {
		QueryParams         queryParamMap
		ServiceError        error
		ExpectedStatus      int
		ExpectedSubMessages []string
	}{
		"with too low limit": {
			QueryParams:         queryParamMap{"dataset_id": datasetID, "limit": "-1"},
			ExpectedStatus:      http.StatusBadRequest,
			ExpectedSubMessages: []string{"min value", "limit"}},
		"with too high limit": {
			QueryParams:         queryParamMap{"dataset_id": datasetID, "limit": "50000"},
			ExpectedStatus:      http.StatusBadRequest,
			ExpectedSubMessages: []string{"max value", "limit"}},
		"with too low offset": {
			QueryParams:         queryParamMap{"dataset_id": datasetID, "root_node_id": rootNodeID, "offset": "-4"},
			ExpectedStatus:      http.StatusBadRequest,
			ExpectedSubMessages: []string{"min value", "offset"}},
		"dataset not found": {
			QueryParams:         queryParamMap{"dataset_id": datasetID},
			ServiceError:        models.DatasetNotFoundError{Id: models.DatasetNodeId(datasetID)},
			ExpectedStatus:      http.StatusNotFound,
			ExpectedSubMessages: []string{"not found", datasetID},
		},
		"package not found": {
			QueryParams:         queryParamMap{"dataset_id": datasetID, "root_node_id": rootNodeID},
			ServiceError:        models.PackageNotFoundError{Id: models.PackageNodeId(rootNodeID), DatasetId: models.DatasetIntId(13)},
			ExpectedStatus:      http.StatusBadRequest,
			ExpectedSubMessages: []string{"not found", rootNodeID},
		},
		"package not a folder": {
			QueryParams:         queryParamMap{"dataset_id": datasetID, "root_node_id": rootNodeID},
			ServiceError:        models.FolderNotFoundError{NodeId: rootNodeID, ActualType: packageType.CSV},
			ExpectedStatus:      http.StatusBadRequest,
			ExpectedSubMessages: []string{"not found", rootNodeID, packageType.CSV.String()},
		},
	} {
		req := newTestRequest("GET",
			"/datasets/trashcan",
			"getTrashcanRequestID",
			tData.QueryParams,
			"")
		datasetsService := MockDatasetsService{GetTrashcanReturnError: tData.ServiceError}
		claims := authorizer.Claims{
			DatasetClaim: dataset.Claim{
				Role:   dataset.Viewer,
				NodeId: datasetID,
				IntId:  1234,
			}}
		handler, err := NewHandler(req, &claims).WithService(&datasetsService)
		if assert.NoError(t, err) {
			t.Run(tName, func(t *testing.T) {
				resp, err := handler.handle(context.Background())
				if assert.NoError(t, err) {
					assert.Equal(t, tData.ExpectedStatus, resp.StatusCode)
					for _, messageFragment := range tData.ExpectedSubMessages {
						assert.Contains(t, resp.Body, messageFragment)
					}
				}
			})
		}
	}
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

type GetTrashcanArgs struct {
	DatasetID  string
	RootNodeID string
	Limit      int
	Offset     int
}

type MockDatasetsService struct {
	ActualGetTrashcanArgs  GetTrashcanArgs
	GetTrashcanReturnValue *models.TrashcanPage
	GetTrashcanReturnError error
}

func (m *MockDatasetsService) GetTrashcanPage(_ context.Context, datasetID string, rootNodeId string, limit int, offset int) (*models.TrashcanPage, error) {
	m.ActualGetTrashcanArgs = GetTrashcanArgs{
		DatasetID:  datasetID,
		RootNodeID: rootNodeId,
		Limit:      limit,
		Offset:     offset,
	}
	if m.GetTrashcanReturnError != nil {
		return &models.TrashcanPage{}, m.GetTrashcanReturnError
	}
	return m.GetTrashcanReturnValue, nil
}
