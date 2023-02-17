package service

import (
	"fmt"
	"github.com/pennsieve/datasets-service/api/store"
)

type DatasetsService interface {
	GetTrashcan(datasetId string) error
}

type datasetsService struct {
	Store store.DatasetsStore
}

func NewDatasetsService(store store.DatasetsStore) DatasetsService {
	return &datasetsService{Store: store}
}

func (s *datasetsService) GetTrashcan(datasetId string) error {
	if s.Store != nil {
		return s.Store.ListFiles(datasetId)
	}
	return fmt.Errorf("illegal state: no DatasetsStore")
}
