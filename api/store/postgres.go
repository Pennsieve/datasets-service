package store

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
)

type DatasetsStore struct {
	DB *sql.DB
}

func (d *DatasetsStore) ListFiles(datasetId string, limit int, offset int) ([]string, error) {
	return nil, fmt.Errorf("GetFiles() not implemented")
}

func NewDatasetsStore(pennsieveDB *sql.DB) *DatasetsStore {
	return &DatasetsStore{pennsieveDB}
}

func NewDatasetStoreAtOrg(pennsieveDB *sql.DB, orgID int64) (*DatasetsStore, error) {
	_, err := pennsieveDB.Exec(fmt.Sprintf("SET search_path = %q;", orgID))
	if err != nil {
		return nil, err
	}
	return &DatasetsStore{pennsieveDB}, nil
}
