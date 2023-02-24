package service

import (
	"context"
	"github.com/pennsieve/datasets-service/api/models"
	"github.com/pennsieve/datasets-service/api/store"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/packageInfo/packageState"
)

type DatasetsService interface {
	GetTrashcanPage(ctx context.Context, datasetID string, limit int, offset int) (*models.TrashcanPage, error)
}

type DatasetsServiceImpl struct {
	Store *store.DatasetsStore
}

func NewDatasetsService(store *store.DatasetsStore) *DatasetsServiceImpl {
	return &DatasetsServiceImpl{Store: store}
}

func (s *DatasetsServiceImpl) GetTrashcanPage(ctx context.Context, datasetId string, limit int, offset int) (*models.TrashcanPage, error) {
	var trashcan models.TrashcanPage
	dataset, err := s.Store.GetDatasetByNodeId(ctx, datasetId)
	if err != nil {
		return &trashcan, err
	}
	page, err := s.Store.GetDatasetPackagesByState(ctx, dataset.Id, packageState.Deleted, limit, offset)
	if err != nil {
		return &trashcan, err
	}
	packages := make([]models.TrashcanItem, len(page.Packages))
	for i, p := range page.Packages {
		packages[i] = models.TrashcanItem{
			ID:   p.NodeId,
			Name: p.Name,
			//TODO get path from dynamodb
		}
	}
	return &models.TrashcanPage{
		Limit:      limit,
		Offset:     offset,
		TotalCount: page.TotalCount,
		Packages:   packages,
		Messages:   []string{}}, nil
}
