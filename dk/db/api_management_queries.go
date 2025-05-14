package db

import (
	"database/sql"
	"fmt"
	"github.com/google/uuid"
	"time"
)

// This file contains the CRUD operations for the API Management tables

// Use ErrNotFound from errors.go

// CreateAPI inserts a new API record
func CreateAPI(db *sql.DB, api *API) error {
	// Generate UUID if not provided
	if api.ID == "" {
		api.ID = uuid.New().String()
	}

	// Generate API key if not provided
	if api.APIKey == "" {
		apiKey, err := generateAPIKey()
		if err != nil {
			return fmt.Errorf("failed to generate API key: %v", err)
		}
		api.APIKey = apiKey
	}

	// Set timestamps
	now := time.Now()
	api.CreatedAt = now
	api.UpdatedAt = now

	query := `
		INSERT INTO apis (
			id, name, description, created_at, updated_at, is_active, 
			api_key, host_user_id, policy_id, is_deprecated, 
			deprecation_date, deprecation_message
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := db.Exec(
		query,
		api.ID,
		api.Name,
		api.Description,
		api.CreatedAt,
		api.UpdatedAt,
		api.IsActive,
		api.APIKey,
		api.HostUserID,
		api.PolicyID,
		api.IsDeprecated,
		api.DeprecationDate,
		api.DeprecationMessage,
	)

	return err
}

// CreateAPITx inserts a new API record within a transaction
func CreateAPITx(tx *sql.Tx, api *API) error {
	// Generate UUID if not provided
	if api.ID == "" {
		api.ID = uuid.New().String()
	}

	// Generate API key if not provided
	if api.APIKey == "" {
		apiKey, err := generateAPIKey()
		if err != nil {
			return fmt.Errorf("failed to generate API key: %v", err)
		}
		api.APIKey = apiKey
	}

	// Set timestamps
	now := time.Now()
	api.CreatedAt = now
	api.UpdatedAt = now

	query := `
		INSERT INTO apis (
			id, name, description, created_at, updated_at, is_active, 
			api_key, host_user_id, policy_id, is_deprecated, 
			deprecation_date, deprecation_message
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := tx.Exec(
		query,
		api.ID,
		api.Name,
		api.Description,
		api.CreatedAt,
		api.UpdatedAt,
		api.IsActive,
		api.APIKey,
		api.HostUserID,
		api.PolicyID,
		api.IsDeprecated,
		api.DeprecationDate,
		api.DeprecationMessage,
	)

	return err
}

// GetAPI retrieves an API by ID
func GetAPI(db *sql.DB, id string) (*API, error) {
	query := `
		SELECT id, name, description, created_at, updated_at, is_active, 
			api_key, host_user_id, policy_id, is_deprecated, 
			deprecation_date, deprecation_message
		FROM apis
		WHERE id = ?
	`

	api := &API{}
	var policyID sql.NullString
	var deprecationDate sql.NullTime
	var deprecationMessage sql.NullString

	err := db.QueryRow(query, id).Scan(
		&api.ID,
		&api.Name,
		&api.Description,
		&api.CreatedAt,
		&api.UpdatedAt,
		&api.IsActive,
		&api.APIKey,
		&api.HostUserID,
		&policyID,
		&api.IsDeprecated,
		&deprecationDate,
		&deprecationMessage,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Handle nullable fields
	if policyID.Valid {
		api.PolicyID = &policyID.String
	}

	if deprecationDate.Valid {
		api.DeprecationDate = &deprecationDate.Time
	}

	if deprecationMessage.Valid {
		api.DeprecationMessage = deprecationMessage.String
	}

	return api, nil
}

// UpdateAPI updates an existing API record
func UpdateAPI(db *sql.DB, api *API) error {
	// Update timestamp
	api.UpdatedAt = time.Now()

	query := `
		UPDATE apis
		SET name = ?, description = ?, updated_at = ?, is_active = ?, 
			api_key = ?, host_user_id = ?, policy_id = ?, is_deprecated = ?, 
			deprecation_date = ?, deprecation_message = ?
		WHERE id = ?
	`

	result, err := db.Exec(
		query,
		api.Name,
		api.Description,
		api.UpdatedAt,
		api.IsActive,
		api.APIKey,
		api.HostUserID,
		api.PolicyID,
		api.IsDeprecated,
		api.DeprecationDate,
		api.DeprecationMessage,
		api.ID,
	)

	if err != nil {
		return err
	}

	// Check if any row was affected
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// DeleteAPI deletes an API record
func DeleteAPI(db *sql.DB, id string) error {
	query := "DELETE FROM apis WHERE id = ?"
	result, err := db.Exec(query, id)
	if err != nil {
		return err
	}

	// Check if any row was affected
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// ListAPIs retrieves a paginated, filtered list of APIs
func ListAPIs(db *sql.DB, status, externalUserID string, limit, offset int, sort, order string) ([]*API, int, error) {
	// Build the query based on filters
	query := "SELECT id, name, description, created_at, updated_at, is_active, api_key, host_user_id, policy_id, is_deprecated, deprecation_date, deprecation_message FROM apis WHERE 1=1"
	countQuery := "SELECT COUNT(*) FROM apis WHERE 1=1"

	args := []interface{}{}

	// Apply status filter
	if status != "" {
		switch status {
		case "active":
			query += " AND is_active = TRUE AND is_deprecated = FALSE"
			countQuery += " AND is_active = TRUE AND is_deprecated = FALSE"
		case "inactive":
			query += " AND is_active = FALSE AND is_deprecated = FALSE"
			countQuery += " AND is_active = FALSE AND is_deprecated = FALSE"
		case "deprecated":
			query += " AND is_deprecated = TRUE"
			countQuery += " AND is_deprecated = TRUE"
		}
	}

	// Apply external user filter
	if externalUserID != "" {
		query += " AND id IN (SELECT api_id FROM api_user_access WHERE external_user_id = ? AND is_active = TRUE)"
		countQuery += " AND id IN (SELECT api_id FROM api_user_access WHERE external_user_id = ? AND is_active = TRUE)"
		args = append(args, externalUserID)
	}

	// Apply sorting
	if sort == "" {
		sort = "created_at" // default
	}

	if sort != "name" && sort != "created_at" {
		sort = "created_at" // fallback to default for invalid sort fields
	}

	query += " ORDER BY " + sort
	if order != "" {
		if order != "asc" && order != "desc" {
			order = "desc" // fallback to default for invalid order
		}
		query += " " + order
	} else {
		query += " DESC" // default order
	}

	// Apply pagination
	query += " LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	// Get total count
	var total int
	err := db.QueryRow(countQuery, args[:len(args)-2]...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count APIs: %v", err)
	}

	// Execute the query
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query APIs: %v", err)
	}
	defer rows.Close()

	// Process results
	apis := []*API{}
	for rows.Next() {
		api := &API{}
		var policyID sql.NullString
		var deprecationDate sql.NullTime
		var deprecationMessage sql.NullString

		err := rows.Scan(
			&api.ID,
			&api.Name,
			&api.Description,
			&api.CreatedAt,
			&api.UpdatedAt,
			&api.IsActive,
			&api.APIKey,
			&api.HostUserID,
			&policyID,
			&api.IsDeprecated,
			&deprecationDate,
			&deprecationMessage,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan API row: %v", err)
		}

		if policyID.Valid {
			policyIDStr := policyID.String
			api.PolicyID = &policyIDStr
		}

		if deprecationDate.Valid {
			api.DeprecationDate = &deprecationDate.Time
		}

		if deprecationMessage.Valid {
			api.DeprecationMessage = deprecationMessage.String
		}

		apis = append(apis, api)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating API rows: %v", err)
	}

	return apis, total, nil
}

// CountAPIExternalUsers counts how many external users have access to an API
func CountAPIExternalUsers(db *sql.DB, apiID string) (int, error) {
	query := "SELECT COUNT(*) FROM api_user_access WHERE api_id = ? AND is_active = TRUE"
	var count int
	err := db.QueryRow(query, apiID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count API external users: %v", err)
	}
	return count, nil
}

// CountAPIDocuments counts how many documents are associated with an API
func CountAPIDocuments(db *sql.DB, apiID string) (int, error) {
	query := "SELECT COUNT(*) FROM document_associations WHERE entity_id = ? AND entity_type = 'api'"
	var count int
	err := db.QueryRow(query, apiID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count API documents: %v", err)
	}
	return count, nil
}

// GetAPIExternalUsers retrieves all external users with access to an API
func GetAPIExternalUsers(db *sql.DB, apiID string) ([]*APIUserAccess, error) {
	query := `
		SELECT id, api_id, external_user_id, access_level, granted_at, granted_by, revoked_at, is_active
		FROM api_user_access
		WHERE api_id = ? AND is_active = TRUE
	`

	rows, err := db.Query(query, apiID)
	if err != nil {
		return nil, fmt.Errorf("failed to query API external users: %v", err)
	}
	defer rows.Close()

	users := []*APIUserAccess{}
	for rows.Next() {
		user := &APIUserAccess{}
		var revokedAt sql.NullTime
		var grantedBy sql.NullString

		err := rows.Scan(
			&user.ID,
			&user.APIID,
			&user.ExternalUserID,
			&user.AccessLevel,
			&user.GrantedAt,
			&grantedBy,
			&revokedAt,
			&user.IsActive,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan API user access row: %v", err)
		}

		if grantedBy.Valid {
			user.GrantedBy = grantedBy.String
		}

		if revokedAt.Valid {
			user.RevokedAt = &revokedAt.Time
		}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating API user access rows: %v", err)
	}

	return users, nil
}

// GetAPIUserAccess retrieves a single API user access record by ID
func GetAPIUserAccess(db *sql.DB, id string) (*APIUserAccess, error) {
	query := `
		SELECT id, api_id, external_user_id, access_level,
		       granted_at, granted_by, revoked_at, is_active
		FROM api_user_access
		WHERE id = ?
	`

	access := &APIUserAccess{}
	var revokedAt sql.NullTime
	var grantedBy sql.NullString

	err := db.QueryRow(query, id).Scan(
		&access.ID,
		&access.APIID,
		&access.ExternalUserID,
		&access.AccessLevel,
		&access.GrantedAt,
		&grantedBy,
		&revokedAt,
		&access.IsActive,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get API user access: %v", err)
	}

	if grantedBy.Valid {
		access.GrantedBy = grantedBy.String
	}

	if revokedAt.Valid {
		access.RevokedAt = &revokedAt.Time
	}

	return access, nil
}

// GetAPIUserAccessByUserID retrieves a single API user access record by API ID and user ID
func GetAPIUserAccessByUserID(db *sql.DB, apiID, userID string) (*APIUserAccess, error) {
	query := `
		SELECT id, api_id, external_user_id, access_level,
		       granted_at, granted_by, revoked_at, is_active
		FROM api_user_access
		WHERE api_id = ? AND external_user_id = ?
	`

	access := &APIUserAccess{}
	var revokedAt sql.NullTime
	var grantedBy sql.NullString

	err := db.QueryRow(query, apiID, userID).Scan(
		&access.ID,
		&access.APIID,
		&access.ExternalUserID,
		&access.AccessLevel,
		&access.GrantedAt,
		&grantedBy,
		&revokedAt,
		&access.IsActive,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get API user access: %v", err)
	}

	if grantedBy.Valid {
		access.GrantedBy = grantedBy.String
	}

	if revokedAt.Valid {
		access.RevokedAt = &revokedAt.Time
	}

	return access, nil
}

// ListAPIUserAccess retrieves a list of API user access records for a specific API
func ListAPIUserAccess(db *sql.DB, apiID string, activeOnly bool, limit, offset int, sort, order string) ([]*APIUserAccess, int, error) {
	// Build query
	baseQuery := "FROM api_user_access WHERE api_id = ?"
	args := []interface{}{apiID}

	if activeOnly {
		baseQuery += " AND is_active = TRUE"
	}

	// Count total
	var total int
	countQuery := "SELECT COUNT(*) " + baseQuery
	err := db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count API user access records: %v", err)
	}

	// Build main query
	query := `
		SELECT id, api_id, external_user_id, access_level,
		       granted_at, granted_by, revoked_at, is_active
		` + baseQuery + `
		ORDER BY ` + sort + ` ` + order + `
		LIMIT ? OFFSET ?
	`
	args = append(args, limit, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query API user access records: %v", err)
	}
	defer rows.Close()

	// Process results
	accessRecords := []*APIUserAccess{}
	for rows.Next() {
		access := &APIUserAccess{}
		var revokedAt sql.NullTime
		var grantedBy sql.NullString

		err := rows.Scan(
			&access.ID,
			&access.APIID,
			&access.ExternalUserID,
			&access.AccessLevel,
			&access.GrantedAt,
			&grantedBy,
			&revokedAt,
			&access.IsActive,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan API user access row: %v", err)
		}

		if grantedBy.Valid {
			access.GrantedBy = grantedBy.String
		}

		if revokedAt.Valid {
			access.RevokedAt = &revokedAt.Time
		}

		accessRecords = append(accessRecords, access)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating API user access rows: %v", err)
	}

	return accessRecords, total, nil
}

// UpdateAPIUserAccess updates an existing API user access record
func UpdateAPIUserAccess(db *sql.DB, access *APIUserAccess) error {
	query := `
		UPDATE api_user_access
		SET access_level = ?, revoked_at = ?, is_active = ?
		WHERE id = ?
	`

	// Handle null fields
	var revokedAt interface{}
	if access.RevokedAt != nil {
		revokedAt = *access.RevokedAt
	}

	result, err := db.Exec(
		query,
		access.AccessLevel,
		revokedAt,
		access.IsActive,
		access.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update API user access: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// GetAPIDocuments retrieves all documents associated with an API
func GetAPIDocuments(db *sql.DB, apiID string) ([]*DocumentAssociation, error) {
	query := `
		SELECT id, document_filename, entity_id, entity_type, created_at
		FROM document_associations
		WHERE entity_id = ? AND entity_type = 'api'
	`

	rows, err := db.Query(query, apiID)
	if err != nil {
		return nil, fmt.Errorf("failed to query API documents: %v", err)
	}
	defer rows.Close()

	docs := []*DocumentAssociation{}
	for rows.Next() {
		doc := &DocumentAssociation{}

		err := rows.Scan(
			&doc.ID,
			&doc.DocumentFilename,
			&doc.EntityID,
			&doc.EntityType,
			&doc.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan document association row: %v", err)
		}

		docs = append(docs, doc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating document association rows: %v", err)
	}

	return docs, nil
}

// Document association database functions

// CreateDocumentAssociation creates a new document association
func CreateDocumentAssociation(db *sql.DB, assoc *DocumentAssociation) error {
	// Check if association already exists
	var count int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM document_associations WHERE document_filename = ? AND entity_id = ? AND entity_type = ?",
		assoc.DocumentFilename, assoc.EntityID, assoc.EntityType,
	).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check for existing document association: %v", err)
	}

	if count > 0 {
		return fmt.Errorf("document is already associated with this entity")
	}

	// Generate UUID if not provided
	if assoc.ID == "" {
		assoc.ID = uuid.New().String()
	}

	// Set timestamp if not provided
	if assoc.CreatedAt.IsZero() {
		assoc.CreatedAt = time.Now()
	}

	// Create new association
	query := `
		INSERT INTO document_associations (
			id, document_filename, entity_id, entity_type, created_at
		) VALUES (?, ?, ?, ?, ?)
	`

	_, err = db.Exec(
		query,
		assoc.ID,
		assoc.DocumentFilename,
		assoc.EntityID,
		assoc.EntityType,
		assoc.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create document association: %v", err)
	}

	return nil
}

// CreateDocumentAssociationTx creates a new document association within a transaction
func CreateDocumentAssociationTx(tx *sql.Tx, assoc *DocumentAssociation) error {
	// Check if association already exists
	var count int
	err := tx.QueryRow(
		"SELECT COUNT(*) FROM document_associations WHERE document_filename = ? AND entity_id = ? AND entity_type = ?",
		assoc.DocumentFilename, assoc.EntityID, assoc.EntityType,
	).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check for existing document association: %v", err)
	}

	if count > 0 {
		// This is not necessarily an error in a transaction, as we might be
		// creating multiple associations in bulk and want to skip duplicates
		return nil
	}

	// Generate UUID if not provided
	if assoc.ID == "" {
		assoc.ID = uuid.New().String()
	}

	// Set timestamp if not provided
	if assoc.CreatedAt.IsZero() {
		assoc.CreatedAt = time.Now()
	}

	// Create new association
	query := `
		INSERT INTO document_associations (
			id, document_filename, entity_id, entity_type, created_at
		) VALUES (?, ?, ?, ?, ?)
	`

	_, err = tx.Exec(
		query,
		assoc.ID,
		assoc.DocumentFilename,
		assoc.EntityID,
		assoc.EntityType,
		assoc.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create document association: %v", err)
	}

	return nil
}

// GetDocumentAssociation retrieves a document association by ID
func GetDocumentAssociation(db *sql.DB, id string) (*DocumentAssociation, error) {
	query := `
		SELECT id, document_filename, entity_id, entity_type, created_at
		FROM document_associations
		WHERE id = ?
	`

	assoc := &DocumentAssociation{}
	err := db.QueryRow(query, id).Scan(
		&assoc.ID,
		&assoc.DocumentFilename,
		&assoc.EntityID,
		&assoc.EntityType,
		&assoc.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get document association: %v", err)
	}

	return assoc, nil
}

// GetDocumentAssociationsByEntity retrieves all document associations for a specific entity
func GetDocumentAssociationsByEntity(db *sql.DB, entityType, entityID string) ([]*DocumentAssociation, int, error) {
	query := `
		SELECT id, document_filename, entity_id, entity_type, created_at
		FROM document_associations
		WHERE entity_type = ? AND entity_id = ?
	`

	// Count total
	var total int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM document_associations WHERE entity_type = ? AND entity_id = ?",
		entityType, entityID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count document associations: %v", err)
	}

	// Execute query
	rows, err := db.Query(query, entityType, entityID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query document associations: %v", err)
	}
	defer rows.Close()

	// Process results
	associations := []*DocumentAssociation{}
	for rows.Next() {
		assoc := &DocumentAssociation{}

		err := rows.Scan(
			&assoc.ID,
			&assoc.DocumentFilename,
			&assoc.EntityID,
			&assoc.EntityType,
			&assoc.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan document association row: %v", err)
		}

		associations = append(associations, assoc)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating document association rows: %v", err)
	}

	return associations, total, nil
}

// ListDocumentAssociations retrieves a paginated list of document associations
func ListDocumentAssociations(db *sql.DB, limit, offset int) ([]*DocumentAssociation, int, error) {
	query := `
		SELECT id, document_filename, entity_id, entity_type, created_at
		FROM document_associations
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	// Count total
	var total int
	err := db.QueryRow("SELECT COUNT(*) FROM document_associations").Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count document associations: %v", err)
	}

	// Execute query
	rows, err := db.Query(query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query document associations: %v", err)
	}
	defer rows.Close()

	// Process results
	associations := []*DocumentAssociation{}
	for rows.Next() {
		assoc := &DocumentAssociation{}

		err := rows.Scan(
			&assoc.ID,
			&assoc.DocumentFilename,
			&assoc.EntityID,
			&assoc.EntityType,
			&assoc.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan document association row: %v", err)
		}

		associations = append(associations, assoc)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating document association rows: %v", err)
	}

	return associations, total, nil
}

// GetAllAssociationsForDocument retrieves all associations for a specific document
func GetAllAssociationsForDocument(db *sql.DB, filename string) ([]*DocumentAssociation, error) {
	query := `
		SELECT id, document_filename, entity_id, entity_type, created_at
		FROM document_associations
		WHERE document_filename = ?
	`

	rows, err := db.Query(query, filename)
	if err != nil {
		return nil, fmt.Errorf("failed to query document associations: %v", err)
	}
	defer rows.Close()

	// Process results
	associations := []*DocumentAssociation{}
	for rows.Next() {
		assoc := &DocumentAssociation{}

		err := rows.Scan(
			&assoc.ID,
			&assoc.DocumentFilename,
			&assoc.EntityID,
			&assoc.EntityType,
			&assoc.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan document association row: %v", err)
		}

		associations = append(associations, assoc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating document association rows: %v", err)
	}

	return associations, nil
}

// DeleteDocumentAssociation deletes a document association by ID
func DeleteDocumentAssociation(db *sql.DB, id string) error {
	query := "DELETE FROM document_associations WHERE id = ?"

	result, err := db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete document association: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// DeleteAllDocumentAssociationsByFilename deletes all associations for a document
func DeleteAllDocumentAssociationsByFilename(db *sql.DB, filename string) error {
	query := "DELETE FROM document_associations WHERE document_filename = ?"

	_, err := db.Exec(query, filename)
	if err != nil {
		return fmt.Errorf("failed to delete document associations: %v", err)
	}

	return nil
}

// DeleteAllDocumentAssociationsByFilenameTx deletes all associations for a document within a transaction
func DeleteAllDocumentAssociationsByFilenameTx(tx *sql.Tx, filename string) error {
	query := "DELETE FROM document_associations WHERE document_filename = ?"

	_, err := tx.Exec(query, filename)
	if err != nil {
		return fmt.Errorf("failed to delete document associations: %v", err)
	}

	return nil
}

// CreateAPIUserAccess inserts a new API user access record
func CreateAPIUserAccess(db *sql.DB, access *APIUserAccess) error {
	// Generate UUID if not provided
	if access.ID == "" {
		access.ID = uuid.New().String()
	}

	// Set timestamp
	if access.GrantedAt.IsZero() {
		access.GrantedAt = time.Now()
	}

	query := `
		INSERT INTO api_user_access (id, api_id, external_user_id, access_level, granted_at, granted_by, revoked_at, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := db.Exec(
		query,
		access.ID,
		access.APIID,
		access.ExternalUserID,
		access.AccessLevel,
		access.GrantedAt,
		access.GrantedBy,
		access.RevokedAt,
		access.IsActive,
	)

	return err
}

// CreateAPIUserAccessTx inserts a new API user access record within a transaction
func CreateAPIUserAccessTx(tx *sql.Tx, access *APIUserAccess) error {
	// Generate UUID if not provided
	if access.ID == "" {
		access.ID = uuid.New().String()
	}

	// Set timestamp
	if access.GrantedAt.IsZero() {
		access.GrantedAt = time.Now()
	}

	query := `
		INSERT INTO api_user_access (id, api_id, external_user_id, access_level, granted_at, granted_by, revoked_at, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := tx.Exec(
		query,
		access.ID,
		access.APIID,
		access.ExternalUserID,
		access.AccessLevel,
		access.GrantedAt,
		access.GrantedBy,
		access.RevokedAt,
		access.IsActive,
	)

	return err
}

// GetPolicy retrieves a policy by ID
func GetPolicy(db *sql.DB, id string) (*Policy, error) {
	query := `
		SELECT id, name, description, type, is_active, created_at, updated_at, created_by
		FROM policies
		WHERE id = ?
	`

	policy := &Policy{}
	var createdBy sql.NullString
	var description sql.NullString

	err := db.QueryRow(query, id).Scan(
		&policy.ID,
		&policy.Name,
		&description,
		&policy.Type,
		&policy.IsActive,
		&policy.CreatedAt,
		&policy.UpdatedAt,
		&createdBy,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Handle nullable fields
	if description.Valid {
		policy.Description = description.String
	}

	if createdBy.Valid {
		policy.CreatedBy = createdBy.String
	}

	return policy, nil
}

// GetPolicyWithRules retrieves a policy with its rules by ID
func GetPolicyWithRules(db *sql.DB, id string) (*Policy, error) {
	// Get the policy first
	policy, err := GetPolicy(db, id)
	if err != nil {
		return nil, err
	}

	// Then get the rules
	query := `
		SELECT id, policy_id, rule_type, limit_value, period, action, priority, created_at
		FROM policy_rules
		WHERE policy_id = ?
		ORDER BY priority
	`

	rows, err := db.Query(query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to query policy rules: %v", err)
	}
	defer rows.Close()

	rules := []PolicyRule{}
	for rows.Next() {
		rule := PolicyRule{}
		var period sql.NullString
		var limitValue sql.NullFloat64

		err := rows.Scan(
			&rule.ID,
			&rule.PolicyID,
			&rule.RuleType,
			&limitValue,
			&period,
			&rule.Action,
			&rule.Priority,
			&rule.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan policy rule row: %v", err)
		}

		if period.Valid {
			rule.Period = period.String
		}

		if limitValue.Valid {
			rule.LimitValue = limitValue.Float64
		}

		rules = append(rules, rule)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating policy rule rows: %v", err)
	}

	policy.Rules = rules

	return policy, nil
}

// CreatePolicyChange records a policy change
func CreatePolicyChange(db *sql.DB, change *PolicyChange) error {
	// Generate UUID if not provided
	if change.ID == "" {
		change.ID = uuid.New().String()
	}

	// Set timestamp
	if change.ChangedAt.IsZero() {
		change.ChangedAt = time.Now()
	}

	query := `
		INSERT INTO policy_changes (id, api_id, old_policy_id, new_policy_id, changed_at, changed_by, effective_date, change_reason)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := db.Exec(
		query,
		change.ID,
		change.APIID,
		change.OldPolicyID,
		change.NewPolicyID,
		change.ChangedAt,
		change.ChangedBy,
		change.EffectiveDate,
		change.ChangeReason,
	)

	return err
}

// CreatePolicyChangeTx records a policy change within a transaction
func CreatePolicyChangeTx(tx *sql.Tx, change *PolicyChange) error {
	// Generate UUID if not provided
	if change.ID == "" {
		change.ID = uuid.New().String()
	}

	// Set timestamp
	if change.ChangedAt.IsZero() {
		change.ChangedAt = time.Now()
	}

	query := `
		INSERT INTO policy_changes (id, api_id, old_policy_id, new_policy_id, changed_at, changed_by, effective_date, change_reason)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := tx.Exec(
		query,
		change.ID,
		change.APIID,
		change.OldPolicyID,
		change.NewPolicyID,
		change.ChangedAt,
		change.ChangedBy,
		change.EffectiveDate,
		change.ChangeReason,
	)

	return err
}

// GetAPIUsageSummaryByPeriod retrieves usage summary for an API by period
func GetAPIUsageSummaryByPeriod(db *sql.DB, apiID, periodType, periodValue string) (*APIUsageSummary, error) {
	query := `
		SELECT id, api_id, external_user_id, period_type, period_start, period_end, 
			total_requests, total_tokens, total_credits, total_time_ms, 
			throttled_requests, blocked_requests, last_updated
		FROM api_usage_summary
		WHERE api_id = ? AND period_type = ? AND period_start = ?
	`

	var periodStart time.Time
	var err error

	// Parse period value based on period type
	switch periodType {
	case "daily":
		periodStart, err = time.Parse("2006-01-02", periodValue)
	case "monthly":
		periodStart, err = time.Parse("2006-01", periodValue)
		// Add logic for other period types as needed
	}

	if err != nil {
		return nil, fmt.Errorf("invalid period value for %s: %v", periodType, err)
	}

	summary := &APIUsageSummary{}
	var externalUserID sql.NullString

	err = db.QueryRow(query, apiID, periodType, periodStart).Scan(
		&summary.ID,
		&summary.APIID,
		&externalUserID,
		&summary.PeriodType,
		&summary.PeriodStart,
		&summary.PeriodEnd,
		&summary.TotalRequests,
		&summary.TotalTokens,
		&summary.TotalCredits,
		&summary.TotalTimeMs,
		&summary.ThrottledRequests,
		&summary.BlockedRequests,
		&summary.LastUpdated,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Handle nullable fields
	if externalUserID.Valid {
		summary.ExternalUserID = externalUserID.String
	}

	return summary, nil
}

// CreateAPIRequest inserts a new API request record
func CreateAPIRequest(db *sql.DB, request *APIRequest) error {
	// Generate UUID if not provided
	if request.ID == "" {
		request.ID = uuid.New().String()
	}

	// Set timestamps
	if request.SubmittedDate.IsZero() {
		request.SubmittedDate = time.Now()
	}

	query := `
		INSERT INTO api_requests (
			id, api_name, description, submitted_date, status,
			requester_id, denial_reason, denied_date, approved_date,
			submission_count, previous_request_id, proposed_policy_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := db.Exec(
		query,
		request.ID,
		request.APIName,
		request.Description,
		request.SubmittedDate,
		request.Status,
		request.RequesterID,
		request.DenialReason,
		request.DeniedDate,
		request.ApprovedDate,
		request.SubmissionCount,
		request.PreviousRequestID,
		request.ProposedPolicyID,
	)

	return err
}

// CreateAPIRequestTx inserts a new API request within a transaction
func CreateAPIRequestTx(tx *sql.Tx, req *APIRequest) error {
	// Generate UUID if not provided
	if req.ID == "" {
		req.ID = uuid.New().String()
	}

	// Set timestamp
	if req.SubmittedDate.IsZero() {
		req.SubmittedDate = time.Now()
	}

	query := `
		INSERT INTO api_requests (
			id, api_name, description, submitted_date, status, requester_id,
			denial_reason, denied_date, approved_date, submission_count,
			previous_request_id, proposed_policy_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := tx.Exec(
		query,
		req.ID,
		req.APIName,
		req.Description,
		req.SubmittedDate,
		req.Status,
		req.RequesterID,
		req.DenialReason,
		req.DeniedDate,
		req.ApprovedDate,
		req.SubmissionCount,
		req.PreviousRequestID,
		req.ProposedPolicyID,
	)

	if err != nil {
		return fmt.Errorf("failed to create API request: %v", err)
	}

	return nil
}

// GetAPIRequest retrieves a single API request by ID
func GetAPIRequest(db *sql.DB, id string) (*APIRequest, error) {
	query := "SELECT id, api_name, description, submitted_date, status, requester_id, denial_reason, " +
		"denied_date, approved_date, submission_count, previous_request_id, proposed_policy_id " +
		"FROM api_requests WHERE id = ?"

	req := &APIRequest{}
	var deniedDate, approvedDate sql.NullTime
	var denialReason sql.NullString
	var previousRequestID, proposedPolicyID sql.NullString

	err := db.QueryRow(query, id).Scan(
		&req.ID,
		&req.APIName,
		&req.Description,
		&req.SubmittedDate,
		&req.Status,
		&req.RequesterID,
		&denialReason,
		&deniedDate,
		&approvedDate,
		&req.SubmissionCount,
		&previousRequestID,
		&proposedPolicyID,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get API request: %v", err)
	}

	if denialReason.Valid {
		req.DenialReason = denialReason.String
	}

	if deniedDate.Valid {
		req.DeniedDate = &deniedDate.Time
	}

	if approvedDate.Valid {
		req.ApprovedDate = &approvedDate.Time
	}

	if previousRequestID.Valid {
		req.PreviousRequestID = &previousRequestID.String
	}

	if proposedPolicyID.Valid {
		req.ProposedPolicyID = &proposedPolicyID.String
	}

	return req, nil
}

// GetAPIRequestTx retrieves a single API request by ID within a transaction
func GetAPIRequestTx(tx *sql.Tx, id string) (*APIRequest, error) {
	query := "SELECT id, api_name, description, submitted_date, status, requester_id, denial_reason, " +
		"denied_date, approved_date, submission_count, previous_request_id, proposed_policy_id " +
		"FROM api_requests WHERE id = ?"

	req := &APIRequest{}
	var deniedDate, approvedDate sql.NullTime
	var denialReason sql.NullString
	var previousRequestID, proposedPolicyID sql.NullString

	err := tx.QueryRow(query, id).Scan(
		&req.ID,
		&req.APIName,
		&req.Description,
		&req.SubmittedDate,
		&req.Status,
		&req.RequesterID,
		&denialReason,
		&deniedDate,
		&approvedDate,
		&req.SubmissionCount,
		&previousRequestID,
		&proposedPolicyID,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get API request: %v", err)
	}

	if denialReason.Valid {
		req.DenialReason = denialReason.String
	}

	if deniedDate.Valid {
		req.DeniedDate = &deniedDate.Time
	}

	if approvedDate.Valid {
		req.ApprovedDate = &approvedDate.Time
	}

	if previousRequestID.Valid {
		req.PreviousRequestID = &previousRequestID.String
	}

	if proposedPolicyID.Valid {
		req.ProposedPolicyID = &proposedPolicyID.String
	}

	return req, nil
}

// UpdateAPIRequest updates an existing API request
func UpdateAPIRequest(db *sql.DB, req *APIRequest) error {
	query := `
		UPDATE api_requests
		SET api_name = ?, description = ?, status = ?,
		    denial_reason = ?, denied_date = ?, approved_date = ?,
		    submission_count = ?, previous_request_id = ?, proposed_policy_id = ?
		WHERE id = ?
	`

	result, err := db.Exec(
		query,
		req.APIName,
		req.Description,
		req.Status,
		req.DenialReason,
		req.DeniedDate,
		req.ApprovedDate,
		req.SubmissionCount,
		req.PreviousRequestID,
		req.ProposedPolicyID,
		req.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update API request: %v", err)
	}

	// Check if any row was affected
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// UpdateAPIRequestTx updates an existing API request within a transaction
func UpdateAPIRequestTx(tx *sql.Tx, req *APIRequest) error {
	query := `
		UPDATE api_requests
		SET api_name = ?, description = ?, status = ?,
		    denial_reason = ?, denied_date = ?, approved_date = ?,
		    submission_count = ?, previous_request_id = ?, proposed_policy_id = ?
		WHERE id = ?
	`

	result, err := tx.Exec(
		query,
		req.APIName,
		req.Description,
		req.Status,
		req.DenialReason,
		req.DeniedDate,
		req.ApprovedDate,
		req.SubmissionCount,
		req.PreviousRequestID,
		req.ProposedPolicyID,
		req.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update API request: %v", err)
	}

	// Check if any row was affected
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// ListAPIRequests retrieves a paginated, filtered list of API requests
func ListAPIRequests(db *sql.DB, status, requesterID, hostUserID string, limit, offset int, sort, order string) ([]*APIRequest, int, error) {
	// Build the query based on filters
	query := "SELECT id, api_name, description, submitted_date, status, requester_id, denial_reason, " +
		"denied_date, approved_date, submission_count, previous_request_id, proposed_policy_id " +
		"FROM api_requests WHERE 1=1"
	countQuery := "SELECT COUNT(*) FROM api_requests WHERE 1=1"

	args := []interface{}{}

	// Apply status filter
	if status != "" {
		query += " AND status = ?"
		countQuery += " AND status = ?"
		args = append(args, status)
	}

	// Apply requester filter
	if requesterID != "" {
		query += " AND requester_id = ?"
		countQuery += " AND requester_id = ?"
		args = append(args, requesterID)
	}

	// Apply sorting
	query += " ORDER BY "
	if sort == "api_name" {
		query += "api_name"
	} else {
		query += "submitted_date" // default
	}

	if order == "asc" {
		query += " ASC"
	} else {
		query += " DESC" // default
	}

	// Apply pagination
	query += " LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	// Get total count
	var total int
	err := db.QueryRow(countQuery, args[:len(args)-2]...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count API requests: %v", err)
	}

	// Execute the query
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query API requests: %v", err)
	}
	defer rows.Close()

	// Process results
	requests := []*APIRequest{}
	for rows.Next() {
		req := &APIRequest{}
		var deniedDate, approvedDate sql.NullTime
		var denialReason sql.NullString
		var previousRequestID, proposedPolicyID sql.NullString

		err := rows.Scan(
			&req.ID,
			&req.APIName,
			&req.Description,
			&req.SubmittedDate,
			&req.Status,
			&req.RequesterID,
			&denialReason,
			&deniedDate,
			&approvedDate,
			&req.SubmissionCount,
			&previousRequestID,
			&proposedPolicyID,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan API request row: %v", err)
		}

		if denialReason.Valid {
			req.DenialReason = denialReason.String
		}

		if deniedDate.Valid {
			req.DeniedDate = &deniedDate.Time
		}

		if approvedDate.Valid {
			req.ApprovedDate = &approvedDate.Time
		}

		if previousRequestID.Valid {
			req.PreviousRequestID = &previousRequestID.String
		}

		if proposedPolicyID.Valid {
			req.ProposedPolicyID = &proposedPolicyID.String
		}

		requests = append(requests, req)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating API request rows: %v", err)
	}

	return requests, total, nil
}

// CountRequestDocuments counts the documents associated with a request
func CountRequestDocuments(db *sql.DB, requestID string) (int, error) {
	query := "SELECT COUNT(*) FROM document_associations WHERE entity_id = ? AND entity_type = 'request'"

	var count int
	err := db.QueryRow(query, requestID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count request documents: %v", err)
	}

	return count, nil
}

// GetRequestDocuments retrieves all documents associated with a request
func GetRequestDocuments(db *sql.DB, requestID string) ([]*DocumentAssociation, error) {
	query := "SELECT id, document_filename, entity_id, entity_type, created_at " +
		"FROM document_associations WHERE entity_id = ? AND entity_type = 'request'"

	rows, err := db.Query(query, requestID)
	if err != nil {
		return nil, fmt.Errorf("failed to query request documents: %v", err)
	}
	defer rows.Close()

	documents := []*DocumentAssociation{}
	for rows.Next() {
		doc := &DocumentAssociation{}

		err := rows.Scan(
			&doc.ID,
			&doc.DocumentFilename,
			&doc.EntityID,
			&doc.EntityType,
			&doc.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan document row: %v", err)
		}

		documents = append(documents, doc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating document rows: %v", err)
	}

	return documents, nil
}

// CountRequestTrackers counts the trackers associated with a request
func CountRequestTrackers(db *sql.DB, requestID string) (int, error) {
	query := "SELECT COUNT(*) FROM request_required_trackers WHERE request_id = ?"

	var count int
	err := db.QueryRow(query, requestID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count request trackers: %v", err)
	}

	return count, nil
}

// RequestTrackerWithName represents a tracker with its name and description
type RequestTrackerWithName struct {
	TrackerID   string
	Name        string
	Description string
}

// GetRequestTrackers retrieves all trackers associated with a request
func GetRequestTrackers(db *sql.DB, requestID string) ([]*RequestTrackerWithName, error) {
	query := "SELECT rrt.tracker_id, t.name, t.description " +
		"FROM request_required_trackers rrt " +
		"JOIN trackers t ON rrt.tracker_id = t.id " +
		"WHERE rrt.request_id = ?"

	rows, err := db.Query(query, requestID)
	if err != nil {
		return nil, fmt.Errorf("failed to query request trackers: %v", err)
	}
	defer rows.Close()

	trackers := []*RequestTrackerWithName{}
	for rows.Next() {
		tracker := &RequestTrackerWithName{}
		var description sql.NullString

		err := rows.Scan(
			&tracker.TrackerID,
			&tracker.Name,
			&description,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tracker row: %v", err)
		}

		if description.Valid {
			tracker.Description = description.String
		}

		trackers = append(trackers, tracker)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tracker rows: %v", err)
	}

	return trackers, nil
}

// CreateRequestTracker creates a new request-tracker association
func CreateRequestTracker(db *sql.DB, assoc *RequestRequiredTracker) error {
	// Generate UUID if not provided
	if assoc.ID == "" {
		assoc.ID = uuid.New().String()
	}

	query := "INSERT INTO request_required_trackers (id, request_id, tracker_id) VALUES (?, ?, ?)"

	_, err := db.Exec(query, assoc.ID, assoc.RequestID, assoc.TrackerID)
	if err != nil {
		return fmt.Errorf("failed to create request-tracker association: %v", err)
	}

	return nil
}

// CreateRequestTrackerTx creates a new request-tracker association within a transaction
func CreateRequestTrackerTx(tx *sql.Tx, assoc *RequestRequiredTracker) error {
	// Generate UUID if not provided
	if assoc.ID == "" {
		assoc.ID = uuid.New().String()
	}

	query := "INSERT INTO request_required_trackers (id, request_id, tracker_id) VALUES (?, ?, ?)"

	_, err := tx.Exec(query, assoc.ID, assoc.RequestID, assoc.TrackerID)
	if err != nil {
		return fmt.Errorf("failed to create request-tracker association: %v", err)
	}

	return nil
}

// CopyDocumentsFromRequest copies document associations from one request to another
func CopyDocumentsFromRequest(tx *sql.Tx, sourceID string, targetID string) error {
	query := "SELECT document_filename FROM document_associations WHERE entity_id = ? AND entity_type = 'request'"

	rows, err := tx.Query(query, sourceID)
	if err != nil {
		return fmt.Errorf("failed to query source documents: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var filename string

		err := rows.Scan(&filename)
		if err != nil {
			return fmt.Errorf("failed to scan document filename: %v", err)
		}

		// Create a new association for another request
		association := &DocumentAssociation{
			ID:               uuid.New().String(),
			DocumentFilename: filename,
			EntityID:         targetID,
			EntityType:       "request", // When copying between requests, use request type
			CreatedAt:        time.Now(),
		}

		if err := CreateDocumentAssociationTx(tx, association); err != nil {
			return fmt.Errorf("failed to create document association: %v", err)
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating document rows: %v", err)
	}

	return nil
}

// CopyDocumentsFromRequestToAPI copies document associations from a request to an API
func CopyDocumentsFromRequestToAPI(tx *sql.Tx, requestID string, apiID string) error {
	query := "SELECT document_filename FROM document_associations WHERE entity_id = ? AND entity_type = 'request'"

	rows, err := tx.Query(query, requestID)
	if err != nil {
		return fmt.Errorf("failed to query source documents: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var filename string

		err := rows.Scan(&filename)
		if err != nil {
			return fmt.Errorf("failed to scan document filename: %v", err)
		}

		// Create a new association for an API
		association := &DocumentAssociation{
			ID:               uuid.New().String(),
			DocumentFilename: filename,
			EntityID:         apiID,
			EntityType:       "api", // When copying to an API, use api type
			CreatedAt:        time.Now(),
		}

		if err := CreateDocumentAssociationTx(tx, association); err != nil {
			return fmt.Errorf("failed to create document association: %v", err)
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating document rows: %v", err)
	}

	return nil
}

// CopyTrackersFromRequest copies tracker associations from one request to another
func CopyTrackersFromRequest(tx *sql.Tx, sourceID string, targetID string) error {
	query := "SELECT tracker_id FROM request_required_trackers WHERE request_id = ?"

	rows, err := tx.Query(query, sourceID)
	if err != nil {
		return fmt.Errorf("failed to query source trackers: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var trackerID string

		err := rows.Scan(&trackerID)
		if err != nil {
			return fmt.Errorf("failed to scan tracker ID: %v", err)
		}

		// Create a new association
		association := &RequestRequiredTracker{
			ID:        uuid.New().String(),
			RequestID: targetID,
			TrackerID: trackerID,
		}

		if err := CreateRequestTrackerTx(tx, association); err != nil {
			return fmt.Errorf("failed to create tracker association: %v", err)
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating tracker rows: %v", err)
	}

	return nil
}

// GetTracker retrieves a tracker by ID
func GetTracker(tx *sql.Tx, id string) (*Tracker, error) {
	query := "SELECT id, name, description, is_active, created_at FROM trackers WHERE id = ?"

	tracker := &Tracker{}
	var description sql.NullString

	err := tx.QueryRow(query, id).Scan(
		&tracker.ID,
		&tracker.Name,
		&description,
		&tracker.IsActive,
		&tracker.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get tracker: %v", err)
	}

	if description.Valid {
		tracker.Description = description.String
	}

	return tracker, nil
}

// Helper function to generate a secure API key
func generateAPIKey() (string, error) {
	// Example implementation using UUID as the base
	key := fmt.Sprintf("api_%s", uuid.New().String())
	return key, nil
}

// ListPolicies retrieves a filtered, paginated list of policies
func ListPolicies(db *sql.DB, policyType string, activeOnly bool, createdBy string, limit, offset int, sort, order string) ([]*Policy, int, error) {
	// Build query
	baseQuery := "FROM policies WHERE 1=1"
	args := []interface{}{}

	if policyType != "" {
		baseQuery += " AND type = ?"
		args = append(args, policyType)
	}

	if activeOnly {
		baseQuery += " AND is_active = TRUE"
	}

	if createdBy != "" && createdBy != "local-user" {
		baseQuery += " AND created_by = ?"
		args = append(args, createdBy)
	}

	// Count total
	var total int
	countQuery := "SELECT COUNT(*) " + baseQuery
	err := db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count policies: %v", err)
	}

	// Build main query
	query := `
		SELECT id, name, description, type, is_active,
		       created_at, updated_at, created_by
		` + baseQuery + `
		ORDER BY ` + sort + ` ` + order + `
		LIMIT ? OFFSET ?
	`
	args = append(args, limit, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query policies: %v", err)
	}
	defer rows.Close()

	// Process results
	policies := []*Policy{}
	for rows.Next() {
		policy := &Policy{}
		var createdBy sql.NullString
		var description sql.NullString

		err := rows.Scan(
			&policy.ID,
			&policy.Name,
			&description,
			&policy.Type,
			&policy.IsActive,
			&policy.CreatedAt,
			&policy.UpdatedAt,
			&createdBy,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan policy row: %v", err)
		}

		if description.Valid {
			policy.Description = description.String
		}

		if createdBy.Valid {
			policy.CreatedBy = createdBy.String
		}

		policies = append(policies, policy)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating policy rows: %v", err)
	}

	return policies, total, nil
}

// UpdatePolicy updates an existing policy
func UpdatePolicy(db *sql.DB, policy *Policy) error {
	query := `
		UPDATE policies
		SET name = ?, description = ?, is_active = ?,
		    updated_at = ?
		WHERE id = ?
	`

	result, err := db.Exec(
		query,
		policy.Name,
		policy.Description,
		policy.IsActive,
		policy.UpdatedAt,
		policy.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update policy: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// UpdatePolicyTx updates an existing policy within a transaction
func UpdatePolicyTx(tx *sql.Tx, policy *Policy) error {
	query := `
		UPDATE policies
		SET name = ?, description = ?, is_active = ?,
		    updated_at = ?
		WHERE id = ?
	`

	result, err := tx.Exec(
		query,
		policy.Name,
		policy.Description,
		policy.IsActive,
		policy.UpdatedAt,
		policy.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update policy: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// DeletePolicy permanently deletes a policy
func DeletePolicy(db *sql.DB, id string) error {
	query := "DELETE FROM policies WHERE id = ?"

	result, err := db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete policy: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// CreatePolicyRule creates a new policy rule
func CreatePolicyRule(db *sql.DB, rule *PolicyRule) error {
	query := `
		INSERT INTO policy_rules (
			id, policy_id, rule_type, limit_value, period,
			action, priority, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := db.Exec(
		query,
		rule.ID,
		rule.PolicyID,
		rule.RuleType,
		rule.LimitValue,
		rule.Period,
		rule.Action,
		rule.Priority,
		rule.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create policy rule: %v", err)
	}

	return nil
}

// CreatePolicyRuleTx creates a new policy rule within a transaction
func CreatePolicyRuleTx(tx *sql.Tx, rule *PolicyRule) error {
	query := `
		INSERT INTO policy_rules (
			id, policy_id, rule_type, limit_value, period,
			action, priority, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := tx.Exec(
		query,
		rule.ID,
		rule.PolicyID,
		rule.RuleType,
		rule.LimitValue,
		rule.Period,
		rule.Action,
		rule.Priority,
		rule.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create policy rule: %v", err)
	}

	return nil
}

// GetPolicyRules retrieves all rules for a policy
func GetPolicyRules(db *sql.DB, policyID string) ([]PolicyRule, error) {
	query := `
		SELECT id, policy_id, rule_type, limit_value, period,
		       action, priority, created_at
		FROM policy_rules
		WHERE policy_id = ?
		ORDER BY priority ASC
	`

	rows, err := db.Query(query, policyID)
	if err != nil {
		return nil, fmt.Errorf("failed to query policy rules: %v", err)
	}
	defer rows.Close()

	// Process results
	rules := []PolicyRule{}
	for rows.Next() {
		rule := PolicyRule{}
		var period sql.NullString
		var limitValue sql.NullFloat64

		err := rows.Scan(
			&rule.ID,
			&rule.PolicyID,
			&rule.RuleType,
			&limitValue,
			&period,
			&rule.Action,
			&rule.Priority,
			&rule.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan policy rule row: %v", err)
		}

		if period.Valid {
			rule.Period = period.String
		}

		if limitValue.Valid {
			rule.LimitValue = limitValue.Float64
		}

		rules = append(rules, rule)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating policy rule rows: %v", err)
	}

	return rules, nil
}

// DeletePolicyRules deletes all rules for a policy
func DeletePolicyRules(db *sql.DB, policyID string) error {
	query := "DELETE FROM policy_rules WHERE policy_id = ?"

	_, err := db.Exec(query, policyID)
	if err != nil {
		return fmt.Errorf("failed to delete policy rules: %v", err)
	}

	return nil
}

// DeletePolicyRulesTx deletes all rules for a policy within a transaction
func DeletePolicyRulesTx(tx *sql.Tx, policyID string) error {
	query := "DELETE FROM policy_rules WHERE policy_id = ?"

	_, err := tx.Exec(query, policyID)
	if err != nil {
		return fmt.Errorf("failed to delete policy rules: %v", err)
	}

	return nil
}

// GetPolicyChangeHistory retrieves the policy change history for an API
func GetPolicyChangeHistory(db *sql.DB, apiID string) ([]*PolicyChange, error) {
	query := `
		SELECT id, api_id, old_policy_id, new_policy_id,
		       changed_at, changed_by, effective_date, change_reason
		FROM policy_changes
		WHERE api_id = ?
		ORDER BY changed_at DESC
	`

	rows, err := db.Query(query, apiID)
	if err != nil {
		return nil, fmt.Errorf("failed to query policy changes: %v", err)
	}
	defer rows.Close()

	// Process results
	changes := []*PolicyChange{}
	for rows.Next() {
		change := &PolicyChange{}
		var oldPolicyID, newPolicyID sql.NullString
		var effectiveDate sql.NullTime
		var changedBy, changeReason sql.NullString

		err := rows.Scan(
			&change.ID,
			&change.APIID,
			&oldPolicyID,
			&newPolicyID,
			&change.ChangedAt,
			&changedBy,
			&effectiveDate,
			&changeReason,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan policy change row: %v", err)
		}

		if oldPolicyID.Valid {
			oldID := oldPolicyID.String
			change.OldPolicyID = &oldID
		}

		if newPolicyID.Valid {
			newID := newPolicyID.String
			change.NewPolicyID = &newID
		}

		if changedBy.Valid {
			change.ChangedBy = changedBy.String
		}

		if effectiveDate.Valid {
			date := effectiveDate.Time
			change.EffectiveDate = &date
		}

		if changeReason.Valid {
			change.ChangeReason = changeReason.String
		}

		changes = append(changes, change)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating policy change rows: %v", err)
	}

	return changes, nil
}

// GetPendingPolicyChanges retrieves pending policy changes that need to be applied
func GetPendingPolicyChanges(db *sql.DB) ([]*PolicyChange, error) {
	query := `
		SELECT pc.id, pc.api_id, pc.old_policy_id, pc.new_policy_id,
		       pc.changed_at, pc.changed_by, pc.effective_date, pc.change_reason
		FROM policy_changes pc
		JOIN apis a ON pc.api_id = a.id
		WHERE
		    pc.effective_date <= ?
		    AND pc.new_policy_id IS NOT NULL
		    AND (a.policy_id IS NULL OR a.policy_id != pc.new_policy_id)
		ORDER BY pc.effective_date ASC
	`

	now := time.Now()

	rows, err := db.Query(query, now)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending policy changes: %v", err)
	}
	defer rows.Close()

	// Process results
	changes := []*PolicyChange{}
	for rows.Next() {
		change := &PolicyChange{}
		var oldPolicyID, newPolicyID sql.NullString
		var effectiveDate sql.NullTime
		var changedBy, changeReason sql.NullString

		err := rows.Scan(
			&change.ID,
			&change.APIID,
			&oldPolicyID,
			&newPolicyID,
			&change.ChangedAt,
			&changedBy,
			&effectiveDate,
			&changeReason,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan policy change row: %v", err)
		}

		if oldPolicyID.Valid {
			oldID := oldPolicyID.String
			change.OldPolicyID = &oldID
		}

		if newPolicyID.Valid {
			newID := newPolicyID.String
			change.NewPolicyID = &newID
		}

		if changedBy.Valid {
			change.ChangedBy = changedBy.String
		}

		if effectiveDate.Valid {
			date := effectiveDate.Time
			change.EffectiveDate = &date
		}

		if changeReason.Valid {
			change.ChangeReason = changeReason.String
		}

		changes = append(changes, change)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating policy change rows: %v", err)
	}

	return changes, nil
}

// ApplyPendingPolicyChange applies a pending policy change
func ApplyPendingPolicyChange(db *sql.DB, change *PolicyChange) error {
	if change.NewPolicyID == nil {
		return fmt.Errorf("cannot apply change without a new policy ID")
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %v", err)
	}
	defer tx.Rollback() // Will be a no-op if transaction succeeds

	// Update API policy
	query := `
		UPDATE apis
		SET policy_id = ?, updated_at = ?
		WHERE id = ?
	`

	now := time.Now()

	_, err = tx.Exec(query, *change.NewPolicyID, now, change.APIID)
	if err != nil {
		return fmt.Errorf("failed to update API policy: %v", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	return nil
}
