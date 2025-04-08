.PHONY: test test-unit 

# Default target
all: test

# Main test target
test: test-unit test-integration

# Unit tests
test-unit:
	@echo "Running unit tests..."
	go test -v ./lib 
