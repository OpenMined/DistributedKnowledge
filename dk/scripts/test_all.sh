#!/bin/bash
# test_all.sh - Unified test runner for the Distributed Knowledge codebase
# This script runs all tests in an organized manner with clear reporting

set -e

# Colors for better output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Function to print section headers
print_header() {
    echo -e "\n${BLUE}${BOLD}========================================================${NC}"
    echo -e "${BLUE}${BOLD}$1${NC}"
    echo -e "${BLUE}${BOLD}========================================================${NC}"
}

# Function to run tests and check for failures
run_test() {
    local test_cmd="$1"
    local test_name="$2"
    
    echo -e "\n${YELLOW}Running $test_name...${NC}"
    
    if eval "$test_cmd"; then
        echo -e "${GREEN}✓ $test_name tests passed${NC}"
        return 0
    else
        echo -e "${RED}✗ $test_name tests failed${NC}"
        return 1
    fi
}

# Function to help show usage
show_usage() {
    echo -e "${BOLD}Usage:${NC} $0 [options]"
    echo
    echo -e "${BOLD}Options:${NC}"
    echo "  --all               Run all tests (default if no option specified)"
    echo "  --unit              Run only unit tests"
    echo "  --integration       Run only integration tests"
    echo "  --db                Run only database tests"
    echo "  --http              Run only HTTP handler tests"
    echo "  --api-management    Run only API management tests"
    echo "  --usage-tracking    Run only usage tracking tests"
    echo "  --quick             Run a minimal subset of critical tests"
    echo "  --help              Display this help message"
    echo
    echo -e "${BOLD}Examples:${NC}"
    echo "  $0 --all            Run all tests"
    echo "  $0 --unit --db      Run unit tests for database components"
    echo "  $0 --quick          Run a quick subset of tests"
    exit 0
}

# Parse command line arguments
ALL=true
UNIT=false
INTEGRATION=false
DB=false
HTTP=false
API_MANAGEMENT=false
USAGE_TRACKING=false
QUICK=false

if [ $# -gt 0 ]; then
    ALL=false
    for arg in "$@"; do
        case $arg in
            --all)
                ALL=true
                ;;
            --unit)
                UNIT=true
                ;;
            --integration)
                INTEGRATION=true
                ;;
            --db)
                DB=true
                ;;
            --http)
                HTTP=true
                ;;
            --api-management)
                API_MANAGEMENT=true
                ;;
            --usage-tracking)
                USAGE_TRACKING=true
                ;;
            --quick)
                QUICK=true
                ;;
            --help)
                show_usage
                ;;
            *)
                echo -e "${RED}Unknown option: $arg${NC}"
                show_usage
                ;;
        esac
    done
fi

# Store failures
FAILURES=()

# Start test execution
print_header "Distributed Knowledge Test Suite"
echo -e "Started at: $(date)"

# Quick mode - run a minimal set of tests that are known to pass cleanly
if [ "$QUICK" = true ]; then
    print_header "Quick Test Mode"
    run_test "go test ./test/utils" "Test utils tests"
    if [ $? -ne 0 ]; then
        FAILURES+=("Test utils tests")
    fi

    run_test "go test ./http -run TestHandleGetDocuments" "HTTP handler tests"
    if [ $? -ne 0 ]; then
        FAILURES+=("HTTP handler tests")
    fi

    run_test "go test ./test/integration/api_management -run TestAPIManagement" "API Management basic test"
    if [ $? -ne 0 ]; then
        FAILURES+=("API Management basic test")
    fi
