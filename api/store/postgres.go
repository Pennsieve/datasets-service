package store

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/pennsieve/datasets-service/api/models"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/dbTable"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/packageInfo/packageState"
	"strconv"
	"strings"
)

var (
	packagesColumns            = []string{"id", "name", "type", "state", "node_id", "parent_id", "dataset_id", "owner_id", "size", "import_id", "attributes", "created_at", "updated_at"}
	packageColumnsString       = strings.Join(packagesColumns, ", ")
	getTrashcanPageQueryFormat = `WITH RECURSIVE trash(id, node_id, type, parent_id, name, state, id_path) AS
                                  (
									SELECT id, node_id, type, %s, name, state, ARRAY [id]
									FROM packages
									WHERE parent_id %s
									AND dataset_id = $1
                                  UNION ALL
									SELECT p.id, p.node_id, p.type, p.parent_id, p.name, p.state, id_path || p.id
									FROM packages p
									JOIN trash t ON t.id = p.parent_id
									WHERE t.state <> 'DELETED'
                                  )
                                  SELECT %s, COUNT(*) OVER() as total_count
                                  FROM trash t JOIN packages p ON t.id = p.id
                                  WHERE t.parent_id %s
  					              AND EXISTS(SELECT 1 from trash t2 where t2.state = 'DELETED' and t.id = ANY(t2.id_path))
					              ORDER BY t.name, t.id
					              LIMIT $2 OFFSET $3;`
	getTrashcanRootPageQuery = fmt.Sprintf(getTrashcanPageQueryFormat, "null::integer", "is null", qualifiedColumns("p", packagesColumns), "is null")
)

type PackagePage struct {
	TotalCount int
	Packages   []dbTable.Package
}

type DatasetsStore struct {
	DB    *sql.DB
	OrgId int
}

func (d *DatasetsStore) GetDatasetByNodeId(ctx context.Context, dsNodeId string) (*dbTable.Dataset, error) {
	const datasetColumns = "id, name, state, description, updated_at, created_at, node_id, permission_bit, type, role, status, automatically_process_packages, license, tags, contributors, banner_id, readme_id, status_id, publication_status_id, size, etag, data_use_agreement_id, changelog_id"
	var ds dbTable.Dataset
	query := fmt.Sprintf("SELECT %s FROM datasets WHERE node_id = $1", datasetColumns)
	if err := d.DB.QueryRowContext(ctx, query, dsNodeId).Scan(
		&ds.Id,
		&ds.Name,
		&ds.State,
		&ds.Description,
		&ds.UpdatedAt,
		&ds.CreatedAt,
		&ds.NodeId,
		&ds.PermissionBit,
		&ds.Type,
		&ds.Role,
		&ds.Status,
		&ds.AutomaticallyProcessPackages,
		&ds.License,
		&ds.Tags,
		&ds.Contributors,
		&ds.BannerId,
		&ds.ReadmeId,
		&ds.StatusId,
		&ds.PublicationStatusId,
		&ds.Size,
		&ds.ETag,
		&ds.DataUseAgreementId,
		&ds.ChangelogId); err == sql.ErrNoRows {
		return &ds, models.DatasetNotFoundError{NodeId: dsNodeId, OrgId: d.OrgId}
	} else {
		return &ds, err
	}
}

func (d *DatasetsStore) GetDatasetPackagesByState(ctx context.Context, datasetId int64, state packageState.State, limit int, offset int) (*PackagePage, error) {
	query := fmt.Sprintf("SELECT %s, COUNT(*) OVER() as total_count FROM packages WHERE state = $1 and dataset_id = $2 ORDER BY id LIMIT $3 OFFSET $4", packageColumnsString)
	rows, err := d.DB.QueryContext(ctx, query,
		state, datasetId, limit, offset)
	var page PackagePage
	if err != nil {
		return &page, err
	}
	defer rows.Close()

	var totalCount int
	packages := make([]dbTable.Package, limit)
	i := 0
	for rows.Next() {
		p := &packages[i]
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
			&p.Attributes,
			&p.CreatedAt,
			&p.UpdatedAt,
			&totalCount); err != nil {
			return &page, err
		}
		i++
	}
	if err := rows.Err(); err != nil {
		return &page, err
	}
	page.TotalCount = totalCount
	page.Packages = packages[:i]

	return &page, nil
}

func (d *DatasetsStore) queryTrashcan(ctx context.Context, query string, datasetId int64, limit int, offset int) (*PackagePage, error) {
	rows, err := d.DB.QueryContext(ctx, query, datasetId, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var page PackagePage
	var totalCount int
	packages := make([]dbTable.Package, limit)
	i := 0
	for rows.Next() {
		p := &packages[i]
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
			&p.Attributes,
			&p.CreatedAt,
			&p.UpdatedAt,
			&totalCount); err != nil {
			return &page, err
		}
		i++
	}
	if err := rows.Err(); err != nil {
		return &page, err
	}
	page.TotalCount = totalCount
	page.Packages = packages[:i]

	return &page, nil
}

func (d *DatasetsStore) GetTrashcanRootPaginated(ctx context.Context, datasetId int64, limit int, offset int) (*PackagePage, error) {
	return d.queryTrashcan(ctx, getTrashcanRootPageQuery, datasetId, limit, offset)
}

func (d *DatasetsStore) GetTrashcanPaginated(ctx context.Context, datasetId int64, parentNodeId string, limit int, offset int) (*PackagePage, error) {
	var parentId int
	if err := d.DB.QueryRowContext(ctx, "SELECT id from packages where node_id = ? ", parentNodeId).Scan(&parentId); err != nil {
		if err == sql.ErrNoRows {
			return nil, models.PackageNotFoundError{NodeId: parentNodeId, OrgId: d.OrgId}
		}
		return nil, err

	}
	pIdStr := strconv.Itoa(parentId)
	equalPIdStr := fmt.Sprintf("= %d", parentId)
	query := fmt.Sprintf(getTrashcanPageQueryFormat, pIdStr, equalPIdStr, qualifiedColumns("p", packagesColumns), equalPIdStr)
	return d.queryTrashcan(ctx, query, datasetId, limit, offset)
}

func NewDatasetsStore(pennsieveDB *sql.DB) *DatasetsStore {
	return &DatasetsStore{DB: pennsieveDB}
}

func NewDatasetStoreAtOrg(pennsieveDB *sql.DB, orgID int) (*DatasetsStore, error) {
	_, err := pennsieveDB.Exec(fmt.Sprintf("SET search_path = %d;", orgID))
	if err != nil {
		return nil, err
	}
	return &DatasetsStore{DB: pennsieveDB, OrgId: orgID}, nil
}

func qualifiedColumns(table string, columns []string) string {
	q := make([]string, len(columns))
	for i, c := range columns {
		q[i] = fmt.Sprintf("%s.%s", table, c)
	}
	return strings.Join(q, ", ")
}
