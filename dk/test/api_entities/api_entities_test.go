package api_entities_test

import (
	"bytes"
	"context"
	"database/sql"
	"dk/db"
	httpPkg "dk/http"
	"dk/test/api_entities"
	"encoding/json"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

// setupTestDB creates an in-memory database for testing
func setupTestDB(t *testing.T) *sql.DB {
	// Use in-memory SQLite database for testing
	dsn := ":memory:?_busy_timeout=5000&_journal_mode=DELETE&cache=shared"
	database, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}

	// Set pragmas for better performance and reliability
	pragmas := []string{
		"PRAGMA busy_timeout = 5000;",
		"PRAGMA cache_size = 1000;",
		"PRAGMA foreign_keys = ON;",
		"PRAGMA synchronous = NORMAL;",
	}

	for _, pragma := range pragmas {
		if _, err := database.Exec(pragma); err != nil {
			t.Fatalf("Failed to set pragma (%s): %v", pragma, err)
		}
	}

	// Run migrations
	if err := db.RunAPIMigrations(database); err != nil {
		t.Fatalf("Failed to run API management migrations: %v", err)
	}

	return database
}

// insertTestData adds test policies and APIs to the database
func insertTestData(t *testing.T, database *sql.DB) ([]string, []string) {
	// Create policies
	policyIDs := make([]string, 3)
	for i := 0; i < 3; i++ {
		policyID := uuid.New().String()
		policyIDs[i] = policyID

		_, err := database.Exec(`
			INSERT INTO policies (id, name, description, type, is_active, created_at, updated_at, created_by)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			policyID, "Test Policy "+strconv.Itoa(i+1), "Policy for testing "+strconv.Itoa(i+1),
			"free", true, time.Now(), time.Now(), "test_user")

		if err != nil {
			t.Fatalf("Failed to insert test policy: %v", err)
		}
	}

	// Create APIs
	apiIDs := make([]string, 5)
	apiStatuses := []struct {
		isActive     bool
		isDeprecated bool
	}{
		{true, false},  // active
		{true, false},  // active
		{false, false}, // inactive
		{true, true},   // deprecated
		{false, true},  // inactive and deprecated
	}

	for i := 0; i < 5; i++ {
		apiID := uuid.New().String()
		apiIDs[i] = apiID

		// Assign different policies cyclically
		policyIndex := i % len(policyIDs)
		policyID := policyIDs[policyIndex]

		// Create the API
		_, err := database.Exec(`
			INSERT INTO apis (id, name, description, created_at, updated_at, is_active, api_key, 
							  host_user_id, policy_id, is_deprecated)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			apiID, "Test API "+strconv.Itoa(i+1), "API for testing "+strconv.Itoa(i+1),
			time.Now(), time.Now(), apiStatuses[i].isActive, "test_key_"+strconv.Itoa(i+1),
			"test_host", policyID, apiStatuses[i].isDeprecated)

		if err != nil {
			t.Fatalf("Failed to insert test API: %v", err)
		}

		// Add some external users to each API
		for j := 0; j < 2; j++ {
			userAccessID := uuid.New().String()
			_, err := database.Exec(`
				INSERT INTO api_user_access (id, api_id, external_user_id, access_level, granted_at, granted_by, is_active)
				VALUES (?, ?, ?, ?, ?, ?, ?)`,
				userAccessID, apiID, "user_"+strconv.Itoa(j+1),
				"read", time.Now(), "test_host", true)

			if err != nil {
				t.Fatalf("Failed to insert test API user access: %v", err)
			}
		}

		// Add documents to each API
		for j := 0; j < 3; j++ {
			docAssocID := uuid.New().String()
			_, err := database.Exec(`
				INSERT INTO document_associations (id, document_filename, entity_id, entity_type, created_at)
				VALUES (?, ?, ?, ?, ?)`,
				docAssocID, "document_"+strconv.Itoa(i+1)+"_"+strconv.Itoa(j+1)+".txt",
				apiID, "api", time.Now())

			if err != nil {
				t.Fatalf("Failed to insert test document association: %v", err)
			}
		}
	}

	return policyIDs, apiIDs
}

