package store

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/pennsieve/datasets-service/api/models"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/dbTable"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/packageInfo/packageState"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/packageInfo/packageType"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"
)

// pingUntilReady pings the db up to 10 times, stopping when
// a ping is successful. Used because there have been problems with
// the test DB not being fully started and ready to make connections.
// But there must be a better way.
func pingUntilReady(db *sql.DB) error {
	var err error
	wait := 100 * time.Millisecond
	for i := 0; i < 10; i++ {
		if err = db.Ping(); err == nil {
			return nil
		}
		time.Sleep(wait)
		wait = 2 * wait

	}
	return err
}

func loadFromFile(t *testing.T, db *sql.DB, sqlFile string) {
	path := filepath.Join("testdata", sqlFile)
	sqlBytes, ioErr := ioutil.ReadFile(path)
	if assert.NoError(t, ioErr) {
		sqlStr := string(sqlBytes)
		_, err := db.Exec(sqlStr)
		assert.NoError(t, err)
	}
}

func truncate(t *testing.T, db *sql.DB, orgID int, table string) {
	query := fmt.Sprintf("TRUNCATE TABLE \"%d\".%s CASCADE", orgID, table)
	_, err := db.Exec(query)
	assert.NoError(t, err)
}

func TestDBConnect(t *testing.T) {
	config := PostgresConfigFromEnv()

	db, err := config.OpenAtSchema("pennsieve")
	defer func() {
		if db != nil {
			assert.NoError(t, db.Close())
		}
	}()
	if assert.NoErrorf(t, err, "could not open postgres DB with config %s", config) {
		err = pingUntilReady(db)
		assert.NoErrorf(t, err, "could not ping postgres DB with config %s", config)
	}
}

func TestGetDatasetByNodeId(t *testing.T) {
	config := PostgresConfigFromEnv()
	db, err := config.Open()
	defer func() {
		if db != nil {
			assert.NoError(t, db.Close())
		}
	}()
	assert.NoErrorf(t, err, "could not open DB with config %s", config)

	orgId := 3
	store, err := NewDatasetStoreAtOrg(db, orgId)
	assert.NoError(t, err)
	input := dbTable.Dataset{
		Id:           1,
		Name:         "Test Dataset",
		State:        "READY",
		Description:  sql.NullString{},
		NodeId:       sql.NullString{String: "N:dataset:1234", Valid: true},
		Role:         sql.NullString{String: "editor", Valid: true},
		Tags:         dbTable.Tags{"test", "sql"},
		Contributors: dbTable.Contributors{},
		StatusId:     int32(1),
	}
	insert := fmt.Sprintf("INSERT INTO \"%d\".datasets (id, name, state, description, node_id, role, tags, contributors, status_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)", orgId)
	_, err = db.Exec(insert, input.Id, input.Name, input.State, input.Description, input.NodeId, input.Role, input.Tags, input.Contributors, input.StatusId)
	defer truncate(t, db, orgId, "datasets")

	if assert.NoError(t, err) {
		actual, err := store.GetDatasetByNodeId(context.Background(), input.NodeId.String)
		if assert.NoError(t, err) {
			assert.Equal(t, input.Name, actual.Name)
			assert.Equal(t, input.State, actual.State)
			assert.Equal(t, input.NodeId, actual.NodeId)
			assert.Equal(t, input.Role, actual.Role)
			assert.Equal(t, input.StatusId, actual.StatusId)

			assert.Equal(t, input.Tags, actual.Tags)
			assert.Equal(t, input.Contributors, actual.Contributors)
			assert.False(t, actual.Description.Valid)
		}
	}

}

func TestGetTrashcanRootPaginated(t *testing.T) {
	config := PostgresConfigFromEnv()
	db, err := config.Open()
	defer func() {
		if db != nil {
			assert.NoError(t, db.Close())
		}
	}()
	assert.NoErrorf(t, err, "could not open DB with config %s", config)
	loadFromFile(t, db, "folder-nav-test.sql")
	defer truncate(t, db, 2, "packages")
	expectedRoot := map[string]PackageSummary{
		"N:package:5ff98fab-d0d6-4cac-9f11-4b6ff50788e8": {
			Name:  "root-file-deleted-1.txt",
			Type:  packageType.Text,
			State: packageState.Deleted,
		},
		"N:collection:82c127ca-b72b-4d8b-a0c3-a9e4c7b14654": {
			Name:  "root-dir-deleted-1",
			Type:  packageType.Collection,
			State: packageState.Deleted,
		},
		"N:collection:180d4f48-ea2b-435c-ac69-780eeaf89745": {
			Name:  "root-dir-1",
			Type:  packageType.Collection,
			State: packageState.Ready,
		},
	}
	store, err := NewDatasetStoreAtOrg(db, 2)
	if assert.NoError(t, err) {
		rootPage, err := store.GetTrashcanRootPaginated(context.Background(), 1, 10, 0)
		if assert.NoError(t, err) {
			assert.Equal(t, 3, rootPage.TotalCount)
			assert.Len(t, rootPage.Packages, 3)
			rootSummary := summarize(rootPage.Packages)
			assert.Equal(t, expectedRoot, rootSummary)
		}
	}

}

