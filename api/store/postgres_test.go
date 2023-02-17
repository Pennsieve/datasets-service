package store

import (
	"database/sql"
	"github.com/stretchr/testify/assert"
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
		rows, err := db.Query("SELECT name from organizations")
		if assert.NoError(t, err) {
			defer assert.NoError(t, rows.Close())

			for rows.Next() {
				var dbName string
				err = rows.Scan(&dbName)
				if assert.NoError(t, err) {
					t.Log(dbName)
				}
			}
		}
		assert.NoError(t, rows.Err())
	}
}
