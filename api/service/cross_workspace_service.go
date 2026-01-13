package service

import (
	"context"
	"database/sql"
	"github.com/pennsieve/datasets-service/api/models"
	"github.com/pennsieve/datasets-service/api/store"
)

// CrossWorkspaceDatasetsService provides methods for operations that span multiple workspaces
type CrossWorkspaceDatasetsService interface {
	GetSharedDatasetsPage(ctx context.Context, userId int, limit int, offset int) (*models.SharedDatasetsPage, error)
}

// crossWorkspaceDatasetsService implements CrossWorkspaceDatasetsService
type crossWorkspaceDatasetsService struct {
	CrossOrgStoreFactory store.CrossOrgStoreFactory
}

// NewCrossWorkspaceDatasetsService creates a new service for cross-workspace operations
func NewCrossWorkspaceDatasetsService(db *sql.DB) CrossWorkspaceDatasetsService {
	crossOrgFactory := store.NewCrossOrgStoreFactory(db)
	
	return &crossWorkspaceDatasetsService{
		CrossOrgStoreFactory: crossOrgFactory,
	}
}

// GetSharedDatasetsPage returns a paginated list of datasets shared with the user
// from workspaces where the user is not a contributor within the workspace.
func (s *crossWorkspaceDatasetsService) GetSharedDatasetsPage(ctx context.Context, userId int, limit int, offset int) (*models.SharedDatasetsPage, error) {
	// Use the cross-org store to fetch shared datasets
	crossOrgStore := s.CrossOrgStoreFactory.NewCrossOrgStore()
	return crossOrgStore.GetSharedDatasetsForUser(ctx, userId, limit, offset)
}