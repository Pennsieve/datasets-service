package store

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests exercise GetSharedDatasetsForUser with organization names that
// contain characters which are dangerous when interpolated into raw SQL. Org
// names are user-settable (workspace rename), so this is a stored SQL-injection
// vector; the query must bind them as parameters rather than string-format them.
//
// Fixture cross-org-simple-test.sql sets up org 100 with user 9001 as a Guest
// (permission_bit = 1) directly sharing dataset N:dataset:alpha1 — the exact
// shape GetSharedDatasetsForUser looks for. Each test renames org 100 to a
// hostile value, runs the query, and restores the name.

const sharedOrgID = 100
const sharedUserID = 9001

// withOrgName temporarily sets org sharedOrgID's name and restores it after.
func withOrgName(t *testing.T, db TestDB, name string) func() {
	t.Helper()
	var original string
	require.NoError(t, db.QueryRow(`SELECT name FROM pennsieve.organizations WHERE id=$1`, sharedOrgID).Scan(&original))
	_, err := db.Exec(`UPDATE pennsieve.organizations SET name=$1 WHERE id=$2`, name, sharedOrgID)
	require.NoError(t, err, "rename org")
	return func() {
		_, _ = db.Exec(`UPDATE pennsieve.organizations SET name=$1 WHERE id=$2`, original, sharedOrgID)
	}
}

func TestGetSharedDatasetsForUser_ApostropheInOrgName(t *testing.T) {
	db := OpenDB(t)
	defer db.Close()
	db.ExecSQLFile("cross-org-simple-test.sql")

	defer withOrgName(t, db, "Michael's Workspace")()

	store := NewCrossOrgQueriesSimple(db.DB, slog.Default())
	page, err := store.GetSharedDatasetsForUser(context.Background(), sharedUserID, 10, 0)
	require.NoError(t, err, "apostrophe in org name must not break the query")
	require.NotNil(t, page)

	found := false
	for _, d := range page.Datasets {
		if d.Content.WorkspaceName == "Michael's Workspace" {
			found = true
		}
	}
	assert.True(t, found, "expected the shared dataset with the apostrophe org name to be returned")
}

func TestGetSharedDatasetsForUser_InjectionInOrgName(t *testing.T) {
	db := OpenDB(t)
	defer db.Close()
	db.ExecSQLFile("cross-org-simple-test.sql")

	// A classic injection payload as the workspace name. Under parameterization it
	// is inert data; under string interpolation it would corrupt/execute SQL.
	payload := "x'; DROP TABLE pennsieve.organizations; --"

	defer withOrgName(t, db, payload)()

	store := NewCrossOrgQueriesSimple(db.DB, slog.Default())
	page, err := store.GetSharedDatasetsForUser(context.Background(), sharedUserID, 10, 0)
	require.NoError(t, err, "injection payload in org name must be handled safely")
	require.NotNil(t, page)

	// The table must still exist (payload was not executed).
	var n int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM pennsieve.organizations`).Scan(&n))
	assert.Greater(t, n, 0, "organizations table must be intact (injection not executed)")

	found := false
	for _, d := range page.Datasets {
		if d.Content.WorkspaceName == payload {
			found = true
		}
	}
	assert.True(t, found, "the org name payload should be returned verbatim as data")
}
