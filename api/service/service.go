package service

import (
	"github.com/pennsieve/datasets-service/api/models"
	"github.com/pennsieve/datasets-service/api/store"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/packageInfo/packageState"
)

type DatasetsService interface {
	GetTrashcanPage(datasetID string, limit int, offset int) (*models.TrashcanPage, error)
}

type DatasetsServiceImpl struct {
	Store *store.DatasetsStore
}

func NewDatasetsService(store *store.DatasetsStore) *DatasetsServiceImpl {
	return &DatasetsServiceImpl{Store: store}
}

func (s *DatasetsServiceImpl) GetTrashcanPage(datasetId string, limit int, offset int) (*models.TrashcanPage, error) {
	var trashcan models.TrashcanPage
	dsIntId, err := s.Store.GetDatasetByNodeId(datasetId)
	if err != nil {
		return &trashcan, err
	}
	page, err := s.Store.GetDatasetPackagesByState(dsIntId, packageState.Deleted, limit, offset)
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
