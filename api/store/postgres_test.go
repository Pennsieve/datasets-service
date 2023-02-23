package store

import (
	"database/sql"
	"fmt"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/packageInfo/packageState"
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

func TestGetDatasetPackagesByState(t *testing.T) {
	config := PostgresConfigFromEnv()
	db, err := config.Open()
	defer func() {
		if db != nil {
			assert.NoError(t, db.Close())
		}
	}()
	assert.NoErrorf(t, err, "could not open DB with config %s", config)
	loadFromFile(t, db, "packages.sql")
	defer truncate(t, db, 2, "packages")

	store, err := NewDatasetStoreAtOrg(db, 2)
	assert.NoError(t, err)
	packagePage, err := store.GetDatasetPackagesByState(1, packageState.Deleted, 5, 0)
	assert.NoError(t, err)
	assert.Equal(t, 54, packagePage.TotalCount)
	assert.Len(t, packagePage.Packages, 5)
	for _, p := range packagePage.Packages {
		assert.Equal(t, packageState.Deleted, p.PackageState)
	}

}

func TestGetDatasetPackagesByStatePagination(t *testing.T) {
	config := PostgresConfigFromEnv()
	db, err := config.Open()
	defer func() {
		if db != nil {
			assert.NoError(t, db.Close())
		}
	}()
	assert.NoErrorf(t, err, "could not open DB with config %s", config)

	// File inserts packages, 54 of which are deleted.
	loadFromFile(t, db, "packages.sql")
	defer truncate(t, db, 2, "packages")

	store, err := NewDatasetStoreAtOrg(db, 2)
	assert.NoError(t, err)

	nodeIdSet := map[string]any{}
	const limit = 10
	offset := 0
	// First five pages
	for i := 0; i < 5; i++ {
		packagePage, err := store.GetDatasetPackagesByState(1, packageState.Deleted, limit, offset)
		assert.NoError(t, err)
		assert.Equal(t, 54, packagePage.TotalCount)
		assert.Len(t, packagePage.Packages, 10)
		for _, p := range packagePage.Packages {
			nodeIdSet[p.NodeId] = nil
		}
		offset = limit * (i + 1)
	}
	// Last page
	packagePage, err := store.GetDatasetPackagesByState(1, packageState.Deleted, limit, offset)
	assert.NoError(t, err)
	assert.Equal(t, 54, packagePage.TotalCount)
	assert.Len(t, packagePage.Packages, 4)
	for _, p := range packagePage.Packages {
		nodeIdSet[p.NodeId] = nil
	}

	assert.Len(t, nodeIdSet, 54)

}
