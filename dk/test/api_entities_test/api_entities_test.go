package api_entities

import (
	"bytes"
	"context"
	"database/sql"
	"dk/db"
	httpPkg "dk/http"
	"dk/utils"
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

	// Create a context with the database using the utility function
	ctx := utils.WithDatabase(context.Background(), database)

	// Define test cases
	testCases := []struct {
		name           string
		queryParams    map[string]string
		expectedStatus int
		checkResponse  func(*testing.T, *httpPkg.APIListResponse)
	}{
		{
			name:           "Get all APIs",
			queryParams:    map[string]string{},
			expectedStatus: 200,
			checkResponse: func(t *testing.T, resp *httpPkg.APIListResponse) {
				if resp.Total != 5 {
					t.Errorf("Expected 5 APIs, got %d", resp.Total)
				}
			},
		},
		{
			name:           "Filter active APIs",
			queryParams:    map[string]string{"status": "active"},
			expectedStatus: 200,
			checkResponse: func(t *testing.T, resp *httpPkg.APIListResponse) {
				// Should get 2 active APIs (not deprecated)
				if len(resp.APIs) != 2 {
					t.Errorf("Expected 2 active APIs, got %d", len(resp.APIs))
				}

				// Verify all returned APIs are active and not deprecated
				for _, api := range resp.APIs {
					if !api.IsActive || api.IsDeprecated {
						t.Errorf("API %s should be active and not deprecated", api.ID)
					}
				}
			},
		},
		{
			name:           "Filter inactive APIs",
			queryParams:    map[string]string{"status": "inactive"},
			expectedStatus: 200,
			checkResponse: func(t *testing.T, resp *httpPkg.APIListResponse) {
				// Should get 1 inactive API (not deprecated)
				if len(resp.APIs) != 1 {
					t.Errorf("Expected 1 inactive API, got %d", len(resp.APIs))
				}

				// Verify all returned APIs are inactive and not deprecated
				for _, api := range resp.APIs {
					if api.IsActive || api.IsDeprecated {
						t.Errorf("API %s should be inactive and not deprecated", api.ID)
					}
				}
			},
		},
		{
			name:           "Filter deprecated APIs",
			queryParams:    map[string]string{"status": "deprecated"},
			expectedStatus: 200,
			checkResponse: func(t *testing.T, resp *httpPkg.APIListResponse) {
				// Should get 2 deprecated APIs
				if len(resp.APIs) != 2 {
					t.Errorf("Expected 2 deprecated APIs, got %d", len(resp.APIs))
				}

				// Verify all returned APIs are deprecated
				for _, api := range resp.APIs {
					if !api.IsDeprecated {
						t.Errorf("API %s should be deprecated", api.ID)
					}
				}
			},
		},
		{
			name:           "Filter by external user ID",
			queryParams:    map[string]string{"external_user_id": "user_1"},
			expectedStatus: 200,
			checkResponse: func(t *testing.T, resp *httpPkg.APIListResponse) {
				// All 5 APIs have user_1 access
				if len(resp.APIs) != 5 {
					t.Errorf("Expected 5 APIs accessible by user_1, got %d", len(resp.APIs))
				}
			},
		},
		{
			name:           "Pagination - first page",
			queryParams:    map[string]string{"limit": "2", "offset": "0"},
			expectedStatus: 200,
			checkResponse: func(t *testing.T, resp *httpPkg.APIListResponse) {
				if resp.Limit != 2 || resp.Offset != 0 || len(resp.APIs) != 2 {
					t.Errorf("Expected limit=2, offset=0, APIs count=2, got limit=%d, offset=%d, APIs count=%d",
						resp.Limit, resp.Offset, len(resp.APIs))
				}
			},
		},
		{
			name:           "Pagination - second page",
			queryParams:    map[string]string{"limit": "2", "offset": "2"},
			expectedStatus: 200,
			checkResponse: func(t *testing.T, resp *httpPkg.APIListResponse) {
				if resp.Limit != 2 || resp.Offset != 2 || len(resp.APIs) != 2 {
					t.Errorf("Expected limit=2, offset=2, APIs count=2, got limit=%d, offset=%d, APIs count=%d",
						resp.Limit, resp.Offset, len(resp.APIs))
				}
			},
		},
		{
			name:           "Sorting by name ascending",
			queryParams:    map[string]string{"sort": "name", "order": "asc"},
			expectedStatus: 200,
			checkResponse: func(t *testing.T, resp *httpPkg.APIListResponse) {
				if len(resp.APIs) < 2 {
					t.Errorf("Expected at least two APIs to check sorting")
					return
				}

				// Check that APIs are sorted by name in ascending order
				for i := 1; i < len(resp.APIs); i++ {
					if resp.APIs[i-1].Name > resp.APIs[i].Name {
						t.Errorf("APIs not sorted correctly by name asc: %s should come before %s",
							resp.APIs[i].Name, resp.APIs[i-1].Name)
					}
				}
			},
		},
		{
			name:           "Sorting by name descending",
			queryParams:    map[string]string{"sort": "name", "order": "desc"},
			expectedStatus: 200,
			checkResponse: func(t *testing.T, resp *httpPkg.APIListResponse) {
				if len(resp.APIs) < 2 {
					t.Errorf("Expected at least two APIs to check sorting")
					return
				}

				// Check that APIs are sorted by name in descending order
				for i := 1; i < len(resp.APIs); i++ {
					if resp.APIs[i-1].Name < resp.APIs[i].Name {
						t.Errorf("APIs not sorted correctly by name desc: %s should come before %s",
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
	ctx := utils.WithDatabase(context.Background(), database)

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
	ctx := utils.WithDatabase(context.Background(), database)
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

			// For successful responses, check the response
			if w.Code == 201 {
				var createdAPI db.API
				if err := json.Unmarshal(w.Body.Bytes(), &createdAPI); err != nil {
					t.Fatalf("Failed to parse response JSON: %v", err)
				}

				// Check that the response contains expected values
				if createdAPI.Name != tc.requestBody.Name {
					t.Errorf("Expected API name %s, got %s", tc.requestBody.Name, createdAPI.Name)
				}

				if createdAPI.Description != tc.requestBody.Description {
					t.Errorf("Expected API description %s, got %s", tc.requestBody.Description, createdAPI.Description)
				}

				if createdAPI.IsActive != tc.requestBody.IsActive {
					t.Errorf("Expected API isActive %v, got %v", tc.requestBody.IsActive, createdAPI.IsActive)
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
	ctx := utils.WithDatabase(context.Background(), database)
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
	}{
		{
			name:  "Update API name and description",
			apiID: apiIDs[0],
			requestBody: httpPkg.UpdateAPIRequest{
				Name:        strPtr("Updated API Name"),
				Description: strPtr("Updated API description"),
			},
			expectedStatus: 200,
		},
		{
			name:  "Update API active status",
			apiID: apiIDs[1],
			requestBody: httpPkg.UpdateAPIRequest{
				IsActive: boolPtr(false),
			},
			expectedStatus: 200,
		},
		{
			name:  "Update API policy",
			apiID: apiIDs[2],
			requestBody: httpPkg.UpdateAPIRequest{
				PolicyID: strPtr(policyIDs[1]),
			},
			expectedStatus: 200,
		},
		{
			name:  "Update API with non-existent ID",
			apiID: "non-existent-id",
			requestBody: httpPkg.UpdateAPIRequest{
				Name: strPtr("This should fail"),
			},
			expectedStatus: 404,
		},
		{
			name:  "Update API with invalid policy ID",
			apiID: apiIDs[3],
			requestBody: httpPkg.UpdateAPIRequest{
				PolicyID: strPtr("non-existent-policy"),
			},
			expectedStatus: 500, // This should trigger a foreign key constraint error
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
	ctx := utils.WithDatabase(context.Background(), database)

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

			// For successful responses, check the response
			if w.Code == 200 {
				var api db.API
				if err := json.Unmarshal(w.Body.Bytes(), &api); err != nil {
					t.Fatalf("Failed to parse response JSON: %v", err)
				}

				if !api.IsDeprecated {
					t.Errorf("API was not marked as deprecated in response")
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
	ctx := utils.WithDatabase(context.Background(), database)

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

			// Verify that the API was deleted from the database
			if tc.expectedStatus == 204 {
				api, err := db.GetAPI(database, tc.apiID)
				if err != db.ErrNotFound {
					t.Errorf("API should be deleted but got: %v, error: %v", api, err)
				}
			}
		})
	}
}
