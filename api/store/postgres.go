package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/pennsieve/datasets-service/api/models"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/packageInfo/packageState"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/pgdb"
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
	Packages   []pgdb.Package
}

type DatasetsStoreImpl struct {
	DB    *sql.DB
	OrgId int
}

func (d *DatasetsStoreImpl) GetOrgId(_ context.Context) int {
	return d.OrgId
}

func (d *DatasetsStoreImpl) GetDatasetByNodeId(ctx context.Context, dsNodeId string) (*pgdb.Dataset, error) {
	const datasetColumns = "id, name, state, description, updated_at, created_at, node_id, permission_bit, type, role, status, automatically_process_packages, license, tags, contributors, banner_id, readme_id, status_id, publication_status_id, size, etag, data_use_agreement_id, changelog_id"
	var ds pgdb.Dataset
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
		&ds.ChangelogId); errors.Is(err, sql.ErrNoRows) {
		return &ds, models.DatasetNotFoundError{Id: models.DatasetNodeId(dsNodeId), OrgId: d.OrgId}
	} else {
		return &ds, err
	}
}

func (d *DatasetsStoreImpl) CountDatasetPackagesByState(ctx context.Context, datasetId int64, state packageState.State) (int, error) {
	var count int
	err := d.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM packages where dataset_id = $1 and state = $2",
		datasetId,
		state).Scan(&count)
	return count, err
}

func (d *DatasetsStoreImpl) GetDatasetPackageByNodeId(ctx context.Context, datasetId int64, packageNodeId string) (*pgdb.Package, error) {
	var pckg pgdb.Package
	queryStr := fmt.Sprintf("SELECT %s FROM packages where dataset_id = $1 and node_id = $2", packageColumnsString)
	if err := d.DB.QueryRowContext(ctx, queryStr, datasetId, packageNodeId).Scan(&pckg); errors.Is(err, sql.ErrNoRows) {
		return &pckg, models.PackageNotFoundError{Id: models.PackageNodeId(packageNodeId), OrgId: d.OrgId, DatasetId: models.DatasetIntId(datasetId)}
	} else {
		return &pckg, err
	}
}

func (d *DatasetsStoreImpl) queryTrashcan(ctx context.Context, query string, datasetId int64, limit int, offset int) (*PackagePage, error) {
	rows, err := d.DB.QueryContext(ctx, query, datasetId, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var page PackagePage
	var totalCount int
	packages := make([]pgdb.Package, limit)
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

func (d *DatasetsStoreImpl) GetTrashcanRootPaginated(ctx context.Context, datasetId int64, limit int, offset int) (*PackagePage, error) {
	return d.queryTrashcan(ctx, getTrashcanRootPageQuery, datasetId, limit, offset)
}

func (d *DatasetsStoreImpl) GetTrashcanPaginated(ctx context.Context, datasetId int64, parentId int64, limit int, offset int) (*PackagePage, error) {
	pIdStr := strconv.FormatInt(parentId, 10)
	equalPIdStr := fmt.Sprintf("= %d", parentId)
	query := fmt.Sprintf(getTrashcanPageQueryFormat, pIdStr, equalPIdStr, qualifiedColumns("p", packagesColumns), equalPIdStr)
	return d.queryTrashcan(ctx, query, datasetId, limit, offset)
}

func NewDatasetsStore(pennsieveDB *sql.DB) *DatasetsStoreImpl {
	return &DatasetsStoreImpl{DB: pennsieveDB}
}

func NewDatasetStoreAtOrg(pennsieveDB *sql.DB, orgID int) (*DatasetsStoreImpl, error) {
	_, err := pennsieveDB.Exec(fmt.Sprintf("SET search_path = %d;", orgID))
	if err != nil {
		return nil, err
	}
	return &DatasetsStoreImpl{DB: pennsieveDB, OrgId: orgID}, nil
}

func qualifiedColumns(table string, columns []string) string {
	q := make([]string, len(columns))
	for i, c := range columns {
		q[i] = fmt.Sprintf("%s.%s", table, c)
	}
	return strings.Join(q, ", ")
}

type DatasetsStore interface {
	GetDatasetByNodeId(ctx context.Context, dsNodeId string) (*pgdb.Dataset, error)
	GetTrashcanRootPaginated(ctx context.Context, datasetId int64, limit int, offset int) (*PackagePage, error)
	GetTrashcanPaginated(ctx context.Context, datasetId int64, parentId int64, limit int, offset int) (*PackagePage, error)
	CountDatasetPackagesByState(ctx context.Context, datasetId int64, state packageState.State) (int, error)
	GetDatasetPackageByNodeId(ctx context.Context, datasetId int64, packageNodeId string) (*pgdb.Package, error)
	GetOrgId(ctx context.Context) int
}