// TestGetAPIs tests the GET /api/apis endpoint
func TestGetAPIs(t *testing.T) {
	// Setup test database with data
	database := setupTestDB(t)
	defer database.Close()

	// Insert test data (policyIDs and apiIDs not used directly in this test)
	_, _ = insertTestData(t, database)

	// Create a context with the database
	ctx := context.WithValue(context.Background(), "db", database)

	// Define test cases
	testCases := []struct {
		name           string
		queryParams    map[string]string
		expectedStatus int
		expectedCount  int
		checkResponse  func(*testing.T, *httpPkg.APIListResponse)
	}{
		{
			name:           "Get all APIs",
			queryParams:    map[string]string{},
			expectedStatus: 200,
			expectedCount:  5,
			checkResponse: func(t *testing.T, resp *httpPkg.APIListResponse) {
				if resp.Total != 5 {
					t.Errorf("Expected total count of 5, got %d", resp.Total)
				}
			},
		},
		{
			name:           "Filter active APIs",
			queryParams:    map[string]string{"status": "active"},
			expectedStatus: 200,
			expectedCount:  2,
			checkResponse: func(t *testing.T, resp *httpPkg.APIListResponse) {
				for _, api := range resp.APIs {
					if !api.IsActive || api.IsDeprecated {
						t.Errorf("Expected only active non-deprecated APIs, got isActive=%v, isDeprecated=%v",
							api.IsActive, api.IsDeprecated)
					}
				}
			},
		},
		{
			name:           "Filter inactive APIs",
			queryParams:    map[string]string{"status": "inactive"},
			expectedStatus: 200,
			expectedCount:  1,
			checkResponse: func(t *testing.T, resp *httpPkg.APIListResponse) {
				for _, api := range resp.APIs {
					if api.IsActive || api.IsDeprecated {
						t.Errorf("Expected only inactive non-deprecated APIs, got isActive=%v, isDeprecated=%v",
							api.IsActive, api.IsDeprecated)
					}
				}
			},
		},
		{
			name:           "Filter deprecated APIs",
			queryParams:    map[string]string{"status": "deprecated"},
			expectedStatus: 200,
			expectedCount:  2,
			checkResponse: func(t *testing.T, resp *httpPkg.APIListResponse) {
				for _, api := range resp.APIs {
					if !api.IsDeprecated {
						t.Errorf("Expected only deprecated APIs, got isDeprecated=%v", api.IsDeprecated)
					}
				}
			},
		},
		{
			name:           "Filter by external user ID",
			queryParams:    map[string]string{"external_user_id": "user_1"},
			expectedStatus: 200,
			expectedCount:  5,
			checkResponse: func(t *testing.T, resp *httpPkg.APIListResponse) {
				// All APIs have user_1 as an external user
				if len(resp.APIs) != 5 {
					t.Errorf("Expected 5 APIs accessible by user_1, got %d", len(resp.APIs))
				}
			},
		},
		{
			name:           "Pagination - first page",
			queryParams:    map[string]string{"limit": "2", "offset": "0"},
			expectedStatus: 200,
			expectedCount:  2,
			checkResponse: func(t *testing.T, resp *httpPkg.APIListResponse) {
				if resp.Limit != 2 || resp.Offset != 0 {
					t.Errorf("Expected limit=2, offset=0, got limit=%d, offset=%d", resp.Limit, resp.Offset)
				}
				if len(resp.APIs) != 2 {
					t.Errorf("Expected 2 APIs in response, got %d", len(resp.APIs))
				}
			},
		},
		{
			name:           "Pagination - second page",
			queryParams:    map[string]string{"limit": "2", "offset": "2"},
			expectedStatus: 200,
			expectedCount:  2,
			checkResponse: func(t *testing.T, resp *httpPkg.APIListResponse) {
				if resp.Limit != 2 || resp.Offset != 2 {
					t.Errorf("Expected limit=2, offset=2, got limit=%d, offset=%d", resp.Limit, resp.Offset)
				}
				if len(resp.APIs) != 2 {
					t.Errorf("Expected 2 APIs in response, got %d", len(resp.APIs))
				}
			},
		},
		{
			name:           "Sorting by name ascending",
			queryParams:    map[string]string{"sort": "name", "order": "asc"},
			expectedStatus: 200,
			expectedCount:  5,
			checkResponse: func(t *testing.T, resp *httpPkg.APIListResponse) {
				// Check if APIs are sorted by name in ascending order
				for i := 1; i < len(resp.APIs); i++ {
					if resp.APIs[i-1].Name > resp.APIs[i].Name {
						t.Errorf("APIs not sorted by name in ascending order: %s > %s",
							resp.APIs[i-1].Name, resp.APIs[i].Name)
					}
				}
			},
		},
		{
			name:           "Sorting by name descending",
			queryParams:    map[string]string{"sort": "name", "order": "desc"},
			expectedStatus: 200,
			expectedCount:  5,
			checkResponse: func(t *testing.T, resp *httpPkg.APIListResponse) {
				// Check if APIs are sorted by name in descending order
				for i := 1; i < len(resp.APIs); i++ {
					if resp.APIs[i-1].Name < resp.APIs[i].Name {
						t.Errorf("APIs not sorted by name in descending order: %s < %s",
							resp.APIs[i-1].Name, resp.APIs[i].Name)
					}
				}
			},
		},
	}

	// Run the test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test request
			req := httptest.NewRequest("GET", "/api/apis", nil)

			// Add query parameters
			q := req.URL.Query()
			for key, value := range tc.queryParams {
				q.Add(key, value)
			}
			req.URL.RawQuery = q.Encode()

			// Create test response recorder
			w := httptest.NewRecorder()

			// Call the handler directly
			httpPkg.HandleGetAPIs(ctx, w, req)

			// Check response status code
			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tc.expectedStatus, w.Code)
			}

			// For successful responses, check the content
			if w.Code == 200 {
				var resp httpPkg.APIListResponse
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("Failed to parse response JSON: %v", err)
				}

				// Check total count first
				if resp.Total != tc.expectedCount {
					t.Errorf("Expected %d APIs, got %d", tc.expectedCount, resp.Total)
				}

				// Run the custom response checks
				tc.checkResponse(t, &resp)
			}
		})
	}
}

