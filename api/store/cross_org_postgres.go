package store

import (
    "context"
    "database/sql"
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
    DB *sql.DB
}

// NewCrossOrgStoreFactory creates a new factory for cross-org stores
func NewCrossOrgStoreFactory(pennsieveDB *sql.DB) CrossOrgStoreFactory {
    return &crossOrgStoreFactory{DB: pennsieveDB}
}

// NewCrossOrgStore returns a CrossOrgStore instance
func (f *crossOrgStoreFactory) NewCrossOrgStore() CrossOrgStore {
    // Use the simple implementation that doesn't require PostgreSQL functions
    return NewCrossOrgQueriesSimple(f.DB)
}