else
    # 1. Unit tests for the test/utils package
    if [ "$ALL" = true ] || ([ "$UNIT" = true ]); then
        print_header "1. Test Utilities Tests"
        if ! run_test "go test ./test/utils" "Test Utilities"; then
            FAILURES+=("Test Utilities Tests")
        fi
    fi

    # 2. Database Schema and Migration Tests
    if [ "$ALL" = true ] || ([ "$DB" = true ] && [ "$UNIT" = true ]); then
        print_header "2. Database Schema and Table Structure Tests"
        if ! run_test "go test ./db/... -run 'TestRunAPIMigrations|TestTableColumnDefinitions|TestForeignKeyRelationships|TestForeignKeyConstraints' -v" "Database Schema"; then
            FAILURES+=("Database Schema Tests")
        fi
    fi

    # 3. Database CRUD Tests
    if [ "$ALL" = true ] || ([ "$DB" = true ] && [ "$UNIT" = true ]); then
        print_header "3. Database CRUD Operation Tests"
        if ! run_test "go test ./db/... -run 'TestPolicyCRUD|TestAPICRUD|TestAPIRequestCRUD|TestDocumentAssociationCRUD|TestAPIUserAccessCRUD|TestAPIAccessLevels|TestPolicyChangeHistory|TestPolicyManagementCRUD' -v" "Database CRUD"; then
            FAILURES+=("Database CRUD Tests")
        fi
    fi

    # 4. Database Concurrency Tests
    if [ "$ALL" = true ] || ([ "$DB" = true ] && [ "$UNIT" = true ]); then
        print_header "4. Database Concurrency Tests"
        if ! run_test "go test ./db/... -run 'TestConcurrent|TestFixedTransactionHandling' -v" "Database Concurrency"; then
            FAILURES+=("Database Concurrency Tests")
        fi
    fi

    # 5. API Management Tests
    if [ "$ALL" = true ] || ([ "$API_MANAGEMENT" = true ] && [ "$UNIT" = true ]); then
        print_header "5. API Management Entity Tests"
        if ! run_test "go test ./test/api_entities_test/... -v" "API Management"; then
            FAILURES+=("API Management Tests")
        fi
    fi

    # 6. HTTP Handler Tests
    if [ "$ALL" = true ] || ([ "$HTTP" = true ] && [ "$UNIT" = true ]); then
        print_header "6. HTTP Handler Tests"
        if ! run_test "go test ./http/... -run 'TestGetDocumentType|TestHandleGetDocuments|TestUsageTrackingHandlersRegistration|TestPolicyEnforcementMiddleware' -v" "HTTP Handlers"; then
            FAILURES+=("HTTP Handler Tests")
        fi
    fi

    # 7. Policy Management Tests
    if [ "$ALL" = true ] || ([ "$API_MANAGEMENT" = true ] && [ "$UNIT" = true ]); then
        print_header "7. Policy Management Tests"
        if ! run_test "go test ./http/... -run 'TestPolicyHandlers' -v" "Policy Management"; then
            FAILURES+=("Policy Management Tests")
        fi
    fi

    # 8. User Access Management Tests
    if [ "$ALL" = true ] || ([ "$API_MANAGEMENT" = true ] && [ "$UNIT" = true ]); then
        print_header "8. User Access Management Tests"
        if ! run_test "go test ./db/... -run 'TestAPIUserAccess|TestListAPIUserAccess|TestGetAPIUserAccessByUserID|TestUpdateAPIUserAccess' -v && go test ./http/... -run 'TestHandle.*User.*' -v" "User Access Management"; then
            FAILURES+=("User Access Management Tests")
        fi
    fi

    # 9. Document Management Tests
    if [ "$ALL" = true ] || ([ "$API_MANAGEMENT" = true ] && [ "$UNIT" = true ]); then
        print_header "9. Document Management Tests"
        if ! run_test "go test ./db/... -run 'TestDocumentAssociationCRUD' -v && go test ./http/... -run 'TestHandleGetDocuments' -v" "Document Management"; then
            FAILURES+=("Document Management Tests")
        fi
    fi

    # 10. Usage Tracking Tests
    if [ "$ALL" = true ] || ([ "$USAGE_TRACKING" = true ] && [ "$UNIT" = true ]); then
        print_header "10. Usage Tracking and Quota Enforcement Tests"
        if ! run_test "go test ./db/... -run 'TestAPIUsageOperations|TestFixedAPIUsageSummaryOperations|TestFixedQuotaNotificationOperations|TestFixedUsageSummaryRefresh' -v" "Usage Tracking Database"; then
            FAILURES+=("Usage Tracking Database Tests")
        fi

        if ! run_test "go test ./http/... -run 'TestPolicyEnforcement|TestLimitExceeded|TestApproachingLimit' -v" "Policy Enforcement"; then
            FAILURES+=("Policy Enforcement Tests")
        fi
    fi

    # 11. Standalone Tests
    if [ "$ALL" = true ] || [ "$UNIT" = true ]; then
        print_header "11. Standalone Tests"
        if ! run_test "go test -v" "Standalone Tests"; then
            FAILURES+=("Standalone Tests")
        fi
    fi

    # 12. Integration Tests
    if [ "$ALL" = true ] || [ "$INTEGRATION" = true ]; then
        print_header "12. Integration Tests"
        if ! run_test "go test ./test/integration/... -v" "Integration Tests"; then
            FAILURES+=("Integration Tests")
        fi
    fi

    # 13. API Management Integration Tests
    if [ "$ALL" = true ] || [ "$INTEGRATION" = true ] || [ "$API_MANAGEMENT" = true ]; then
        print_header "13. API Management Integration Tests"
        if ! run_test "go test ./test/integration/api_management/... -v" "API Management Integration"; then
            FAILURES+=("API Management Integration Tests")
        fi
    fi

    # 14. Usage Tracking Integration Tests
    if [ "$ALL" = true ] || [ "$INTEGRATION" = true ] || [ "$USAGE_TRACKING" = true ]; then
        print_header "14. Usage Tracking Integration Tests"
        if ! run_test "go test ./test/integration/usage_tracking/... -v" "Usage Tracking Integration"; then
            FAILURES+=("Usage Tracking Integration Tests")
        fi
    fi
fi

# Report results
print_header "Test Suite Summary"
echo -e "Finished at: $(date)"

if [ ${#FAILURES[@]} -eq 0 ]; then
    echo -e "${GREEN}All tests passed successfully!${NC}"
    exit 0
else
    echo -e "${RED}The following test groups failed:${NC}"
    for failure in "${FAILURES[@]}"; do
        echo -e "${RED}- $failure${NC}"
    done
    exit 1
fi