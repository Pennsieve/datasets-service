package service

import (
	"github.com/pennsieve/datasets-service/api/models"
	"github.com/pennsieve/datasets-service/api/store"
)

type DatasetsService interface {
	GetTrashcan(datasetID string, limit int, offset int) (*models.TrashcanPage, error)
}

type DatasetsServiceImpl struct {
	Store *store.DatasetsStore
}

func NewDatasetsService(store *store.DatasetsStore) *DatasetsServiceImpl {
	return &DatasetsServiceImpl{Store: store}
}

func (s *DatasetsServiceImpl) GetTrashcan(datasetId string, limit int, offset int) (*models.TrashcanPage, error) {
	return &models.TrashcanPage{Messages: []string{"GetTrashcan not implemented yet"}}, nil
}
