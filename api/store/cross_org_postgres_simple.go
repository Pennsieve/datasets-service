package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"log/slog"

	"github.com/lib/pq"
	"github.com/pennsieve/datasets-service/api/logging"
	"github.com/pennsieve/datasets-service/api/models"
	pg "github.com/pennsieve/pennsieve-go-core/pkg/queries/pgdb"
)

// CrossOrgQueriesSimple implements CrossOrgStore
type crossOrgQueriesSimple struct {
	db     pg.DBTX
	Logger *slog.Logger
}

// NewCrossOrgQueriesSimple creates a new cross-org store that uses dynamic SQL.
// Takes the request-scoped logger so DB failures carry the invocation's
// trace/request context; a nil logger falls back to slog.Default().
func NewCrossOrgQueriesSimple(db pg.DBTX, logger *slog.Logger) CrossOrgStore {
	if logger == nil {
		logger = slog.Default()
	}
	return &crossOrgQueriesSimple{db: db, Logger: logger}
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
		q.Logger.Error("failed to query organizations",
			slog.Any(logging.ErrorKey, err),
			slog.Int(logging.UserIdKey, userId))
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
			q.Logger.Error("failed to scan organization",
				slog.Any(logging.ErrorKey, err),
				slog.Int(logging.UserIdKey, userId))
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
	queryArgs = append(queryArgs, userId) // $1, shared by all UNION branches

	for i, orgId := range orgIds {
		// Build the query part for this organization.
		// Note: Users with permission_bit = 1 are guests and cannot be part of teams,
		// so we only check dataset_user table for direct access.
		//
		// org_id and the "%d".datasets/dataset_user schema names come from
		// organizations.id (an integer from our own DB) and are safe to format in.
		// org_node_id and org_name are bound as parameters, not interpolated:
		// org_name is user-settable (workspace rename), so string-formatting it
		// would be a stored SQL-injection vector.
		queryArgs = append(queryArgs, orgNodeIds[i], orgNames[i])
		nodeIDPlaceholder := len(queryArgs) - 1 // index of orgNodeIds[i]
		namePlaceholder := len(queryArgs)       // index of orgNames[i]

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
				$%d::text as org_node_id,
				$%d::text as org_name
			FROM "%d".datasets d
			WHERE EXISTS (
				-- User has direct access (guests cannot be part of teams)
				SELECT 1 FROM "%d".dataset_user du
				WHERE du.dataset_id = d.id
				AND du.user_id = $1
			)
			AND d.state NOT IN ('DELETED', 'DELETING')
		`, orgId, nodeIDPlaceholder, namePlaceholder, orgId, orgId)

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
		q.Logger.Error("failed to query shared datasets",
			slog.Any(logging.ErrorKey, err),
			slog.Int(logging.UserIdKey, userId),
			slog.String(logging.QueryKey, fullQuery))
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
			q.Logger.Error("failed to scan dataset row",
				slog.Any(logging.ErrorKey, err),
				slog.Int(logging.UserIdKey, userId))
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

		// The UNION parts reference $1 (userId) plus a bound org_node_id/org_name
		// pair per branch; reuse those same args (everything except the trailing
		// limit/offset appended above).
		countArgs := queryArgs[:len(queryArgs)-2]
		err := q.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&totalCount)
		if err != nil && err != sql.ErrNoRows {
			q.Logger.Error("failed to get total count",
				slog.Any(logging.ErrorKey, err),
				slog.Int(logging.UserIdKey, userId))
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
