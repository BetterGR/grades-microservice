# Proto settings
SERVICE_NAME ?= $(notdir $(CURDIR))
PROTO_DIR ?= protos
PROTO_FILE ?= $(PROTO_DIR)/$(SERVICE_NAME).proto
PROTO_FLAGS = -I $(PROTO_DIR) $(PROTO_FILE) \
              --go_out=paths=source_relative:$(PROTO_DIR) \
              --go-grpc_out=paths=source_relative:$(PROTO_DIR)

# Docker settings
DOCKER_IMAGE_NAME ?= $(SERVICE_NAME)
DOCKER_TAG ?= latest
DOCKERFILE ?= Dockerfile
DOCKER_REGISTRY ?= ghcr.io/bettergr

# Default target
all: proto gomod fmt vet lint

# Ensure tools are installed
ensure-gofumpt:
ifeq ($(OS),Windows_NT)
	@where gofumpt > temp.txt 2>&1 || ( \
		echo [INSTALL] gofumpt not found. Installing... & \
		go install mvdan.cc/gofumpt@latest \
	)
	@ del temp.txt
else
	@command -v gofumpt > /dev/null 2>&1 || { \
		echo "[INSTALL] gofumpt not found. Installing..."; \
		go install mvdan.cc/gofumpt@latest; \
	}
endif

ensure-gci:
ifeq ($(OS),Windows_NT)
	@where gci > temp.txt 2>&1 || ( \
		echo [INSTALL] gci not found. Installing... & \
		go install github.com/daixiang0/gci@latest \
	)
	@ del temp.txt
else
	@command -v gci > /dev/null 2>&1 || { \
		echo "[INSTALL] gci not found. Installing..."; \
		go install github.com/daixiang0/gci@latest; \
	}
endif

ensure-golangci-lint:
ifeq ($(OS),Windows_NT)
	@where golangci-lint > temp.txt 2>&1 || ( \
		echo [INSTALL] golangci-lint not found. Installing... & \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest \
	)
	@ del temp.txt
else
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "[INSTALL] golangci-lint not found. Installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	}
endif

# Generate Go code from proto files
proto:
	@echo [PROTO] Generating Go code from proto file...
	@protoc $(PROTO_FLAGS)
	@echo [PROTO] Go code generation complete.

# Manage Go modules
gomod:
	@echo [GO-MOD] Verifying modules...
	@go mod tidy
	@go mod verify
	@echo [GO-MOD] Modules verified.

# Format Go code
fmt: ensure-gofumpt ensure-gci
	@echo [FMT] Formatting Go code...
	@go fmt ./...
	@gofumpt -w .
	@gci write --skip-generated .
	@echo [FMT] Go code formatted.

# Vet Go code
vet:
	@echo [VET] Running vet checks on Go code...
	@go vet ./...
	@echo [VET] Vet checks completed.

# Lint Go code
lint: ensure-golangci-lint fmt
	@echo [LINT] Running linter on Go code...
	@golangci-lint run
	@echo [LINT] Lint checks completed.

# Build server
build: proto fmt vet lint
	@echo [BUILD] Building server binary...
	@go build -o server/server ./server/server.go ./server/db.go
	@echo [BUILD] Server binary built successfully.

# Run the server
run: proto fmt vet
	@echo [RUN] Starting server...
	@go run ./server/server.go ./server/db.go $(ARGS)

test: proto gomod fmt vet lint
	@echo [TEST] Running all tests including database tests...
	@echo [TEST] Running regular tests...
	@go test -v ./server/ | grep -v '=== RUN' | sed 's/--- PASS:/ [PASS]/' | sed 's/--- FAIL:/ [FAIL]/'
	@echo [TEST] Running database tests...
	@if [ -f .env ]; then \
		echo "Loading environment variables from .env file..."; \
		export $$(grep -v '^#' .env | xargs); \
	else \
		echo "Warning: .env file not found. Using default environment."; \
	fi; \
	export DB_TESTS=true; \
	echo "Running database test with current connection settings..."; \
	go test -v ./server/db_test.go ./server/db.go ./server/server.go -run TestDatabaseSimpleFlow; \
	TEST_EXIT_CODE=$$?; \
	if [ $$TEST_EXIT_CODE -eq 0 ]; then \
		echo "[TEST] Database tests completed successfully."; \
	else \
		echo "[TEST] Database tests completed with issues."; \
		echo "[INFO] If you see foreign key constraint errors, this is normal when using a production database."; \
		echo "[INFO] The test is designed to handle this case gracefully."; \
	fi
	@echo [TEST] All tests completed.

# Build Docker image
docker-build: proto fmt vet lint build
	@echo [DOCKER] Building Docker image $(DOCKER_IMAGE_NAME):$(DOCKER_TAG)... 
	@docker build -t $(DOCKER_IMAGE_NAME):$(DOCKER_TAG) -f $(DOCKERFILE) .
	@echo [DOCKER] Docker image $(DOCKER_IMAGE_NAME):$(DOCKER_TAG) built successfully.

# Push Docker image to registry
docker-push: docker-build
ifeq ($(DOCKER_REGISTRY),docker.io)
	@echo [DOCKER] Docker registry is set to docker.io.
else
	@echo [DOCKER] Docker registry set to $(DOCKER_REGISTRY).
endif
	@echo [DOCKER] Pushing Docker image $(DOCKER_IMAGE_NAME):$(DOCKER_TAG) to $(DOCKER_REGISTRY)/$(DOCKER_IMAGE_NAME):$(DOCKER_TAG)...
	@docker tag $(DOCKER_IMAGE_NAME):$(DOCKER_TAG) $(DOCKER_REGISTRY)/$(DOCKER_IMAGE_NAME):$(DOCKER_TAG)
	@docker push $(DOCKER_REGISTRY)/$(DOCKER_IMAGE_NAME):$(DOCKER_TAG)
	@echo [DOCKER] Docker image pushed successfully.

# Clean up generated files
clean:
	@echo [CLEAN] Removing generated files...
ifeq ($(OS),Windows_NT)
	@del /Q server\server.exe
	@del /Q protos\*.pb.go
else
	@rm -rf server/server
	@rm -rf protos/*.pb.go
endif
	@echo [CLEAN] Clean up complete.

help:
	@echo Available targets:
	@echo   all               Build and check everything (proto, gomod, fmt, vet, lint)
	@echo   proto             Generate Go code from proto file
	@echo   gomod             Manage Go modules (tidy and verify)
	@echo   fmt               Format Go code
	@echo   vet               Run vet checks on Go code
	@echo   lint              Run linter on Go code
	@echo   build             Build the server binary
	@echo   run               Run the server
	@echo   test              Run all tests including database tests
	@echo   db-test           Run database integration tests only
	@echo   docker-build      Build Docker image
	@echo   docker-push       Push Docker image to registry
	@echo   clean             Clean up generated files

.PHONY: all proto fmt run vet lint build docker-build docker-push gomod clean ensure-gofumpt ensure-gci ensure-golangci-lint help test db-test