// TestGetAPIDetails tests the GET /api/apis/:id endpoint
func TestGetAPIDetails(t *testing.T) {
	// Setup test database with data
	database := setupTestDB(t)
	defer database.Close()

	// Insert test data
	_, apiIDs := insertTestData(t, database)

	// Create a context with the database
	ctx := context.WithValue(context.Background(), "db", database)

	// Define test cases
	testCases := []struct {
		name           string
		apiID          string
		expectedStatus int
		checkResponse  func(*testing.T, *httpPkg.APIDetailResponse)
	}{
		{
			name:           "Get existing API details",
			apiID:          apiIDs[0],
			expectedStatus: 200,
			checkResponse: func(t *testing.T, resp *httpPkg.APIDetailResponse) {
				if resp.ID != apiIDs[0] {
					t.Errorf("Expected API ID %s, got %s", apiIDs[0], resp.ID)
				}

				// Check that external users are present
				if len(resp.ExternalUsers) != 2 {
					t.Errorf("Expected 2 external users, got %d", len(resp.ExternalUsers))
				}

				// Check that documents are present
				if len(resp.Documents) != 3 {
					t.Errorf("Expected 3 documents, got %d", len(resp.Documents))
				}

				// Check that the policy is present
				if resp.Policy == nil {
					t.Errorf("Expected policy to be present")
				}
			},
		},
		{
			name:           "Get non-existent API",
			apiID:          "non-existent-id",
			expectedStatus: 404,
			checkResponse:  nil,
		},
		{
			name:           "Invalid API ID",
			apiID:          "",
			expectedStatus: 400,
			checkResponse:  nil,
		},
	}

	// Run the test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test request
			req := httptest.NewRequest("GET", "/api/apis/"+tc.apiID, nil)

			// Add the API ID as a path parameter
			req = req.WithContext(context.WithValue(req.Context(), "id", tc.apiID))

			// Create test response recorder
			w := httptest.NewRecorder()

			// Call the handler directly
			httpPkg.HandleGetAPI(ctx, w, req)

			// Check response status code
			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tc.expectedStatus, w.Code)
			}

			// For successful responses, check the content
			if w.Code == 200 && tc.checkResponse != nil {
				var resp httpPkg.APIDetailResponse
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("Failed to parse response JSON: %v", err)
				}

				// Run the custom response checks
				tc.checkResponse(t, &resp)
			}
		})
	}
}

