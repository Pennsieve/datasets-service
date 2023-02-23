package store

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/dbTable"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/packageInfo"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/packageInfo/packageState"
)

type PackageAttributes []packageInfo.PackageAttribute

func (a *PackageAttributes) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func (a *PackageAttributes) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &a)
}

type PackagePage struct {
	TotalCount int
	Packages   []dbTable.Package
}

type DatasetsStore struct {
	DB *sql.DB
}

func (d *DatasetsStore) GetDatasetPackagesByState(datasetId int, state packageState.State, limit int, offset int) (PackagePage, error) {
	const packagesColumns = "id, name, type, state, node_id, parent_id, dataset_id, owner_id, size, import_id, attributes, created_at, updated_at"
	query := fmt.Sprintf("SELECT %s, COUNT(*) OVER() as total_count FROM packages WHERE state = $1 and dataset_id = $2 ORDER BY id LIMIT $3 OFFSET $4", packagesColumns)
	rows, err := d.DB.Query(query,
		state, datasetId, limit, offset)
	if err != nil {
		return PackagePage{}, err
	}
	defer rows.Close()

	var totalCount int
	var packages []dbTable.Package
	for rows.Next() {
		var p dbTable.Package
		var a PackageAttributes
		if err := rows.Scan(
			&p.Id,
			&p.Name,
			&p.PackageType,
			&p.PackageState,
			&p.NodeId,
			&p.ParentId,
			&p.DatasetId,
			&p.OwnerId,
			&p.Size,
			&p.ImportId,
			&a,
			&p.CreatedAt,
			&p.UpdatedAt,
			&totalCount); err != nil {
			return PackagePage{}, err
		}
		p.Attributes = a
		packages = append(packages, p)
	}
	if err := rows.Err(); err != nil {
		return PackagePage{}, err
	}

	return PackagePage{TotalCount: totalCount, Packages: packages}, nil
}

func NewDatasetsStore(pennsieveDB *sql.DB) *DatasetsStore {
	return &DatasetsStore{pennsieveDB}
}

func NewDatasetStoreAtOrg(pennsieveDB *sql.DB, orgID int64) (*DatasetsStore, error) {
	_, err := pennsieveDB.Exec(fmt.Sprintf("SET search_path = %d;", orgID))
	if err != nil {
		return nil, err
	}
	return &DatasetsStore{pennsieveDB}, nil
}
