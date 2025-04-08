.PHONY: test test-integration 

# Default target
all: test

# Main test target
test: test-integration

# Integration tests
test-integration:
	@echo "Running integration tests..."
	go test -v ./integration
