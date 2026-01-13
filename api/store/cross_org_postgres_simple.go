package store

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/lib/pq"
	"github.com/pennsieve/datasets-service/api/models"
	pg "github.com/pennsieve/pennsieve-go-core/pkg/queries/pgdb"
	log "github.com/sirupsen/logrus"
	"strings"
)

// CrossOrgQueriesSimple implements CrossOrgStore
type crossOrgQueriesSimple struct {
	db pg.DBTX
}

// NewCrossOrgQueriesSimple creates a new cross-org store that uses dynamic SQL
func NewCrossOrgQueriesSimple(db pg.DBTX) CrossOrgStore {
	return &crossOrgQueriesSimple{db: db}
}

// GetSharedDatasetsForUser retrieves all datasets shared with a user across all organizations
// This implementation builds dynamic SQL queries for each organization
func (q *crossOrgQueriesSimple) GetSharedDatasetsForUser(ctx context.Context, userId int, limit int, offset int) (*models.SharedDatasetsPage, error) {
	// Handle negative values gracefully
	if limit < 0 {
		limit = 0
	}
	if offset < 0 {
		offset = 0
	}

	// Step 1: Get all organizations where user has permission_bit == 1 (guest/limited access)
	// This indicates they have some access but aren't a full workspace contributor
	orgsQuery := `
		SELECT DISTINCT o.id, o.node_id, o.name
		FROM pennsieve.organizations o
		INNER JOIN pennsieve.organization_user ou ON ou.organization_id = o.id
		WHERE ou.user_id = $1
		AND ou.permission_bit = 1
		ORDER BY o.id
	`

	orgRows, err := q.db.QueryContext(ctx, orgsQuery, userId)
	if err != nil {
		log.WithError(err).Error("Failed to query organizations")
		return nil, fmt.Errorf("failed to query organizations: %w", err)
	}
	defer orgRows.Close()

	var orgIds []int
	var orgNodeIds []string
	var orgNames []string
	for orgRows.Next() {
		var orgId int
		var orgNodeId string
		var orgName string
		if err := orgRows.Scan(&orgId, &orgNodeId, &orgName); err != nil {
			log.WithError(err).Error("Failed to scan organization")
			return nil, fmt.Errorf("failed to scan organization: %w", err)
		}
		orgIds = append(orgIds, orgId)
		orgNodeIds = append(orgNodeIds, orgNodeId)
		orgNames = append(orgNames, orgName)
	}

	if err := orgRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating org rows: %w", err)
	}

	// Step 2: Build a UNION query for all organizations
	var unionParts []string
	var queryArgs []interface{}
	queryArgs = append(queryArgs, userId) // Add userId once for all UNION queries

	for i, orgId := range orgIds {
		// Build the query part for this organization
		// Note: Users with permission_bit = 1 are guests and cannot be part of teams,
		// so we only check dataset_user table for direct access
		orgQuery := fmt.Sprintf(`
			SELECT 
				d.node_id,
				d.name,
				d.description,
				d.state,
				d.created_at,
				d.updated_at,
				d.status,
				d.tags,
				d.data_use_agreement_id,
				d.id,
				%d as org_id,
				'%s' as org_node_id,
				'%s' as org_name
			FROM "%d".datasets d
			WHERE EXISTS (
				-- User has direct access (guests cannot be part of teams)
				SELECT 1 FROM "%d".dataset_user du
				WHERE du.dataset_id = d.id
				AND du.user_id = $1
			)
			AND d.state NOT IN ('DELETED', 'DELETING')
		`, orgId, orgNodeIds[i], orgNames[i], orgId, orgId)

		unionParts = append(unionParts, orgQuery)
	}

	if len(unionParts) == 0 {
		// No organizations with shared datasets
		return &models.SharedDatasetsPage{
			Limit:      limit,
			Offset:     offset,
			TotalCount: 0,
			Datasets:   []models.SharedDatasetItem{},
		}, nil
	}

	// Step 3: Combine all parts with UNION and add pagination
	fullQuery := fmt.Sprintf(`
		WITH all_shared_datasets AS (
			%s
		)
		SELECT 
			node_id,
			name,
			description,
			state,
			created_at,
			updated_at,
			status,
			tags,
			data_use_agreement_id,
			id,
			org_node_id,
			org_name,
			COUNT(*) OVER() as total_count
		FROM all_shared_datasets
		ORDER BY updated_at DESC, name
		LIMIT $%d OFFSET $%d
	`, strings.Join(unionParts, " UNION ALL "), len(queryArgs)+1, len(queryArgs)+2)

	queryArgs = append(queryArgs, limit, offset)

	// Execute the query
	rows, err := q.db.QueryContext(ctx, fullQuery, queryArgs...)
	if err != nil {
		log.WithError(err).WithField("query", fullQuery).Error("Failed to query shared datasets")
		return nil, fmt.Errorf("failed to query shared datasets: %w", err)
	}
	defer rows.Close()

	var datasets []models.SharedDatasetItem
	var totalCount int
	hasRows := false

	for rows.Next() {
		hasRows = true
		var content models.SharedDatasetContent
		var description sql.NullString
		var dataUseAgreementId sql.NullInt32
		var tags pq.StringArray
		var intId int

		err := rows.Scan(
			&content.ID,
			&content.Name,
			&description,
			&content.State,
			&content.CreatedAt,
			&content.UpdatedAt,
			&content.Status,
			&tags,
			&dataUseAgreementId,
			&intId,
			&content.WorkspaceNodeID,
			&content.WorkspaceName,
			&totalCount,
		)
		if err != nil {
			log.WithError(err).Error("Failed to scan dataset row")
			return nil, fmt.Errorf("failed to scan dataset: %w", err)
		}

		content.IntId = intId
		if description.Valid {
			content.Description = description.String
		}
		if dataUseAgreementId.Valid {
			agreementId := int(dataUseAgreementId.Int32)
			content.DataUseAgreementID = &agreementId
		}
		content.Tags = []string(tags)

		datasets = append(datasets, models.SharedDatasetItem{
			Content: content,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating dataset rows: %w", err)
	}

	// If we got no rows (e.g., offset beyond results), get the total count separately
	if !hasRows && len(unionParts) > 0 {
		countQuery := fmt.Sprintf(`
			WITH all_shared_datasets AS (
				%s
			)
			SELECT COUNT(*) FROM all_shared_datasets
		`, strings.Join(unionParts, " UNION ALL "))

		err := q.db.QueryRowContext(ctx, countQuery, userId).Scan(&totalCount)
		if err != nil && err != sql.ErrNoRows {
			log.WithError(err).Error("Failed to get total count")
			return nil, fmt.Errorf("failed to get total count: %w", err)
		}
	}

	return &models.SharedDatasetsPage{
		Limit:      limit,
		Offset:     offset,
		TotalCount: totalCount,
		Datasets:   datasets,
	}, nil
}
