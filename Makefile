# =============================================================================
# Free Chat - Makefile
# =============================================================================

.PHONY: all build run test test-coverage test-integration fmt lint clean \
		mysql-start mysql-stop mysql-restart mysql-status mysql-create \
		mysql-drop mysql-reset migrate migrate-test migrate-up migrate-down \
		deps tidy generate help docker-build docker-run docker-stop

# -----------------------------------------------------------------------------
# Variables
# -----------------------------------------------------------------------------

APP_NAME := free-chat
BUILD_DIR := ./bin
MAIN_FILE := ./cmd/server/main.go
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

# Go
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOFMT := gofmt
GOVET := $(GOCMD) vet

# MySQL
MYSQL_USER ?= root
MYSQL_PASS ?= passwd
MYSQL_HOST ?= localhost
MYSQL_PORT ?= 3306
DB_NAME ?= chat_db
DB_TEST ?= chat_db_test

# Docker
DOCKER_IMAGE := $(APP_NAME)
DOCKER_TAG := $(VERSION)

# Colors
GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
NC := \033[0m # No Color

# -----------------------------------------------------------------------------
# Main targets
# -----------------------------------------------------------------------------

all: fmt lint test build
	@echo -e "$(GREEN) All checks passed$(NC)"

build:
	@echo -e "$(YELLOW)Building...$(NC)"
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_FILE)
	@echo -e "$(GREEN) Build complete: $(BUILD_DIR)/$(APP_NAME)$(NC)"

run: build
	@echo -e "$(YELLOW)Running server...$(NC)"
	$(BUILD_DIR)/$(APP_NAME)

dev:
	@echo -e "$(YELLOW)Running in development mode...$(NC)"
	$(GOCMD) run $(MAIN_FILE)

# -----------------------------------------------------------------------------
# Testing
# -----------------------------------------------------------------------------

test:
	@echo -e "$(YELLOW)Running tests...$(NC)"
	$(GOTEST) -v -race -count=1 -short ./...
	@echo -e "$(GREEN) Tests passed$(NC)"

test-coverage:
	@echo -e "$(YELLOW)Running tests with coverage...$(NC)"
	$(GOTEST) -v -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo -e "$(GREEN) Coverage report: coverage.html$(NC)"

test-integration:
	@echo -e "$(YELLOW)Running integration tests...$(NC)"
	$(GOTEST) -v -race -count=1 -tags=integration ./...
	@echo -e "$(GREEN) Integration tests passed$(NC)"

test-verbose:
	$(GOTEST) -v -race -count=1 ./...

# -----------------------------------------------------------------------------
# Code quality
# -----------------------------------------------------------------------------

fmt:
	@echo -e "$(YELLOW)Formatting code...$(NC)"
	$(GOFMT) -w .
	@echo -e "$(GREEN) Code formatted$(NC)"

lint:
	@echo -e "$(YELLOW)Linting...$(NC)"
	$(GOVET) ./...
	@if command -v golangci-lint > /dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo -e "$(YELLOW)golangci-lint not installed, skipping advanced linting$(NC)"; \
		echo -e "$(YELLOW)Install with: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin$(NC)"; \
	fi
	@echo -e "$(GREEN) Lint passed$(NC)"

vet:
	$(GOVET) ./...

# -----------------------------------------------------------------------------
# Dependencies
# -----------------------------------------------------------------------------

deps:
	@echo -e "$(YELLOW)Downloading dependencies...$(NC)"
	$(GOGET) -v -d ./...

tidy:
	@echo -e "$(YELLOW)Tidying modules...$(NC)"
	$(GOCMD) mod tidy
	@echo -e "$(GREEN) Modules tidied$(NC)"

generate:
	@echo -e "$(YELLOW)Generating code...$(NC)"
	$(GOCMD) generate ./...

# -----------------------------------------------------------------------------
# Database
# -----------------------------------------------------------------------------

mysql-start:
	@echo -e "$(YELLOW)Starting MySQL...$(NC)"
	@if command -v mysql.server > /dev/null 2>&1; then \
		mysql.server start; \
	elif command -v systemctl > /dev/null 2>&1; then \
		sudo systemctl start mysql; \
	elif command -v service > /dev/null 2>&1; then \
		sudo service mysql start; \
	else \
		echo -e "$(RED)Cannot determine how to start MySQL$(NC)"; \
		exit 1; \
	fi
	@echo -e "$(GREEN) MySQL started$(NC)"

mysql-stop:
	@echo -e "$(YELLOW)Stopping MySQL...$(NC)"
	@if command -v mysql.server > /dev/null 2>&1; then \
		mysql.server stop; \
	elif command -v systemctl > /dev/null 2>&1; then \
		sudo systemctl stop mysql; \
	elif command -v service > /dev/null 2>&1; then \
		sudo service mysql stop; \
	fi
	@echo -e "$(GREEN) MySQL stopped$(NC)"

mysql-restart: mysql-stop mysql-start

mysql-status:
	@if command -v mysql.server > /dev/null 2>&1; then \
		mysql.server status; \
	elif command -v systemctl > /dev/null 2>&1; then \
		sudo systemctl status mysql --no-pager; \
	else \
		echo "MySQL status unknown"; \
	fi

mysql-create:
	@echo -e "$(YELLOW)Creating databases...$(NC)"
	mysql -u $(MYSQL_USER) $(if $(MYSQL_PASS),-p$(MYSQL_PASS),) -e "CREATE DATABASE IF NOT EXISTS $(DB_NAME);"
	mysql -u $(MYSQL_USER) $(if $(MYSQL_PASS),-p$(MYSQL_PASS),) -e "CREATE DATABASE IF NOT EXISTS $(DB_TEST);"
	@echo -e "$(GREEN) Databases created$(NC)"

