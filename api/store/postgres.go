package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"github.com/pennsieve/datasets-service/api/models"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/packageInfo/packageState"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/pgdb"
	pg "github.com/pennsieve/pennsieve-go-core/pkg/queries/pgdb"
	log "github.com/sirupsen/logrus"
	"strconv"
	"strings"
)

var (
	packagesColumns            = []string{"id", "name", "type", "state", "node_id", "parent_id", "dataset_id", "owner_id", "size", "import_id", "attributes", "created_at", "updated_at"}
	packageColumnsString       = strings.Join(packagesColumns, ", ")
	getTrashcanPageQueryFormat = `WITH RECURSIVE trash(id, node_id, type, parent_id, name, state, id_path) AS
                                  (
									SELECT id, node_id, type, %[1]s, name, state, ARRAY [id]
									FROM "%[4]d".packages
									WHERE parent_id %[2]s
									AND dataset_id = $1
                                  UNION ALL
									SELECT p.id, p.node_id, p.type, p.parent_id, p.name, p.state, id_path || p.id
									FROM "%[4]d".packages p
									JOIN trash t ON t.id = p.parent_id
									WHERE t.state <> 'DELETED' AND t.state <> 'DELETING'
                                  )
                                  SELECT %[3]s, COUNT(*) OVER() as total_count
                                  FROM trash t JOIN "%[4]d".packages p ON t.id = p.id
                                  WHERE t.parent_id %[2]s
  					              AND EXISTS(SELECT 1 FROM trash t2 WHERE (t2.state = 'DELETED' OR t2.state = 'DELETING') and t.id = ANY(t2.id_path))
					              ORDER BY t.name, t.id
					              LIMIT $2 OFFSET $3;`
	getManifestQueryFormat = `WITH RECURSIVE parents (dataset_id, state, id, name, parent_id, node_id, path) AS
		                      (
		                         SELECT p.dataset_id, p.state, p.id, p.name,  p.parent_id, p.node_id,  array[parent_id]
		                             FROM "%[1]d".packages p
		                         WHERE p.dataset_id = %[2]d AND p.parent_id IS NULL AND p.state NOT IN ('DELETING', 'DELETED')
		                      UNION
		                         SELECT children.dataset_id, children.state, children.id, children.name,  children.parent_id, children.node_id, path || children.parent_id
		                         FROM "%[1]d".packages children
		                         INNER JOIN parents ON
		                            parents.id = children.parent_id
		                         WHERE children.state NOT IN ('DELETING', 'DELETED') AND parents.node_id LIKE 'N:collection:%%'
							  )
		                      SELECT parents.id AS package_id, parents.name AS package_name, f.name, path, node_id, f.size, f.checksum
		                      FROM parents
		                      LEFT JOIN "%[1]d".files f ON parents.id = f.package_id`

	//getManifestQueryFormatOld = `WITH RECURSIVE parents (dataset_id, state, id, name, file_name, parent_id, node_id, checksum, size, path) AS
	//							(
	//								SELECT p.dataset_id, p.state, p.id, p.name, f.name, p.parent_id, p.node_id, f.checksum, f.size, array[parent_id]
	//							    FROM "%[1]d".packages p
	//								FULL JOIN "%[1]d".files f ON p.id = f.package_id
	//								WHERE p.dataset_id = %[2]d AND p.parent_id IS NULL AND p.state NOT IN ('DELETING', 'DELETED')
	//							UNION
	//								SELECT children.dataset_id, children.state, children.id, children.name, files.name, children.parent_id, children.node_id, files.checksum, files.size, path || children.parent_id
	//								FROM "%[1]d".packages children
	//								FULL JOIN "%[1]d".files ON children.id = files.package_id
	//								INNER JOIN parents ON
	//									parents.id = children.parent_id
	//								WHERE children.state NOT IN ('DELETING', 'DELETED')
	//							)
	//							SELECT id AS package_id, name AS package_name, file_name, path, node_id, size, checksum
	//							FROM parents`
)

type PackagePage struct {
	TotalCount int
	Packages   []pgdb.Package
}

type DatasetsStoreFactory interface {
	NewSimpleStore(orgId int) DatasetsStore
	ExecStoreTx(ctx context.Context, orgId int, fn func(store DatasetsStore) error) error
}

func NewPostgresStoreFactory(pennsieveDB *sql.DB) DatasetsStoreFactory {
	return &datasetsStoreFactory{DB: pennsieveDB}
}

type datasetsStoreFactory struct {
	DB       *sql.DB
	S3Client *s3.Client
}

// NewSimpleStore returns a DatasetsStore instance that
// will run statements directly on database
func (d *datasetsStoreFactory) NewSimpleStore(orgId int) DatasetsStore {
	return NewQueries(d.DB, orgId)
}

