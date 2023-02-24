package service

import (
	"context"
	"github.com/pennsieve/datasets-service/api/models"
	"github.com/pennsieve/datasets-service/api/store"
)

type DatasetsService interface {
	GetTrashcanPage(ctx context.Context, datasetID string, rootNodeId string, limit int, offset int) (*models.TrashcanPage, error)
}

type DatasetsServiceImpl struct {
	Store *store.DatasetsStore
}

func NewDatasetsService(store *store.DatasetsStore) *DatasetsServiceImpl {
	return &DatasetsServiceImpl{Store: store}
}

func (s *DatasetsServiceImpl) GetTrashcanPage(ctx context.Context, datasetId string, rootNodeId string, limit int, offset int) (*models.TrashcanPage, error) {
	var trashcan models.TrashcanPage
	dataset, err := s.Store.GetDatasetByNodeId(ctx, datasetId)
	if err != nil {
		return &trashcan, err
	}
	var page *store.PackagePage
	var tErr error
	if len(rootNodeId) == 0 {
		page, tErr = s.Store.GetTrashcanRootPaginated(ctx, dataset.Id, limit, offset)
	} else {
		page, tErr = s.Store.GetTrashcanPaginated(ctx, dataset.Id, rootNodeId, limit, offset)
	}
	if tErr != nil {
		return &trashcan, err
	}
	packages := make([]models.TrashcanItem, len(page.Packages))
	for i, p := range page.Packages {
		packages[i] = models.TrashcanItem{
			ID:     p.Id,
			Name:   p.Name,
			NodeId: p.NodeId,
			Type:   p.PackageType.String(),
			State:  p.PackageState.String(),
		}
	}
	return &models.TrashcanPage{
		Limit:      limit,
		Offset:     offset,
		TotalCount: page.TotalCount,
		Packages:   packages,
		Messages:   []string{}}, nil
}
