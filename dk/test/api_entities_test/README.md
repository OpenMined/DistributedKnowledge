# API Entities Endpoint Tests

This directory contains tests for the API Entities endpoints as described in the implementation document (`implementation/2_api_entities_endpoints.md`).

## Overview

These tests verify the functionality of the following endpoints:

1. `GET /api/apis` - List all APIs
2. `GET /api/apis/:id` - Get details of a specific API
3. `POST /api/apis` - Create a new API
4. `PATCH /api/apis/:id` - Update an existing API
5. `POST /api/apis/:id/deprecate` - Mark an API as deprecated
6. `DELETE /api/apis/:id` - Delete an API

## Test Setup

The tests use:

- In-memory SQLite database for testing
- Mock implementations of the API handlers
- HTTP test tools from Go's standard library

## Running the Tests

Execute the tests using the unified test script:

```bash
cd /home/ubuntu/workspace/DistributedKnowledge/dk
./test_all.sh --api-management
```

Alternatively, use the `just` command:

```bash
cd /home/ubuntu/workspace/DistributedKnowledge/dk
just test-api
```

You can also run the tests directly:

```bash
cd /home/ubuntu/workspace/DistributedKnowledge/dk/test/api_entities_test
go test -v
```

## Test Coverage

The tests cover:

- Various query parameters for API listing
- Filtering APIs by status
- Pagination and sorting
- Error handling for non-existent resources
- Validation of request bodies
- All required status codes as per the specification

## Implementation Notes

These tests use mock handlers that simulate the expected behavior of the actual endpoints. This allows testing the API contract without needing the full implementation. The actual database operations are not performed during these tests.

When implementing the actual handlers, they should conform to the behavior verified by these tests.

## Next Steps

After implementing the actual handlers:

1. Update the tests to use the real handlers instead of mocks
2. Add integration tests that test the full stack
3. Implement tests for the next set of endpoints (API Request endpoints)