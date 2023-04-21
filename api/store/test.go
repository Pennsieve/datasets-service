package store

import (
	"database/sql"
	"fmt"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type TestDB struct {
	*sql.DB
	t *testing.T
}

// PingUntilReady pings the db up to 10 times, stopping when
// a ping is successful. Used because there have been problems on Jenkins with
// the test DB not being fully started and ready to make connections.
// But there must be a better way.
func (tdb *TestDB) PingUntilReady() error {
	var err error
	wait := 100 * time.Millisecond
	for i := 0; i < 10; i++ {
		if err = tdb.Ping(); err == nil {
			return nil
		}
		time.Sleep(wait)
		wait = 2 * wait

	}
	return err
}

func OpenDB(t *testing.T) TestDB {
	pgConfig := PostgresConfigFromEnv()
	db, err := pgConfig.Open()
	if err != nil {
		assert.FailNowf(t, "cannot open database", "config: %s, err: %v", pgConfig, err)
	}
	testDB := TestDB{
		DB: db,
		t:  t,
	}
	if err = testDB.PingUntilReady(); err != nil {
		assert.FailNow(testDB.t, "cannot ping database", "config: %s, err: %v", pgConfig, err)
	}
	return testDB
}

func (tdb *TestDB) ExecSQLFile(sqlFile string) {
	path := filepath.Join("testdata", sqlFile)
	sqlBytes, err := os.ReadFile(path)
	if err != nil {
		assert.FailNowf(tdb.t, "error reading SQL file", "%s: %v", path, err)
	}
	sqlStr := string(sqlBytes)
	_, err = tdb.Exec(sqlStr)
	if err != nil {
		assert.FailNowf(tdb.t, "error executing SQL file", "%s: %v", path, err)
	}
}

func (tdb *TestDB) Truncate(orgID int, table string) {
	query := fmt.Sprintf(`TRUNCATE TABLE "%d".%s CASCADE`, orgID, table)
	_, err := tdb.Exec(query)
	if err != nil {
		assert.FailNowf(tdb.t, "error truncating table", "orgID: %d, table: %s, error: %v", orgID, table, err)
	}
}

func (tdb *TestDB) TruncatePennsieve(table string) {
	query := fmt.Sprintf(`TRUNCATE TABLE pennsieve.%s CASCADE`, table)
	_, err := tdb.Exec(query)
	if err != nil {
		assert.FailNowf(tdb.t, "error truncating table in pennsieve schema", "table: %s, error: %v", table, err)
	}
}

func (tdb *TestDB) Close() {
	if err := tdb.DB.Close(); err != nil {
		assert.FailNowf(tdb.t, "error closing database", "error: %v", err)
	}
}

func (tdb *TestDB) CloseRows(rows *sql.Rows) {
	if err := rows.Close(); err != nil {
		assert.FailNowf(tdb.t, "error cloting rows", "error: %v", err)
	}
}

func (tdb *TestDB) Queries(orgId int) *Queries {
	return &Queries{
		db:    tdb.DB,
		OrgId: orgId,
	}
}