// TestCreateAPI tests the POST /api/apis endpoint
func TestCreateAPI(t *testing.T) {
	// Setup test database with data
	database := setupTestDB(t)
	defer database.Close()

	// Insert test data
	policyIDs, _ := insertTestData(t, database)

	// Create a context with the database and a test user ID
	ctx := context.WithValue(context.Background(), "db", database)
	ctx = context.WithValue(ctx, "user_id", "test_creator")

	// Define test cases
	testCases := []struct {
		name           string
		requestBody    httpPkg.CreateAPIRequest
		expectedStatus int
	}{
		{
			name: "Create API with valid data",
			requestBody: httpPkg.CreateAPIRequest{
				Name:        "New Test API",
				Description: "API created during testing",
				PolicyID:    policyIDs[0],
				DocumentIDs: []string{"doc1.txt", "doc2.txt"},
				ExternalUsers: []struct {
					UserID      string `json:"user_id"`
					AccessLevel string `json:"access_level"`
				}{
					{UserID: "ext_user_1", AccessLevel: "read"},
					{UserID: "ext_user_2", AccessLevel: "write"},
				},
				IsActive: true,
			},
			expectedStatus: 201,
		},
		{
			name: "Create API without name",
			requestBody: httpPkg.CreateAPIRequest{
				Description: "API without name",
				PolicyID:    policyIDs[1],
				IsActive:    true,
			},
			expectedStatus: 400,
		},
		{
			name: "Create API with invalid policy ID",
			requestBody: httpPkg.CreateAPIRequest{
				Name:        "API with invalid policy",
				Description: "This should fail with a server error",
				PolicyID:    "non-existent-policy",
				IsActive:    true,
			},
			expectedStatus: 500, // Should fail due to foreign key constraint
		},
	}

	// Run the test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Convert request body to JSON
			reqBody, err := json.Marshal(tc.requestBody)
			if err != nil {
				t.Fatalf("Failed to marshal request body: %v", err)
			}

			// Create test request
			req := httptest.NewRequest("POST", "/api/apis", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")

			// Create test response recorder
			w := httptest.NewRecorder()

			// Call the handler directly
			httpPkg.HandleCreateAPI(ctx, w, req)

			// Check response status code
			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status code %d, got %d: %s", tc.expectedStatus, w.Code, w.Body.String())
			}

			// For successful responses, check that the API was created in the database
			if w.Code == 201 {
				var createdAPI db.API
				if err := json.Unmarshal(w.Body.Bytes(), &createdAPI); err != nil {
					t.Fatalf("Failed to parse response JSON: %v", err)
				}

				// Verify the API exists in the database
				var count int
				err := database.QueryRow("SELECT COUNT(*) FROM apis WHERE id = ?", createdAPI.ID).Scan(&count)
				if err != nil {
					t.Fatalf("Failed to query database: %v", err)
				}

				if count != 1 {
					t.Errorf("Expected API to be in database, got count = %d", count)
				}

				// Check that external users were created
				var userCount int
				err = database.QueryRow("SELECT COUNT(*) FROM api_user_access WHERE api_id = ?", createdAPI.ID).Scan(&userCount)
				if err != nil {
					t.Fatalf("Failed to query database for user access: %v", err)
				}

				if userCount != len(tc.requestBody.ExternalUsers) {
					t.Errorf("Expected %d user access entries, got %d", len(tc.requestBody.ExternalUsers), userCount)
				}

				// Check that documents were associated
				var docCount int
				err = database.QueryRow("SELECT COUNT(*) FROM document_associations WHERE entity_id = ?", createdAPI.ID).Scan(&docCount)
				if err != nil {
					t.Fatalf("Failed to query database for document associations: %v", err)
				}

				if docCount != len(tc.requestBody.DocumentIDs) {
					t.Errorf("Expected %d document associations, got %d", len(tc.requestBody.DocumentIDs), docCount)
				}
			}
		})
	}
}