func TestGetTrashcanPaginatedLevelOne(t *testing.T) {
	config := PostgresConfigFromEnv()
	db, err := config.Open()
	defer func() {
		if db != nil {
			assert.NoError(t, db.Close())
		}
	}()
	assert.NoErrorf(t, err, "could not open DB with config %s", config)
	loadFromFile(t, db, "folder-nav-test.sql")
	defer truncate(t, db, 2, "packages")
	expectedLevelOneRootDir1 := map[string]PackageSummary{
		"N:package:7a1e270b-eb23-4b26-b106-d32101399a8a": {
			Name:  "one-file-deleted-1.csv",
			Type:  packageType.CSV,
			State: packageState.Deleted,
		},
		"N:collection:e9bfe050-b375-43a1-91ec-b519439ad011": {
			Name:  "one-dir-1",
			Type:  packageType.Collection,
			State: packageState.Ready,
		},
		"N:collection:b8ab062e-e7d0-4668-b098-c322ae460820": {
			Name:  "one-dir-deleted-1",
			Type:  packageType.Collection,
			State: packageState.Deleted,
		},
	}
	expectedLevelOneRootDirDeleted1 := map[string]PackageSummary{
		"N:package:8d18065b-e7d7-4792-8de4-6fc7ecb79a46": {
			Name:  "one-file-deleted-1.csv",
			Type:  packageType.CSV,
			State: packageState.Deleted,
		},
		"N:package:40443908-a2e1-474c-8367-d04ffbda7947": {
			Name:  "one-file-deleted-2",
			Type:  packageType.Unsupported,
			State: packageState.Deleted,
		},
		"N:collection:8397346c-b824-4ee7-a49d-892860892d41": {
			Name:  "one-dir-deleted-1",
			Type:  packageType.Collection,
			State: packageState.Deleted,
		},
	}
	store, err := NewDatasetStoreAtOrg(db, 2)
	if assert.NoError(t, err) {
		oneRootDir1Page, err := store.GetTrashcanPaginated(context.Background(), 1, "N:collection:180d4f48-ea2b-435c-ac69-780eeaf89745", 10, 0)
		if assert.NoError(t, err) {
			assert.Equal(t, 3, oneRootDir1Page.TotalCount)
			oneRootDir1Summary := summarize(oneRootDir1Page.Packages)
			assert.Equal(t, expectedLevelOneRootDir1, oneRootDir1Summary)
		}
		oneRootDirDeleted1, err := store.GetTrashcanPaginated(context.Background(), 1, "N:collection:82c127ca-b72b-4d8b-a0c3-a9e4c7b14654", 10, 0)
		if assert.NoError(t, err) {
			assert.Equal(t, 3, oneRootDirDeleted1.TotalCount)
			oneRootDirDeleted1Summary := summarize(oneRootDirDeleted1.Packages)
			assert.Equal(t, expectedLevelOneRootDirDeleted1, oneRootDirDeleted1Summary)
		}
	}

}

func TestGetTrashcanPaginated_BadPackage(t *testing.T) {
	config := PostgresConfigFromEnv()
	db, err := config.Open()
	defer func() {
		if db != nil {
			assert.NoError(t, db.Close())
		}
	}()
	assert.NoErrorf(t, err, "could not open DB with config %s", config)
	loadFromFile(t, db, "folder-nav-test.sql")
	defer truncate(t, db, 2, "packages")
	store, err := NewDatasetStoreAtOrg(db, 2)
	if assert.NoError(t, err) {
		badRootNodeId := "N:collection:bad"
		_, err := store.GetTrashcanPaginated(context.Background(), 1, badRootNodeId, 10, 0)
		if assert.Error(t, err) {
			assert.Equal(t, models.PackageNotFoundError{OrgId: 2, NodeId: badRootNodeId}, err)
		}
	}

}

type PackageSummary struct {
	Name  string
	Type  packageType.Type
	State packageState.State
}

func summarize(packages []dbTable.Package) map[string]PackageSummary {
	summary := make(map[string]PackageSummary, len(packages))
	for _, p := range packages {
		summary[p.NodeId] = PackageSummary{
			Name:  p.Name,
			Type:  p.PackageType,
			State: p.PackageState,
		}
	}
	return summary
}
