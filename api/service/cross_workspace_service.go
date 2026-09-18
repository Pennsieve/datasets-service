package service

import (
	"context"
	"database/sql"
	"github.com/pennsieve/datasets-service/api/models"
	"github.com/pennsieve/datasets-service/api/store"
	"log/slog"
)

// CrossWorkspaceDatasetsService provides methods for operations that span multiple workspaces
type CrossWorkspaceDatasetsService interface {
	GetSharedDatasetsPage(ctx context.Context, userId int, limit int, offset int) (*models.SharedDatasetsPage, error)
}

// crossWorkspaceDatasetsService implements CrossWorkspaceDatasetsService
type crossWorkspaceDatasetsService struct {
	CrossOrgStoreFactory store.CrossOrgStoreFactory
	Logger               *slog.Logger
}

// NewCrossWorkspaceDatasetsService creates a new service for cross-workspace
// operations. Takes the request-scoped logger so the cross-org store's failures
// carry the invocation's trace/request context; a nil logger falls back to
// slog.Default().
func NewCrossWorkspaceDatasetsService(db *sql.DB, logger *slog.Logger) CrossWorkspaceDatasetsService {
	if logger == nil {
		logger = slog.Default()
	}
	crossOrgFactory := store.NewCrossOrgStoreFactory(db, logger)

	return &crossWorkspaceDatasetsService{
		CrossOrgStoreFactory: crossOrgFactory,
		Logger:               logger,
	}
}

// GetSharedDatasetsPage returns a paginated list of datasets shared with the user
// from workspaces where the user is not a contributor within the workspace.
func (s *crossWorkspaceDatasetsService) GetSharedDatasetsPage(ctx context.Context, userId int, limit int, offset int) (*models.SharedDatasetsPage, error) {
	// Use the cross-org store to fetch shared datasets
	crossOrgStore := s.CrossOrgStoreFactory.NewCrossOrgStore()
	return crossOrgStore.GetSharedDatasetsForUser(ctx, userId, limit, offset)
}