// TestUpdateAPI tests the PATCH /api/apis/:id endpoint
func TestUpdateAPI(t *testing.T) {
	// Setup test database with data
	database := setupTestDB(t)
	defer database.Close()

	// Insert test data
	policyIDs, apiIDs := insertTestData(t, database)

	// Create a context with the database and a test user ID
	ctx := context.WithValue(context.Background(), "db", database)
	ctx = context.WithValue(ctx, "user_id", "test_updater")

	// Helper function to create a string pointer
	strPtr := func(s string) *string {
		return &s
	}

	// Helper function to create a bool pointer
	boolPtr := func(b bool) *bool {
		return &b
	}

	// Define test cases
	testCases := []struct {
		name           string
		apiID          string
		requestBody    httpPkg.UpdateAPIRequest
		expectedStatus int
		checkUpdate    func(*testing.T, *sql.DB, string)
	}{
		{
			name:  "Update API name and description",
			apiID: apiIDs[0],
			requestBody: httpPkg.UpdateAPIRequest{
				Name:        strPtr("Updated API Name"),
				Description: strPtr("Updated API description"),
			},
			expectedStatus: 200,
			checkUpdate: func(t *testing.T, db *sql.DB, apiID string) {
				var name, description string
				err := db.QueryRow("SELECT name, description FROM apis WHERE id = ?", apiID).Scan(&name, &description)
				if err != nil {
					t.Fatalf("Failed to query updated API: %v", err)
				}

				if name != "Updated API Name" {
					t.Errorf("Name was not updated correctly, got %s", name)
				}
				if description != "Updated API description" {
					t.Errorf("Description was not updated correctly, got %s", description)
				}
			},
		},
		{
			name:  "Update API active status",
			apiID: apiIDs[1],
			requestBody: httpPkg.UpdateAPIRequest{
				IsActive: boolPtr(false),
			},
			expectedStatus: 200,
			checkUpdate: func(t *testing.T, db *sql.DB, apiID string) {
				var isActive bool
				err := db.QueryRow("SELECT is_active FROM apis WHERE id = ?", apiID).Scan(&isActive)
				if err != nil {
					t.Fatalf("Failed to query updated API: %v", err)
				}

				if isActive != false {
					t.Errorf("IsActive was not updated correctly, got %v", isActive)
				}
			},
		},
		{
			name:  "Update API policy",
			apiID: apiIDs[2],
			requestBody: httpPkg.UpdateAPIRequest{
				PolicyID: strPtr(policyIDs[1]),
			},
			expectedStatus: 200,
			checkUpdate: func(t *testing.T, db *sql.DB, apiID string) {
				var policyID string
				err := db.QueryRow("SELECT policy_id FROM apis WHERE id = ?", apiID).Scan(&policyID)
				if err != nil {
					t.Fatalf("Failed to query updated API: %v", err)
				}

				if policyID != policyIDs[1] {
					t.Errorf("PolicyID was not updated correctly, got %s", policyID)
				}

				// Check that a policy change record was created
				var count int
				err = db.QueryRow("SELECT COUNT(*) FROM policy_changes WHERE api_id = ? AND new_policy_id = ?",
					apiID, policyIDs[1]).Scan(&count)
				if err != nil {
					t.Fatalf("Failed to query policy changes: %v", err)
				}

				if count != 1 {
					t.Errorf("Expected 1 policy change record, got %d", count)
				}
			},
		},
		{
			name:  "Update API with non-existent ID",
			apiID: "non-existent-id",
			requestBody: httpPkg.UpdateAPIRequest{
				Name: strPtr("This should fail"),
			},
			expectedStatus: 404,
			checkUpdate:    nil,
		},
		{
			name:  "Update API with invalid policy ID",
			apiID: apiIDs[3],
			requestBody: httpPkg.UpdateAPIRequest{
				PolicyID: strPtr("non-existent-policy"),
			},
			expectedStatus: 500, // This should trigger a foreign key constraint error
			checkUpdate:    nil,
		},
	}

	// Run the test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Convert request body to JSON
			reqBody, err := json.Marshal(tc.requestBody)
			if err != nil {
				t.Fatalf("Failed to marshal request body: %v", err)
			}

			// Create test request
			req := httptest.NewRequest("PATCH", "/api/apis/"+tc.apiID, bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")

			// Add the API ID as a path parameter
			req = req.WithContext(context.WithValue(req.Context(), "id", tc.apiID))

			// Create test response recorder
			w := httptest.NewRecorder()

			// Call the handler directly
			httpPkg.HandleUpdateAPI(ctx, w, req)

			// Check response status code
			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status code %d, got %d: %s", tc.expectedStatus, w.Code, w.Body.String())
			}

			// For successful responses, check that the API was updated correctly
			if w.Code == 200 && tc.checkUpdate != nil {
				tc.checkUpdate(t, database, tc.apiID)
			}
		})
	}
}

