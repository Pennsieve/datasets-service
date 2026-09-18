package store

import (
    "context"
    "database/sql"
    "log/slog"

    "github.com/pennsieve/datasets-service/api/models"
)

// CrossOrgStore provides methods for queries that span multiple organization schemas
type CrossOrgStore interface {
    GetSharedDatasetsForUser(ctx context.Context, userId int, limit int, offset int) (*models.SharedDatasetsPage, error)
}

// CrossOrgStoreFactory creates CrossOrgStore instances
type CrossOrgStoreFactory interface {
    NewCrossOrgStore() CrossOrgStore
}

// crossOrgStoreFactory implements CrossOrgStoreFactory
type crossOrgStoreFactory struct {
    DB     *sql.DB
    Logger *slog.Logger
}

// NewCrossOrgStoreFactory creates a new factory for cross-org stores. Takes the
// request-scoped logger so DB failures carry the invocation's trace/request
// context; pass slog.Default() outside of a request.
func NewCrossOrgStoreFactory(pennsieveDB *sql.DB, logger *slog.Logger) CrossOrgStoreFactory {
    if logger == nil {
        logger = slog.Default()
    }
    return &crossOrgStoreFactory{DB: pennsieveDB, Logger: logger}
}

// NewCrossOrgStore returns a CrossOrgStore instance
func (f *crossOrgStoreFactory) NewCrossOrgStore() CrossOrgStore {
    // Use the simple implementation that doesn't require PostgreSQL functions
    return NewCrossOrgQueriesSimple(f.DB, f.Logger)
}
