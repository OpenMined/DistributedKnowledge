package api_entities

import (
	"testing"
)

// TestAPIEntitiesAndRequests runs all API entity and request tests in one suite
func TestAPIEntitiesAndRequests(t *testing.T) {
	t.Run("GetAPIs", TestGetAPIs)
	t.Run("GetAPIDetails", TestGetAPIDetails)
	t.Run("CreateAPI", TestCreateAPI)
	t.Run("UpdateAPI", TestUpdateAPI)
	t.Run("DeprecateAPI", TestDeprecateAPI)
	t.Run("DeleteAPI", TestDeleteAPI)

	t.Run("GetAPIRequests", TestGetAPIRequests)
	t.Run("GetAPIRequestDetails", TestGetAPIRequestDetails)
	t.Run("CreateAPIRequest", TestCreateAPIRequest)
	t.Run("UpdateAPIRequestStatus", TestUpdateAPIRequestStatus)
	t.Run("ResubmitAPIRequest", TestResubmitAPIRequest)
}

// TestMain is a test entry point that can be used to run all tests
func TestMain(t *testing.T) {
	TestAPIEntitiesAndRequests(t)
}
