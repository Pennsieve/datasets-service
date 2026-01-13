package handler

import (
	"context"
	"net/http"
	"testing"

	"github.com/pennsieve/datasets-service/api/models"
	"github.com/pennsieve/pennsieve-go-core/pkg/authorizer"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockCrossWorkspaceDatasetsService struct {
	mock.Mock
}

func (m *MockCrossWorkspaceDatasetsService) GetSharedDatasetsPage(ctx context.Context, userId int, limit int, offset int) (*models.SharedDatasetsPage, error) {
	args := m.Called(ctx, userId, limit, offset)
	return args.Get(0).(*models.SharedDatasetsPage), args.Error(1)
}

func (m *MockCrossWorkspaceDatasetsService) OnGetSharedDatasetsPageReturn(userId int, limit int, offset int, returnedPage *models.SharedDatasetsPage) {
	m.On("GetSharedDatasetsPage", mock.Anything, userId, limit, offset).Return(returnedPage, nil)
}

func (m *MockCrossWorkspaceDatasetsService) OnGetSharedDatasetsPageFail(userId int, limit int, offset int, returnedError error) {
	m.On("GetSharedDatasetsPage", mock.Anything, userId, limit, offset).Return(&models.SharedDatasetsPage{}, returnedError)
}

func TestSharedDatasetsRoute(t *testing.T) {
	expectedUserId := 123
	for tName, expectedQueryParams := range map[string]queryParamMap{
		"without any params":  {},
		"with limit param":    {"limit": "30"},
		"with offset param":   {"offset": "10"},
		"with both params":    {"limit": "50", "offset": "20"},
	} {
		req := newTestRequest("GET",
			"/shared-datasets",
			"getSharedDatasetsRequestID",
			expectedQueryParams,
			"")
		mockService := new(MockCrossWorkspaceDatasetsService)

		claims := authorizer.Claims{
			UserClaim: &user.Claim{
				Id: int64(expectedUserId),
			}}

		expectedLimit := expectedQueryParams.expectedLimit(t)
		expectedOffset := expectedQueryParams.expectedOffset(t)
		expectedPage := &models.SharedDatasetsPage{
			Limit:      expectedLimit,
			Offset:     expectedOffset,
			TotalCount: 2,
			Datasets: []models.SharedDatasetItem{
				{
					Content: models.SharedDatasetContent{
						ID:    "N:dataset:1",
						Name:  "Test Dataset 1",
						State: "READY",
					},
				},
				{
					Content: models.SharedDatasetContent{
						ID:    "N:dataset:2",
						Name:  "Test Dataset 2", 
						State: "READY",
					},
				},
			},
		}
		mockService.OnGetSharedDatasetsPageReturn(expectedUserId, expectedLimit, expectedOffset, expectedPage)
		
		handler := NewHandler(req, &claims)
		sharedHandler := &SharedDatasetsHandler{RequestHandler: *handler}
		sharedHandler.crossWorkspaceDatasetsService = mockService
		
		t.Run(tName, func(t *testing.T) {
			resp, err := sharedHandler.handle(context.Background())
			if assert.NoError(t, err) {
				mockService.AssertExpectations(t)
				assert.Equal(t, http.StatusOK, resp.StatusCode)
				assert.Contains(t, resp.Body, "Test Dataset 1")
				assert.Contains(t, resp.Body, "Test Dataset 2")
			}
		})
	}
}

func TestSharedDatasetsRouteUnauthorized(t *testing.T) {
	tests := []struct {
		name   string
		claims *authorizer.Claims
	}{
		{
			name:   "nil claims",
			claims: nil,
		},
		{
			name:   "nil user claim",
			claims: &authorizer.Claims{UserClaim: nil},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newTestRequest("GET", "/shared-datasets", "testRequestID", map[string]string{}, "")
			mockService := new(MockCrossWorkspaceDatasetsService)
			
			handler := NewHandler(req, tt.claims)
			sharedHandler := &SharedDatasetsHandler{RequestHandler: *handler}
			sharedHandler.crossWorkspaceDatasetsService = mockService
			
			resp, err := sharedHandler.handle(context.Background())
			assert.NoError(t, err)
			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
			assert.Contains(t, resp.Body, "unauthorized")
			
			// Should not call service when unauthorized
			mockService.AssertNotCalled(t, "GetSharedDatasetsPage")
		})
	}
}