mysql-drop:
	@echo -e "$(RED)Dropping databases...$(NC)"
	mysql -u $(MYSQL_USER) $(if $(MYSQL_PASS),-p$(MYSQL_PASS),) -e "DROP DATABASE IF EXISTS $(DB_NAME);"
	mysql -u $(MYSQL_USER) $(if $(MYSQL_PASS),-p$(MYSQL_PASS),) -e "DROP DATABASE IF EXISTS $(DB_TEST);"
	@echo -e "$(GREEN) Databases dropped$(NC)"

mysql-reset: mysql-drop mysql-create migrate

# -----------------------------------------------------------------------------
# Migrations
# -----------------------------------------------------------------------------

migrate: migrate-up

migrate-up:
	@echo -e "$(YELLOW)Running migrations...$(NC)"
	mysql -u $(MYSQL_USER) $(if $(MYSQL_PASS),-p$(MYSQL_PASS),) $(DB_NAME) < migrations/001_init.sql
	mysql -u $(MYSQL_USER) $(if $(MYSQL_PASS),-p$(MYSQL_PASS),) $(DB_NAME) < migrations/002_messages.sql
	mysql -u $(MYSQL_USER) $(if $(MYSQL_PASS),-p$(MYSQL_PASS),) $(DB_NAME) < migrations/003_media.sql
	@echo -e "$(GREEN) Migrations complete$(NC)"

migrate-test:
	@echo -e "$(YELLOW)Running test migrations...$(NC)"
	mysql -u $(MYSQL_USER) $(if $(MYSQL_PASS),-p$(MYSQL_PASS),) $(DB_TEST) < migrations/001_init.sql
	mysql -u $(MYSQL_USER) $(if $(MYSQL_PASS),-p$(MYSQL_PASS),) $(DB_TEST) < migrations/002_messages.sql
	mysql -u $(MYSQL_USER) $(if $(MYSQL_PASS),-p$(MYSQL_PASS),) $(DB_TEST) < migrations/003_media.sql
	@echo -e "$(GREEN) Test migrations complete$(NC)"

# -----------------------------------------------------------------------------
# Docker
# -----------------------------------------------------------------------------

docker-build:
	@echo -e "$(YELLOW)Building Docker image...$(NC)"
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_IMAGE):latest
	@echo -e "$(GREEN) Docker image built$(NC)"

docker-run: docker-build
	@echo -e "$(YELLOW)Running Docker container...$(NC)"
	docker run -d --name $(APP_NAME) -p 8080:8080 --env-file .env $(DOCKER_IMAGE):latest
	@echo -e "$(GREEN) Container running$(NC)"

docker-stop:
	@echo -e "$(YELLOW)Stopping Docker container...$(NC)"
	docker stop $(APP_NAME) || true
	docker rm $(APP_NAME) || true
	@echo -e "$(GREEN) Container stopped$(NC)"

# -----------------------------------------------------------------------------
# Cleanup
# -----------------------------------------------------------------------------

clean:
	@echo -e "$(YELLOW)Cleaning...$(NC)"
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	rm -rf uploads/images/* uploads/gifs/* uploads/voice/*
	@echo -e "$(GREEN) Cleaned$(NC)"

# -----------------------------------------------------------------------------
# Help
# -----------------------------------------------------------------------------

help:
	@echo ""
	@echo -e "$(GREEN)Free Chat - Available Commands$(NC)"
	@echo ""
	@echo "  $(YELLOW)Build & Run:$(NC)"
	@echo "	make build		  - Build the application"
	@echo "	make run			- Build and run the application"
	@echo "	make dev			- Run with go run (no build)"
	@echo "	make all			- Format, lint, test, and build"
	@echo ""
	@echo "  $(YELLOW)Testing:$(NC)"
	@echo "	make test		   - Run unit tests"
	@echo "	make test-coverage  - Run tests with coverage report"
	@echo "	make test-integration - Run integration tests"
	@echo "	make test-verbose   - Run tests with verbose output"
	@echo ""
	@echo "  $(YELLOW)Code Quality:$(NC)"
	@echo "	make fmt			- Format code with gofmt"
	@echo "	make lint		   - Run linters"
	@echo "	make vet			- Run go vet"
	@echo ""
	@echo "  $(YELLOW)Dependencies:$(NC)"
	@echo "	make deps		   - Download dependencies"
	@echo "	make tidy		   - Tidy go modules"
	@echo "	make generate	   - Run go generate"
	@echo ""
	@echo "  $(YELLOW)Database:$(NC)"
	@echo "	make mysql-start	- Start MySQL server"
	@echo "	make mysql-stop	 - Stop MySQL server"
	@echo "	make mysql-restart  - Restart MySQL server"
	@echo "	make mysql-status   - Show MySQL status"
	@echo "	make mysql-create   - Create databases"
	@echo "	make mysql-drop	 - Drop databases"
	@echo "	make mysql-reset	- Drop, create, and migrate"
	@echo ""
	@echo "  $(YELLOW)Migrations:$(NC)"
	@echo "	make migrate		- Run all migrations"
	@echo "	make migrate-test   - Run migrations on test DB"
	@echo ""
	@echo "  $(YELLOW)Docker:$(NC)"
	@echo "	make docker-build   - Build Docker image"
	@echo "	make docker-run	 - Build and run container"
	@echo "	make docker-stop	- Stop and remove container"
	@echo ""
	@echo "  $(YELLOW)Other:$(NC)"
	@echo "	make clean		  - Remove build artifacts"
	@echo "	make help		   - Show this help"
	@echo ""