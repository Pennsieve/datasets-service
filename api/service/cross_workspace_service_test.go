package service

import (
	"context"
	"testing"

	"github.com/pennsieve/datasets-service/api/store"
	"github.com/stretchr/testify/assert"
)

func TestGetSharedDatasetsPage(t *testing.T) {
	db := store.OpenDB(t)
	defer db.Close()

	// Load test data
	db.ExecSQLFile("shared-datasets-test.sql")
	defer func() {
		// Clean up all test organizations
		db.Truncate(100, "dataset_user")
		db.Truncate(100, "datasets")
		db.Truncate(101, "dataset_user")
		db.Truncate(101, "datasets")
		db.TruncatePennsieve("organization_user")
		db.TruncatePennsieve("organizations")
		db.TruncatePennsieve("users")
	}()

	service := NewCrossWorkspaceDatasetsService(db.DB)

	tests := []struct {
		name           string
		userId         int
		limit          int
		offset         int
		expectedCount  int
		expectedTotal  int
		expectedNames  []string
	}{
		{
			name:           "user with guest access to multiple orgs",
			userId:         9001, // guest.user@example.com with permission_bit=1 in orgs 100 and 101
			limit:          10,
			offset:         0,
			expectedCount:  4,                                                                              // alpha1, alpha2, beta1, beta2
			expectedTotal:  4,
			expectedNames:  []string{"Alpha Dataset 1", "Alpha Dataset 2", "Beta Dataset 1", "Beta Dataset 2"},
		},
		{
			name:           "user with full contributor access (should not see shared datasets)",
			userId:         9002, // full.contributor@example.com with permission_bit=32 in org 100
			limit:          10,
			offset:         0,
			expectedCount:  0, // Full contributors don't appear in shared datasets
			expectedTotal:  0,
			expectedNames:  []string{},
		},
		{
			name:           "user with guest access to single org",
			userId:         9003, // another.guest@example.com with permission_bit=1 in org 101 only
			limit:          10,
			offset:         0,
			expectedCount:  1, // Only has access to beta1
			expectedTotal:  1,
			expectedNames:  []string{"Beta Dataset 1"},
		},
		{
			name:           "pagination - limit 2",
			userId:         9001,
			limit:          2,
			offset:         0,
			expectedCount:  2,
			expectedTotal:  4, // Still 4 total, but only 2 in this page
			expectedNames:  []string{}, // Don't check exact names due to ordering
		},
		{
			name:           "pagination - offset 2",
			userId:         9001,
			limit:          10,
			offset:         2,
			expectedCount:  2,
			expectedTotal:  4,
			expectedNames:  []string{}, // Don't check exact names due to ordering
		},
		{
			name:           "user with no organization access",
			userId:         999, // Non-existent user
			limit:          10,
			offset:         0,
			expectedCount:  0,
			expectedTotal:  0,
			expectedNames:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page, err := service.GetSharedDatasetsPage(context.Background(), tt.userId, tt.limit, tt.offset)
			
			assert.NoError(t, err)
			assert.NotNil(t, page)
			assert.Equal(t, tt.limit, page.Limit)
			assert.Equal(t, tt.offset, page.Offset)
			assert.Equal(t, tt.expectedTotal, page.TotalCount)
			assert.Len(t, page.Datasets, tt.expectedCount)

			if len(tt.expectedNames) > 0 {
				var actualNames []string
				for _, dataset := range page.Datasets {
					actualNames = append(actualNames, dataset.Content.Name)
				}
				
				// For exact name checking, ensure we have all expected names
				for _, expectedName := range tt.expectedNames {
					assert.Contains(t, actualNames, expectedName)
				}
			}

			// Validate dataset structure for non-empty results
			for _, dataset := range page.Datasets {
				assert.NotEmpty(t, dataset.Content.ID, "Dataset ID should not be empty")
				assert.NotEmpty(t, dataset.Content.Name, "Dataset name should not be empty") 
				assert.NotEmpty(t, dataset.Content.State, "Dataset state should not be empty")
				assert.NotZero(t, dataset.Content.IntId, "Dataset IntId should not be zero")
				assert.NotNil(t, dataset.Content.Tags, "Dataset tags should not be nil")
				assert.NotEmpty(t, dataset.Content.WorkspaceNodeID, "Workspace node ID should not be empty")
				assert.NotEmpty(t, dataset.Content.WorkspaceName, "Workspace name should not be empty")
				
				// Ensure no deleted datasets appear
				assert.NotEqual(t, "DELETED", dataset.Content.State)
				assert.NotEqual(t, "DELETING", dataset.Content.State)
			}
		})
	}
}

func TestGetSharedDatasetsPageEdgeCases(t *testing.T) {
	db := store.OpenDB(t)
	defer db.Close()

	// Load test data  
	db.ExecSQLFile("shared-datasets-test.sql")
	defer func() {
		// Clean up all test organizations
		db.Truncate(100, "dataset_user")
		db.Truncate(100, "datasets")
		db.Truncate(101, "dataset_user") 
		db.Truncate(101, "datasets")
		db.TruncatePennsieve("organization_user")
		db.TruncatePennsieve("organizations")
		db.TruncatePennsieve("users")
	}()

	service := NewCrossWorkspaceDatasetsService(db.DB)

	t.Run("negative limit should be handled gracefully", func(t *testing.T) {
		page, err := service.GetSharedDatasetsPage(context.Background(), 9001, -1, 0)
		// The service should handle this gracefully, likely treating it as 0 or a default value
		assert.NoError(t, err)
		assert.NotNil(t, page)
	})

	t.Run("negative offset should be handled gracefully", func(t *testing.T) {
		page, err := service.GetSharedDatasetsPage(context.Background(), 9001, 10, -1)
		// The service should handle this gracefully, likely treating it as 0
		assert.NoError(t, err)
		assert.NotNil(t, page)
	})

	t.Run("large offset beyond results", func(t *testing.T) {
		page, err := service.GetSharedDatasetsPage(context.Background(), 9001, 10, 1000)
		assert.NoError(t, err)
		assert.NotNil(t, page)
		assert.Equal(t, 4, page.TotalCount) // Still knows total count
		assert.Len(t, page.Datasets, 0)     // But no results in this page
	})
}