func TestSharedDatasetsRouteHandledErrors(t *testing.T) {
	userId := 123
	for tName, tData := range map[string]struct {
		QueryParams         queryParamMap
		ServiceError        error
		ExpectedStatus      int
		ExpectedSubMessages []string
	}{
		"with too low limit": {
			QueryParams:         queryParamMap{"limit": "-1"},
			ExpectedStatus:      http.StatusBadRequest,
			ExpectedSubMessages: []string{"min value", "limit"},
		},
		"with too high limit": {
			QueryParams:         queryParamMap{"limit": "50000"},
			ExpectedStatus:      http.StatusBadRequest,
			ExpectedSubMessages: []string{"max value", "limit"},
		},
		"with too low offset": {
			QueryParams:         queryParamMap{"offset": "-4"},
			ExpectedStatus:      http.StatusBadRequest,
			ExpectedSubMessages: []string{"min value", "offset"},
		},
		"with non-numeric limit": {
			QueryParams:         queryParamMap{"limit": "abc"},
			ExpectedStatus:      http.StatusBadRequest,
			ExpectedSubMessages: []string{"strconv.Atoi", "abc"},
		},
		"with non-numeric offset": {
			QueryParams:         queryParamMap{"offset": "xyz"},
			ExpectedStatus:      http.StatusBadRequest,
			ExpectedSubMessages: []string{"strconv.Atoi", "xyz"},
		},
	} {
		req := newTestRequest("GET",
			"/shared-datasets",
			"getSharedDatasetsRequestID",
			tData.QueryParams,
			"")
		mockService := new(MockCrossWorkspaceDatasetsService)
		
		claims := authorizer.Claims{
			UserClaim: &user.Claim{
				Id: int64(userId),
			}}

		// Only set up service mock if we expect it to be called (i.e., no query param validation error)
		if tData.ServiceError != nil {
			mockService.OnGetSharedDatasetsPageFail(
				userId, 
				tData.QueryParams.expectedLimit(t), 
				tData.QueryParams.expectedOffset(t),
				tData.ServiceError,
			)
		}

		handler := NewHandler(req, &claims)
		sharedHandler := &SharedDatasetsHandler{RequestHandler: *handler}
		sharedHandler.crossWorkspaceDatasetsService = mockService
		
		t.Run(tName, func(t *testing.T) {
			resp, err := sharedHandler.handle(context.Background())
			if assert.NoError(t, err) {
				assert.Equal(t, tData.ExpectedStatus, resp.StatusCode)
				for _, messageFragment := range tData.ExpectedSubMessages {
					assert.Contains(t, resp.Body, messageFragment)
				}
			}
		})
	}
}

func TestSharedDatasetsRouteMethodNotAllowed(t *testing.T) {
	req := newTestRequest("POST", "/shared-datasets", "testRequestID", map[string]string{}, "")
	mockService := new(MockCrossWorkspaceDatasetsService)
	
	claims := authorizer.Claims{
		UserClaim: &user.Claim{
			Id: 123,
		}}

	handler := NewHandler(req, &claims)
	sharedHandler := &SharedDatasetsHandler{RequestHandler: *handler}
	sharedHandler.crossWorkspaceDatasetsService = mockService
	
	resp, err := sharedHandler.handle(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	assert.Contains(t, resp.Body, "method not allowed")
	
	// Should not call service for unsupported method
	mockService.AssertNotCalled(t, "GetSharedDatasetsPage")
}

func TestSharedDatasetsRouteNoCrossWorkspaceService(t *testing.T) {
	req := newTestRequest("GET", "/shared-datasets", "testRequestID", map[string]string{}, "")
	
	claims := authorizer.Claims{
		UserClaim: &user.Claim{
			Id: 123,
		}}

	handler := NewHandler(req, &claims)
	sharedHandler := &SharedDatasetsHandler{RequestHandler: *handler}
	// Note: not setting crossWorkspaceDatasetsService

	resp, err := sharedHandler.handle(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.Contains(t, resp.Body, "cross-workspace service not configured")
}