// ExecStoreTx will execute the function fn, passing in a new DatasetsStore instance that
// is backed by a database transaction. Any methods fn runs against the passed in DatasetsStore will run
// in this transaction. If fn returns a non-nil error, the transaction will be rolled back.
// Otherwise, the transaction will be committed.
func (d *datasetsStoreFactory) ExecStoreTx(ctx context.Context, orgId int, fn func(DatasetsStore) error) error {
	tx, err := d.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	q := NewQueries(tx, orgId)
	err = fn(q)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit()
}

type Queries struct {
	db    pg.DBTX
	OrgId int
}

func NewQueries(db pg.DBTX, orgId int) *Queries {
	return &Queries{db: db, OrgId: orgId}
}

func (q *Queries) GetDatasetByNodeId(ctx context.Context, dsNodeId string) (*pgdb.Dataset, error) {
	const datasetColumns = "id, name, state, description, updated_at, created_at, node_id, permission_bit, type, role, status, automatically_process_packages, license, tags, contributors, banner_id, readme_id, status_id, publication_status_id, size, etag, data_use_agreement_id, changelog_id"
	var ds pgdb.Dataset
	query := fmt.Sprintf(`SELECT %s FROM "%d".datasets WHERE node_id = $1`, datasetColumns, q.OrgId)
	if err := q.db.QueryRowContext(ctx, query, dsNodeId).Scan(
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
		return &ds, models.DatasetNotFoundError{Id: models.DatasetNodeId(dsNodeId), OrgId: q.OrgId}
	} else {
		return &ds, err
	}
}

func (q *Queries) CountDatasetPackagesByStates(ctx context.Context, datasetId int64, states []packageState.State) (int, error) {
	query := fmt.Sprintf(`SELECT COUNT(*) FROM "%d".packages WHERE dataset_id = $1 AND state = ANY($2)`, q.OrgId)
	var count int
	err := q.db.QueryRowContext(ctx, query, datasetId, pq.Array(states)).Scan(&count)
	return count, err
}

func (q *Queries) GetDatasetPackageByNodeId(ctx context.Context, datasetId int64, packageNodeId string) (*pgdb.Package, error) {
	var p pgdb.Package
	queryStr := fmt.Sprintf(`SELECT %s FROM "%d".packages where dataset_id = $1 and node_id = $2`, packageColumnsString, q.OrgId)
	if err := q.db.QueryRowContext(ctx, queryStr, datasetId, packageNodeId).Scan(
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
		&p.UpdatedAt); errors.Is(err, sql.ErrNoRows) {
		return &p, models.PackageNotFoundError{Id: models.PackageNodeId(packageNodeId), OrgId: q.OrgId, DatasetId: models.DatasetIntId(datasetId)}
	} else {
		return &p, err
	}
}

func (q *Queries) queryTrashcan(ctx context.Context, query string, datasetId int64, limit int, offset int) (*PackagePage, error) {
	rows, err := q.db.QueryContext(ctx, query, datasetId, limit, offset)
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

func (q *Queries) GetTrashcanRootPaginated(ctx context.Context, datasetId int64, limit int, offset int) (*PackagePage, error) {
	getTrashcanRootPageQuery := fmt.Sprintf(getTrashcanPageQueryFormat, "null::integer", "is null", qualifiedColumns("p", packagesColumns), q.OrgId)
	return q.queryTrashcan(ctx, getTrashcanRootPageQuery, datasetId, limit, offset)
}

func (q *Queries) GetTrashcanPaginated(ctx context.Context, datasetId int64, parentId int64, limit int, offset int) (*PackagePage, error) {
	pIdStr := strconv.FormatInt(parentId, 10)
	equalPIdStr := fmt.Sprintf("= %d", parentId)
	query := fmt.Sprintf(getTrashcanPageQueryFormat, pIdStr, equalPIdStr, qualifiedColumns("p", packagesColumns), q.OrgId)
	return q.queryTrashcan(ctx, query, datasetId, limit, offset)
}

func (q *Queries) GetDatasetManifest(ctx context.Context, datasetId int64) ([]models.DatasetManifest, error) {

	query := fmt.Sprintf(getManifestQueryFormat, q.OrgId, datasetId)

	rows, err := q.db.QueryContext(ctx, query)
	if err != nil {
		log.Println("ERROR: ", err)
		return nil, err
	}
	defer rows.Close()
	var files []models.DatasetManifest
	for rows.Next() {
		var m models.DatasetManifest
		err = rows.Scan(
			&m.PackageId,
			&m.PackageName,
			&m.FileName,
			pq.Array(&m.Path),
			&m.PackageNodeId,
			&m.Size,
			&m.CheckSum)

		if err != nil {
			log.Println("ERROR: ", err)
			return nil, err
		}

		files = append(files, m)
	}

	return files, nil

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
	CountDatasetPackagesByStates(ctx context.Context, datasetId int64, states []packageState.State) (int, error)
	GetDatasetPackageByNodeId(ctx context.Context, datasetId int64, packageNodeId string) (*pgdb.Package, error)
	GetDatasetManifest(ctx context.Context, datasetId int64) ([]models.DatasetManifest, error)
}
