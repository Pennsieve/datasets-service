package service

import (
	"github.com/pennsieve/datasets-service/api/models"
	"github.com/pennsieve/datasets-service/api/store"
)

type DatasetsService struct {
	Store *store.DatasetsStore
}

func NewDatasetsService(store *store.DatasetsStore) *DatasetsService {
	return &DatasetsService{Store: store}
}

func (s *DatasetsService) GetTrashcan(datasetId string, limit int, offset int) (models.TrashcanPage, error) {
	return models.TrashcanPage{Messages: []string{"GetTrashcan not implemented yet"}}, nil
}
