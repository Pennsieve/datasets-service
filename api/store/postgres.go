package store

import (
	"database/sql"
	_ "github.com/lib/pq"
)

type DatasetsStore interface {
	ListFiles(datasetId string) error
}

type datasetsStore struct {
	DB *sql.DB
}

func (d *datasetsStore) ListFiles(datasetId string) error {
	return nil
}

func NewDatasetsStore(pennsieveDB *sql.DB) DatasetsStore {
	return &datasetsStore{pennsieveDB}
}