// TestDeprecateAPI tests the POST /api/apis/:id/deprecate endpoint
func TestDeprecateAPI(t *testing.T) {
	// Setup test database with data
	database := setupTestDB(t)
	defer database.Close()

	// Insert test data
	_, apiIDs := insertTestData(t, database)

	// Create a context with the database
	ctx := context.WithValue(context.Background(), "db", database)

	// Define test cases
	testCases := []struct {
		name           string
		apiID          string
		requestBody    httpPkg.DeprecateAPIRequest
		expectedStatus int
	}{
		{
			name:  "Deprecate active API",
			apiID: apiIDs[0], // This is an active API
			requestBody: httpPkg.DeprecateAPIRequest{
				DeprecationDate:    time.Now().AddDate(0, 3, 0), // 3 months in the future
				DeprecationMessage: "This API will be deprecated in 3 months",
			},
			expectedStatus: 200,
		},
		{
			name:  "Deprecate already deprecated API",
			apiID: apiIDs[3], // This is already a deprecated API
			requestBody: httpPkg.DeprecateAPIRequest{
				DeprecationDate:    time.Now().AddDate(0, 1, 0), // 1 month in the future
				DeprecationMessage: "Updated deprecation message",
			},
			expectedStatus: 200,
		},
		{
			name:  "Deprecate non-existent API",
			apiID: "non-existent-id",
			requestBody: httpPkg.DeprecateAPIRequest{
				DeprecationDate:    time.Now().AddDate(0, 6, 0),
				DeprecationMessage: "This should fail",
			},
			expectedStatus: 404,
		},
	}

	// Run the test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Convert request body to JSON
			reqBody, err := json.Marshal(tc.requestBody)
			if err != nil {
				t.Fatalf("Failed to marshal request body: %v", err)
			}

			// Create test request
			req := httptest.NewRequest("POST", "/api/apis/"+tc.apiID+"/deprecate", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")

			// Add the API ID as a path parameter
			req = req.WithContext(context.WithValue(req.Context(), "id", tc.apiID))

			// Create test response recorder
			w := httptest.NewRecorder()

			// Call the handler directly
			httpPkg.HandleDeprecateAPI(ctx, w, req)

			// Check response status code
			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status code %d, got %d: %s", tc.expectedStatus, w.Code, w.Body.String())
			}

			// For successful responses, check that the API was deprecated correctly
			if w.Code == 200 {
				var isDeprecated bool
				var deprecationMessage string
				err := database.QueryRow("SELECT is_deprecated, deprecation_message FROM apis WHERE id = ?", tc.apiID).
					Scan(&isDeprecated, &deprecationMessage)
				if err != nil {
					t.Fatalf("Failed to query deprecated API: %v", err)
				}

				if !isDeprecated {
					t.Errorf("API was not marked as deprecated")
				}

				if deprecationMessage != tc.requestBody.DeprecationMessage {
					t.Errorf("Deprecation message was not set correctly, expected '%s', got '%s'",
						tc.requestBody.DeprecationMessage, deprecationMessage)
				}
			}
		})
	}
}

