package store

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestDBConnect(t *testing.T) {
	config := PostgresConfigFromEnv()

	db, err := config.OpenAtSchema("pennsieve")
	defer func() {
		if db != nil {
			assert.NoError(t, db.Close())
		}
	}()
	if assert.NoErrorf(t, err, "could not open postgres DB with config %s", config) {
		time.Sleep(10 * time.Second)
		err = db.Ping()
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
		err = rows.Err()
		assert.NoError(t, err)
	}
}
