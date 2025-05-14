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

// setupTrackers adds test trackers to the database
func setupTrackers(t *testing.T, database *sql.DB) []string {
	trackerIDs := make([]string, 3)
	for i := 0; i < 3; i++ {
		trackerID := uuid.New().String()
		trackerIDs[i] = trackerID

		_, err := database.Exec(`
			INSERT INTO trackers (id, name, description, created_at, is_active)
			VALUES (?, ?, ?, ?, ?)`,
			trackerID, "Test Tracker "+strconv.Itoa(i+1), "Tracker for testing "+strconv.Itoa(i+1),
			time.Now(), true)

		if err != nil {
			t.Fatalf("Failed to insert test tracker: %v", err)
		}
	}

	return trackerIDs
}

// setupDocuments adds test documents to the database
func setupDocuments(t *testing.T, database *sql.DB) []string {
	// In a real implementation, we would store actual documents
	// For testing, we'll just create entries in document associations
	documentIDs := []string{
		"test_document_1.txt",
		"test_document_2.txt",
		"test_document_3.txt",
	}

	return documentIDs
}

// setupAPIRequests adds test API requests to the database
func setupAPIRequests(t *testing.T, database *sql.DB, policyIDs, documentIDs, trackerIDs []string) []string {
	requestIDs := make([]string, 5)
	requestStatuses := []string{
		"pending",
		"approved",
		"denied",
		"pending", // For resubmission test
		"denied",  // For resubmission test
	}

	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	twoDaysAgo := now.AddDate(0, 0, -2)

	for i := 0; i < 5; i++ {
		requestID := uuid.New().String()
		requestIDs[i] = requestID

		// For the fifth request, set its previous request ID to the fourth
		var previousRequestID *string
		if i == 4 {
			previousRequestID = &requestIDs[3]
		}

		// For request 1 and 4, create a proposed policy ID
		var proposedPolicyID *string
		if i == 0 || i == 3 {
			proposedPolicyID = &policyIDs[0]
		}

		// Create submitted date with different timestamps for sorting
		submittedDate := now
		if i == 1 {
			submittedDate = yesterday
		} else if i == 2 {
			submittedDate = twoDaysAgo
		}

		// Create approved or denied dates for approved/denied requests
		var approvedDate, deniedDate *time.Time
		var denialReason string

		if requestStatuses[i] == "approved" {
			approvedDate = &now
		} else if requestStatuses[i] == "denied" {
			deniedDate = &now
			denialReason = "Test denial reason for request " + strconv.Itoa(i+1)
		}

		// Insert the API request
		_, err := database.Exec(`
			INSERT INTO api_requests (
				id, api_name, description, submitted_date, status, requester_id,
				denial_reason, denied_date, approved_date, submission_count,
				previous_request_id, proposed_policy_id
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			requestID, "Test API Request "+strconv.Itoa(i+1), "API Request for testing "+strconv.Itoa(i+1),
			submittedDate, requestStatuses[i], "requester_"+strconv.Itoa(i%2+1), // alternate between two requesters
			denialReason, deniedDate, approvedDate, i%3+1, // submission count 1-3
			previousRequestID, proposedPolicyID)

		if err != nil {
			t.Fatalf("Failed to insert test API request: %v", err)
		}

		// Associate documents with the request
		for j := 0; j < 2; j++ {
			docIndex := (i + j) % len(documentIDs)
			docAssocID := uuid.New().String()

			_, err := database.Exec(`
				INSERT INTO document_associations (id, document_filename, entity_id, entity_type, created_at)
				VALUES (?, ?, ?, ?, ?)`,
				docAssocID, documentIDs[docIndex], requestID, "request", time.Now())

			if err != nil {
				t.Fatalf("Failed to insert test document association: %v", err)
			}
		}

		// Associate trackers with the request
		for j := 0; j < 2; j++ {
			trackerIndex := (i + j) % len(trackerIDs)
			trackerAssocID := uuid.New().String()

			_, err := database.Exec(`
				INSERT INTO request_required_trackers (id, request_id, tracker_id)
				VALUES (?, ?, ?)`,
				trackerAssocID, requestID, trackerIDs[trackerIndex])

			if err != nil {
				t.Fatalf("Failed to insert test tracker association: %v", err)
			}
		}
	}

	return requestIDs
}

// TestGetAPIRequests tests the GET /api/requests endpoint
func TestGetAPIRequests(t *testing.T) {
	// Setup test database with data
	database := setupTestDB(t)
	defer database.Close()

	// Insert test data
	policyIDs, _ := insertTestData(t, database)
	trackerIDs := setupTrackers(t, database)
	documentIDs := setupDocuments(t, database)
	_ = setupAPIRequests(t, database, policyIDs, documentIDs, trackerIDs)

	// Create a context with the database
	ctx := utils.WithDatabase(context.Background(), database)

	// Define test cases
	testCases := []struct {
		name           string
		queryParams    map[string]string
		expectedStatus int
		checkResponse  func(*testing.T, *httpPkg.APIRequestListResponse)
	}{
		{
			name:           "Get all API requests",
			queryParams:    map[string]string{},
			expectedStatus: 200,
			checkResponse: func(t *testing.T, resp *httpPkg.APIRequestListResponse) {
				if resp.Total != 5 {
					t.Errorf("Expected 5 API requests, got %d", resp.Total)
				}
			},
		},
		{
			name:           "Filter pending API requests",
			queryParams:    map[string]string{"status": "pending"},
			expectedStatus: 200,
			checkResponse: func(t *testing.T, resp *httpPkg.APIRequestListResponse) {
				if len(resp.Requests) != 2 {
					t.Errorf("Expected 2 pending API requests, got %d", len(resp.Requests))
				}

				for _, req := range resp.Requests {
					if req.Status != "pending" {
						t.Errorf("Expected request %s to be pending, got %s", req.ID, req.Status)
					}
				}
			},
		},
		{
			name:           "Filter approved API requests",
			queryParams:    map[string]string{"status": "approved"},
			expectedStatus: 200,
			checkResponse: func(t *testing.T, resp *httpPkg.APIRequestListResponse) {
				if len(resp.Requests) != 1 {
					t.Errorf("Expected 1 approved API request, got %d", len(resp.Requests))
				}

				for _, req := range resp.Requests {
					if req.Status != "approved" {
						t.Errorf("Expected request %s to be approved, got %s", req.ID, req.Status)
					}
				}
			},
		},
		{
			name:           "Filter denied API requests",
			queryParams:    map[string]string{"status": "denied"},
			expectedStatus: 200,
			checkResponse: func(t *testing.T, resp *httpPkg.APIRequestListResponse) {
				if len(resp.Requests) != 2 {
					t.Errorf("Expected 2 denied API requests, got %d", len(resp.Requests))
				}

				for _, req := range resp.Requests {
					if req.Status != "denied" {
						t.Errorf("Expected request %s to be denied, got %s", req.ID, req.Status)
					}
				}
			},
		},
		{
			name:           "Filter by requester",
			queryParams:    map[string]string{"requester_id": "requester_1"},
			expectedStatus: 200,
			checkResponse: func(t *testing.T, resp *httpPkg.APIRequestListResponse) {
				for _, req := range resp.Requests {
					if req.Requester.ID != "requester_1" {
						t.Errorf("Expected requester ID to be requester_1, got %s", req.Requester.ID)
					}
				}
			},
		},
		{
			name:           "Pagination - limit 2, offset 0",
			queryParams:    map[string]string{"limit": "2", "offset": "0"},
			expectedStatus: 200,
			checkResponse: func(t *testing.T, resp *httpPkg.APIRequestListResponse) {
				if resp.Limit != 2 || resp.Offset != 0 || len(resp.Requests) != 2 {
					t.Errorf("Expected limit=2, offset=0, requests=2, got limit=%d, offset=%d, requests=%d",
						resp.Limit, resp.Offset, len(resp.Requests))
				}
			},
		},
		{
			name:           "Pagination - limit 2, offset 2",
			queryParams:    map[string]string{"limit": "2", "offset": "2"},
			expectedStatus: 200,
			checkResponse: func(t *testing.T, resp *httpPkg.APIRequestListResponse) {
				if resp.Limit != 2 || resp.Offset != 2 || len(resp.Requests) != 2 {
					t.Errorf("Expected limit=2, offset=2, requests=2, got limit=%d, offset=%d, requests=%d",
						resp.Limit, resp.Offset, len(resp.Requests))
				}
			},
		},
		{
			name:           "Sort by submitted_date ascending",
			queryParams:    map[string]string{"sort": "submitted_date", "order": "asc"},
			expectedStatus: 200,
			checkResponse: func(t *testing.T, resp *httpPkg.APIRequestListResponse) {
				if len(resp.Requests) < 2 {
					t.Errorf("Expected at least 2 requests, got %d", len(resp.Requests))
					return
				}

				lastDate := time.Time{}
				for _, req := range resp.Requests {
					if !lastDate.IsZero() && req.SubmittedDate.Before(lastDate) {
						t.Errorf("Requests not sorted correctly by submitted_date asc")
					}
					lastDate = req.SubmittedDate
				}
			},
		},
		{
			name:           "Sort by api_name descending",
			queryParams:    map[string]string{"sort": "api_name", "order": "desc"},
			expectedStatus: 200,
			checkResponse: func(t *testing.T, resp *httpPkg.APIRequestListResponse) {
				if len(resp.Requests) < 2 {
					t.Errorf("Expected at least 2 requests, got %d", len(resp.Requests))
					return
				}

				lastName := ""
				for i, req := range resp.Requests {
					if i > 0 && req.APIName > lastName {
						t.Errorf("Requests not sorted correctly by api_name desc")
					}
					lastName = req.APIName
				}
			},
		},
	}

	// Run the test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test request
			req := httptest.NewRequest("GET", "/api/requests", nil)

			// Add query parameters
			q := req.URL.Query()
			for key, value := range tc.queryParams {
				q.Add(key, value)
			}
			req.URL.RawQuery = q.Encode()

			// Create test response recorder
			w := httptest.NewRecorder()

			// Call the handler directly
			httpPkg.HandleGetAPIRequests(ctx, w, req)

			// Check response status code
			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tc.expectedStatus, w.Code)
			}

			// For successful responses, check the content
			if w.Code == 200 {
				var resp httpPkg.APIRequestListResponse
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("Failed to parse response JSON: %v", err)
				}

				// Run the custom response checks
				tc.checkResponse(t, &resp)
			}
		})
	}
}

// TestGetAPIRequestDetails tests the GET /api/requests/:id endpoint
func TestGetAPIRequestDetails(t *testing.T) {
	// Setup test database with data
	database := setupTestDB(t)
	defer database.Close()

	// Insert test data
	policyIDs, _ := insertTestData(t, database)
	trackerIDs := setupTrackers(t, database)
	documentIDs := setupDocuments(t, database)
	requestIDs := setupAPIRequests(t, database, policyIDs, documentIDs, trackerIDs)

	// Create contexts with different user IDs
	ctxWithDB := utils.WithDatabase(context.Background(), database)
	ctxWithHostUser := context.WithValue(ctxWithDB, "user_id", "local-user")
	ctxWithRequester := context.WithValue(ctxWithDB, "user_id", "requester_1")
	ctxWithOtherUser := context.WithValue(ctxWithDB, "user_id", "other-user")

	// Define test cases
	testCases := []struct {
		name           string
		requestID      string
		ctx            context.Context
		expectedStatus int
		checkResponse  func(*testing.T, *httpPkg.APIRequestDetailResponse)
	}{
		{
			name:           "Get existing API request as host user",
			requestID:      requestIDs[0],
			ctx:            ctxWithHostUser,
			expectedStatus: 200,
			checkResponse: func(t *testing.T, resp *httpPkg.APIRequestDetailResponse) {
				if resp.ID != requestIDs[0] {
					t.Errorf("Expected request ID %s, got %s", requestIDs[0], resp.ID)
				}

				// Check that documents are present
				if len(resp.Documents) != 2 {
					t.Errorf("Expected 2 documents, got %d", len(resp.Documents))
				}

				// Check that trackers are present
				if len(resp.RequiredTrackers) != 2 {
					t.Errorf("Expected 2 trackers, got %d", len(resp.RequiredTrackers))
				}

				// Check that proposed policy is present
				if resp.ProposedPolicy == nil {
					t.Errorf("Expected proposed policy to be present")
				}
			},
		},
		{
			name:           "Get existing API request as requester",
			requestID:      requestIDs[0], // First request has requester_1
			ctx:            ctxWithRequester,
			expectedStatus: 200,
			checkResponse: func(t *testing.T, resp *httpPkg.APIRequestDetailResponse) {
				if resp.ID != requestIDs[0] {
					t.Errorf("Expected request ID %s, got %s", requestIDs[0], resp.ID)
				}

				if resp.Requester.ID != "requester_1" {
					t.Errorf("Expected requester ID to be requester_1, got %s", resp.Requester.ID)
				}
			},
		},
		{
			name:           "Get other's API request as unauthorized user",
			requestID:      requestIDs[0],
			ctx:            ctxWithOtherUser,
			expectedStatus: 403,
			checkResponse:  nil,
		},
		{
			name:           "Get approved API request details",
			requestID:      requestIDs[1], // This is an approved request
			ctx:            ctxWithHostUser,
			expectedStatus: 200,
			checkResponse: func(t *testing.T, resp *httpPkg.APIRequestDetailResponse) {
				if resp.Status != "approved" {
					t.Errorf("Expected status to be approved, got %s", resp.Status)
				}

				if resp.ApprovedDate == nil {
					t.Errorf("Expected approved date to be present")
				}
			},
		},
		{
			name:           "Get denied API request details",
			requestID:      requestIDs[2], // This is a denied request
			ctx:            ctxWithHostUser,
			expectedStatus: 200,
			checkResponse: func(t *testing.T, resp *httpPkg.APIRequestDetailResponse) {
				if resp.Status != "denied" {
					t.Errorf("Expected status to be denied, got %s", resp.Status)
				}

				if resp.DeniedDate == nil {
					t.Errorf("Expected denied date to be present")
				}

				if resp.DenialReason == "" {
					t.Errorf("Expected denial reason to be present")
				}
			},
		},
		{
			name:           "Get API request with previous request",
			requestID:      requestIDs[4], // This has a previous request
			ctx:            ctxWithHostUser,
			expectedStatus: 200,
			checkResponse: func(t *testing.T, resp *httpPkg.APIRequestDetailResponse) {
				if resp.PreviousRequest == nil {
					t.Errorf("Expected previous request to be present")
				}

				if resp.PreviousRequest != nil && resp.PreviousRequest.ID != requestIDs[3] {
					t.Errorf("Expected previous request ID to be %s, got %s", requestIDs[3], resp.PreviousRequest.ID)
				}
			},
		},
		{
			name:           "Get non-existent API request",
			requestID:      "non-existent-id",
			ctx:            ctxWithHostUser,
			expectedStatus: 404,
			checkResponse:  nil,
		},
	}

	// Run the test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test request
			req := httptest.NewRequest("GET", "/api/requests/"+tc.requestID, nil)

			// Add the request ID as a path parameter using PathValue method
			req = req.WithContext(context.WithValue(req.Context(), httpPkg.PathParamContextKey, map[string]string{"id": tc.requestID}))

			// Create test response recorder
			w := httptest.NewRecorder()

			// Call the handler directly with the appropriate context
			httpPkg.HandleGetAPIRequest(tc.ctx, w, req)

			// Check response status code
			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status code %d, got %d: %s", tc.expectedStatus, w.Code, w.Body.String())
			}

			// For successful responses, check the content
			if w.Code == 200 && tc.checkResponse != nil {
				var resp httpPkg.APIRequestDetailResponse
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("Failed to parse response JSON: %v", err)
				}

				// Run the custom response checks
				tc.checkResponse(t, &resp)
			}
		})
	}
}

// TestCreateAPIRequest tests the POST /api/requests endpoint
func TestCreateAPIRequest(t *testing.T) {
	// Setup test database with data
	database := setupTestDB(t)
	defer database.Close()

	// Insert test data
	policyIDs, _ := insertTestData(t, database)
	trackerIDs := setupTrackers(t, database)
	documentIDs := setupDocuments(t, database)

	// Create a context with the database and external user ID
	ctx := utils.WithDatabase(context.Background(), database)
	ctx = context.WithValue(ctx, "user_id", "external-user")

	// Define test cases
	testCases := []struct {
		name           string
		requestBody    httpPkg.CreateAPIRequestRequest
		expectedStatus int
	}{
		{
			name: "Create API request with valid data",
			requestBody: httpPkg.CreateAPIRequestRequest{
				APIName:            "New Test API Request",
				Description:        "API Request created during testing",
				DocumentIDs:        []string{documentIDs[0], documentIDs[1]},
				RequiredTrackerIDs: []string{trackerIDs[0], trackerIDs[1]},
				ProposedPolicyID:   policyIDs[0],
			},
			expectedStatus: 201,
		},
		{
			name: "Create API request without API name",
			requestBody: httpPkg.CreateAPIRequestRequest{
				Description:        "API Request without name",
				DocumentIDs:        []string{documentIDs[0]},
				RequiredTrackerIDs: []string{trackerIDs[0]},
			},
			expectedStatus: 400,
		},
		{
			name: "Create API request with invalid tracker ID",
			requestBody: httpPkg.CreateAPIRequestRequest{
				APIName:            "API Request with invalid tracker",
				Description:        "This should fail due to tracker validation",
				DocumentIDs:        []string{documentIDs[0]},
				RequiredTrackerIDs: []string{"non-existent-tracker"},
			},
			expectedStatus: 400,
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
			req := httptest.NewRequest("POST", "/api/requests", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")

			// Create test response recorder
			w := httptest.NewRecorder()

			// Call the handler directly
			httpPkg.HandleCreateAPIRequest(ctx, w, req)

			// Check response status code
			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status code %d, got %d: %s", tc.expectedStatus, w.Code, w.Body.String())
			}

			// For successful responses, check the response
			if w.Code == 201 {
				var createdRequest db.APIRequest
				if err := json.Unmarshal(w.Body.Bytes(), &createdRequest); err != nil {
					t.Fatalf("Failed to parse response JSON: %v", err)
				}

				// Check that the response contains expected values
				if createdRequest.APIName != tc.requestBody.APIName {
					t.Errorf("Expected API name %s, got %s", tc.requestBody.APIName, createdRequest.APIName)
				}

				if createdRequest.Description != tc.requestBody.Description {
					t.Errorf("Expected description %s, got %s", tc.requestBody.Description, createdRequest.Description)
				}

				if createdRequest.Status != "pending" {
					t.Errorf("Expected status to be pending, got %s", createdRequest.Status)
				}

				if createdRequest.SubmissionCount != 1 {
					t.Errorf("Expected submission count to be 1, got %d", createdRequest.SubmissionCount)
				}

				// Check that documents were associated correctly
				docCount, err := db.CountRequestDocuments(database, createdRequest.ID)
				if err != nil {
					t.Fatalf("Failed to count request documents: %v", err)
				}
				if docCount != len(tc.requestBody.DocumentIDs) {
					t.Errorf("Expected %d associated documents, got %d", len(tc.requestBody.DocumentIDs), docCount)
				}

				// Check that trackers were associated correctly
				trackerCount, err := db.CountRequestTrackers(database, createdRequest.ID)
				if err != nil {
					t.Fatalf("Failed to count request trackers: %v", err)
				}
				if trackerCount != len(tc.requestBody.RequiredTrackerIDs) {
					t.Errorf("Expected %d associated trackers, got %d", len(tc.requestBody.RequiredTrackerIDs), trackerCount)
				}
			}
		})
	}
}

// TestUpdateAPIRequestStatus tests the PATCH /api/requests/:id/status endpoint
func TestUpdateAPIRequestStatus(t *testing.T) {
	// Setup test database with data
	database := setupTestDB(t)
	defer database.Close()

	// Insert test data
	policyIDs, _ := insertTestData(t, database)
	trackerIDs := setupTrackers(t, database)
	documentIDs := setupDocuments(t, database)
	requestIDs := setupAPIRequests(t, database, policyIDs, documentIDs, trackerIDs)

	// Create a context with the database and host user ID
	ctx := utils.WithDatabase(context.Background(), database)
	ctx = context.WithValue(ctx, "user_id", "local-user")

	// Define test cases
	testCases := []struct {
		name           string
		requestID      string
		requestBody    httpPkg.UpdateAPIRequestStatusRequest
		expectedStatus int
	}{
		{
			name:      "Approve pending API request with policy",
			requestID: requestIDs[0], // This is a pending request
			requestBody: httpPkg.UpdateAPIRequestStatusRequest{
				Status:   "approved",
				PolicyID: policyIDs[0],
			},
			expectedStatus: 200,
		},
		{
			name:      "Approve pending API request with policy and create API",
			requestID: requestIDs[3], // This is another pending request
			requestBody: httpPkg.UpdateAPIRequestStatusRequest{
				Status:    "approved",
				PolicyID:  policyIDs[1],
				CreateAPI: true,
			},
			expectedStatus: 200,
		},
		{
			name:      "Deny pending API request with reason",
			requestID: requestIDs[0], // This is now approved, but handler should check original status
			requestBody: httpPkg.UpdateAPIRequestStatusRequest{
				Status:       "denied",
				DenialReason: "Test denial reason",
			},
			expectedStatus: 400, // Should fail because request is no longer pending
		},
		{
			name:      "Approve request without policy ID",
			requestID: requestIDs[0],
			requestBody: httpPkg.UpdateAPIRequestStatusRequest{
				Status: "approved",
			},
			expectedStatus: 400,
		},
		{
			name:      "Deny request without denial reason",
			requestID: requestIDs[3],
			requestBody: httpPkg.UpdateAPIRequestStatusRequest{
				Status: "denied",
			},
			expectedStatus: 400,
		},
		{
			name:      "Update non-existent API request",
			requestID: "non-existent-id",
			requestBody: httpPkg.UpdateAPIRequestStatusRequest{
				Status:       "denied",
				DenialReason: "This should fail",
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
			req := httptest.NewRequest("PATCH", "/api/requests/"+tc.requestID+"/status", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")

			// Add the request ID as a path parameter
			req = req.WithContext(context.WithValue(req.Context(), httpPkg.PathParamContextKey, map[string]string{"id": tc.requestID}))

			// Create test response recorder
			w := httptest.NewRecorder()

			// Call the handler directly
			httpPkg.HandleUpdateAPIRequestStatus(ctx, w, req)

			// Check response status code
			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status code %d, got %d: %s", tc.expectedStatus, w.Code, w.Body.String())
			}

			// For successful responses, check the response
			if w.Code == 200 {
				var updatedRequest db.APIRequest
				if err := json.Unmarshal(w.Body.Bytes(), &updatedRequest); err != nil {
					t.Fatalf("Failed to parse response JSON: %v", err)
				}

				// Check that the status was updated correctly
				if updatedRequest.Status != tc.requestBody.Status {
					t.Errorf("Expected status to be %s, got %s", tc.requestBody.Status, updatedRequest.Status)
				}

				// If approved, check that approved date is set
				if tc.requestBody.Status == "approved" {
					if updatedRequest.ApprovedDate == nil {
						t.Errorf("Expected approved date to be set")
					}

					// If create_api was true, check that an API was created
					if tc.requestBody.CreateAPI {
						// Query the database to find an API with the same name
						var apiID string
						err := database.QueryRow("SELECT id FROM apis WHERE name = ?", updatedRequest.APIName).Scan(&apiID)
						if err != nil {
							t.Errorf("Expected to find an API with name %s, but got error: %v", updatedRequest.APIName, err)
						}

						// Check that the API has the correct policy
						var policyID string
						err = database.QueryRow("SELECT policy_id FROM apis WHERE id = ?", apiID).Scan(&policyID)
						if err != nil || policyID != tc.requestBody.PolicyID {
							t.Errorf("Expected API to have policy ID %s, but got %s (err: %v)", tc.requestBody.PolicyID, policyID, err)
						}

						// Check that documents were copied
						var docCount int
						err = database.QueryRow("SELECT COUNT(*) FROM document_associations WHERE entity_id = ? AND entity_type = 'api'", apiID).Scan(&docCount)
						if err != nil || docCount == 0 {
							t.Errorf("Expected API to have documents, but got count %d (err: %v)", docCount, err)
						}

						// Check that user access was granted
						var userAccessCount int
						err = database.QueryRow("SELECT COUNT(*) FROM api_user_access WHERE api_id = ?", apiID).Scan(&userAccessCount)
						if err != nil || userAccessCount == 0 {
							t.Errorf("Expected API to have user access, but got count %d (err: %v)", userAccessCount, err)
						}
					}
				}

				// If denied, check that denied date and reason are set
				if tc.requestBody.Status == "denied" {
					if updatedRequest.DeniedDate == nil {
						t.Errorf("Expected denied date to be set")
					}

					if updatedRequest.DenialReason != tc.requestBody.DenialReason {
						t.Errorf("Expected denial reason to be %s, got %s", tc.requestBody.DenialReason, updatedRequest.DenialReason)
					}
				}
			}
		})
	}
}

// TestResubmitAPIRequest tests the POST /api/requests/:id/resubmit endpoint
func TestResubmitAPIRequest(t *testing.T) {
	// Setup test database with data
	database := setupTestDB(t)
	defer database.Close()

	// Insert test data
	policyIDs, _ := insertTestData(t, database)
	trackerIDs := setupTrackers(t, database)
	documentIDs := setupDocuments(t, database)
	requestIDs := setupAPIRequests(t, database, policyIDs, documentIDs, trackerIDs)

	// Create contexts with different user IDs
	ctxWithDB := utils.WithDatabase(context.Background(), database)
	ctxWithRequester1 := context.WithValue(ctxWithDB, "user_id", "requester_1")
	// We need a requester_2 context for the "Resubmit as different user" test
	ctxWithRequester2 := context.WithValue(ctxWithDB, "user_id", "requester_2")

	// Define test cases
	testCases := []struct {
		name           string
		requestID      string
		ctx            context.Context
		requestBody    httpPkg.ResubmitAPIRequestRequest
		expectedStatus int
	}{
		{
			name:      "Resubmit denied API request with updates",
			requestID: requestIDs[2], // This is a denied request
			ctx:       ctxWithRequester1,
			requestBody: httpPkg.ResubmitAPIRequestRequest{
				Description:        "Updated description for resubmission",
				DocumentIDs:        []string{documentIDs[0]},
				RequiredTrackerIDs: []string{trackerIDs[0]},
				ProposedPolicyID:   policyIDs[1],
			},
			expectedStatus: 201,
		},
		{
			name:        "Resubmit denied API request without changes",
			requestID:   requestIDs[4], // This is another denied request
			ctx:         ctxWithRequester1,
			requestBody: httpPkg.ResubmitAPIRequestRequest{
				// Empty request - should copy from original
			},
			expectedStatus: 201,
		},
		{
			name:      "Resubmit approved API request",
			requestID: requestIDs[1], // This is an approved request
			ctx:       ctxWithRequester1,
			requestBody: httpPkg.ResubmitAPIRequestRequest{
				Description: "This should fail",
			},
			expectedStatus: 400, // Should fail because only denied requests can be resubmitted
		},
		{
			name:      "Resubmit as different user",
			requestID: requestIDs[2], // This is a denied request for requester_1
			ctx:       ctxWithRequester2,
			requestBody: httpPkg.ResubmitAPIRequestRequest{
				Description: "This should fail",
			},
			expectedStatus: 403, // Should fail because only the original requester can resubmit
		},
		{
			name:      "Resubmit non-existent API request",
			requestID: "non-existent-id",
			ctx:       ctxWithRequester1,
			requestBody: httpPkg.ResubmitAPIRequestRequest{
				Description: "This should fail",
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
			req := httptest.NewRequest("POST", "/api/requests/"+tc.requestID+"/resubmit", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")

			// Add the request ID as a path parameter
			req = req.WithContext(context.WithValue(req.Context(), httpPkg.PathParamContextKey, map[string]string{"id": tc.requestID}))

			// Create test response recorder
			w := httptest.NewRecorder()

			// Call the handler directly
			httpPkg.HandleResubmitAPIRequest(tc.ctx, w, req)

			// Check response status code
			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status code %d, got %d: %s", tc.expectedStatus, w.Code, w.Body.String())
			}

			// For successful responses, check the response
			if w.Code == 201 {
				var newRequest db.APIRequest
				if err := json.Unmarshal(w.Body.Bytes(), &newRequest); err != nil {
					t.Fatalf("Failed to parse response JSON: %v", err)
				}

				// Check that the new request has the correct status
				if newRequest.Status != "pending" {
					t.Errorf("Expected status to be pending, got %s", newRequest.Status)
				}

				// Check that the previous request ID is set
				if newRequest.PreviousRequestID == nil || *newRequest.PreviousRequestID != tc.requestID {
					t.Errorf("Expected previous request ID to be %s, got %v", tc.requestID, newRequest.PreviousRequestID)
				}

				// Check that the submission count is incremented
				// Get the original request to compare
				var originalCount int
				err := database.QueryRow("SELECT submission_count FROM api_requests WHERE id = ?", tc.requestID).Scan(&originalCount)
				if err != nil {
					t.Fatalf("Failed to get original submission count: %v", err)
				}

				if newRequest.SubmissionCount != originalCount+1 {
					t.Errorf("Expected submission count to be %d, got %d", originalCount+1, newRequest.SubmissionCount)
				}

				// Check that documents were associated correctly
				docCount, err := db.CountRequestDocuments(database, newRequest.ID)
				if err != nil {
					t.Fatalf("Failed to count request documents: %v", err)
				}

				expectedDocCount := len(tc.requestBody.DocumentIDs)
				if expectedDocCount == 0 {
					// If no docs specified, should copy from original
					expectedDocCount, err = db.CountRequestDocuments(database, tc.requestID)
					if err != nil {
						t.Fatalf("Failed to count original request documents: %v", err)
					}
				}

				if docCount != expectedDocCount {
					t.Errorf("Expected %d associated documents, got %d", expectedDocCount, docCount)
				}

				// Check that trackers were associated correctly
				trackerCount, err := db.CountRequestTrackers(database, newRequest.ID)
				if err != nil {
					t.Fatalf("Failed to count request trackers: %v", err)
				}

				expectedTrackerCount := len(tc.requestBody.RequiredTrackerIDs)
				if expectedTrackerCount == 0 {
					// If no trackers specified, should copy from original
					expectedTrackerCount, err = db.CountRequestTrackers(database, tc.requestID)
					if err != nil {
						t.Fatalf("Failed to count original request trackers: %v", err)
					}
				}

				if trackerCount != expectedTrackerCount {
					t.Errorf("Expected %d associated trackers, got %d", expectedTrackerCount, trackerCount)
				}
			}
		})
	}
}