// TestDeleteAPI tests the DELETE /api/apis/:id endpoint
func TestDeleteAPI(t *testing.T) {
	// Setup test database with data
	database := setupTestDB(t)
	defer database.Close()

	// Insert test data
	_, apiIDs := insertTestData(t, database)

	// Create a context with the database
	ctx := context.WithValue(context.Background(), "db", database)

	// Define test cases
	testCases := []struct {
		name           string
		apiID          string
		expectedStatus int
	}{
		{
			name:           "Delete existing API",
			apiID:          apiIDs[0],
			expectedStatus: 204,
		},
		{
			name:           "Delete non-existent API",
			apiID:          "non-existent-id",
			expectedStatus: 404,
		},
	}

	// Run the test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test request
			req := httptest.NewRequest("DELETE", "/api/apis/"+tc.apiID, nil)

			// Add the API ID as a path parameter
			req = req.WithContext(context.WithValue(req.Context(), "id", tc.apiID))

			// Create test response recorder
			w := httptest.NewRecorder()

			// Call the handler directly
			httpPkg.HandleDeleteAPI(ctx, w, req)

			// Check response status code
			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status code %d, got %d: %s", tc.expectedStatus, w.Code, w.Body.String())
			}

			// For successful deletions, check that the API was actually deleted
			if w.Code == 204 {
				var count int
				err := database.QueryRow("SELECT COUNT(*) FROM apis WHERE id = ?", tc.apiID).Scan(&count)
				if err != nil {
					t.Fatalf("Failed to query database: %v", err)
				}

				if count != 0 {
					t.Errorf("API was not deleted from the database")
				}

				// Check that associated records were also deleted (via cascade)
				var userCount int
				err = database.QueryRow("SELECT COUNT(*) FROM api_user_access WHERE api_id = ?", tc.apiID).Scan(&userCount)
				if err != nil {
					t.Fatalf("Failed to query api_user_access: %v", err)
				}

				if userCount != 0 {
					t.Errorf("API user access records were not deleted")
				}

				var docCount int
				err = database.QueryRow("SELECT COUNT(*) FROM document_associations WHERE entity_id = ? AND entity_type = 'api'",
					tc.apiID).Scan(&docCount)
				if err != nil {
					t.Fatalf("Failed to query document_associations: %v", err)
				}

				if docCount != 0 {
					t.Errorf("Document association records were not deleted")
				}
			}
		})
	}
}
