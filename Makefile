# =============================================================================
# Free Chat - Makefile
# =============================================================================

.PHONY: all build run test test-coverage test-integration fmt lint clean \
		mysql-start mysql-stop mysql-restart mysql-status mysql-create \
		mysql-drop mysql-reset migrate migrate-test migrate-up migrate-down \
		deps tidy generate help docker-build docker-run docker-stop geoip

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
	@printf '%b\n' "$(GREEN) All checks passed$(NC)"

build:
	@printf '%b\n' "$(YELLOW)Building...$(NC)"
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_FILE)
	@printf '%b\n' "$(GREEN) Build complete: $(BUILD_DIR)/$(APP_NAME)$(NC)"

run: build
	@printf '%b\n' "$(YELLOW)Running server...$(NC)"
	$(BUILD_DIR)/$(APP_NAME)

dev:
	@printf '%b\n' "$(YELLOW)Running in development mode...$(NC)"
	$(GOCMD) run $(MAIN_FILE)

# -----------------------------------------------------------------------------
# Testing
# -----------------------------------------------------------------------------

test:
	@printf '%b\n' "$(YELLOW)Running tests...$(NC)"
	$(GOTEST) -v -race -count=1 -short ./...
	@printf '%b\n' "$(GREEN) Tests passed$(NC)"

test-coverage:
	@printf '%b\n' "$(YELLOW)Running tests with coverage...$(NC)"
	$(GOTEST) -v -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@printf '%b\n' "$(GREEN) Coverage report: coverage.html$(NC)"

test-integration:
	@printf '%b\n' "$(YELLOW)Running integration tests...$(NC)"
	$(GOTEST) -v -race -count=1 -tags=integration ./...
	@printf '%b\n' "$(GREEN) Integration tests passed$(NC)"

test-verbose:
	$(GOTEST) -v -race -count=1 ./...

# -----------------------------------------------------------------------------
# Code quality
# -----------------------------------------------------------------------------

fmt:
	@printf '%b\n' "$(YELLOW)Formatting code...$(NC)"
	$(GOFMT) -w .
	@printf '%b\n' "$(GREEN) Code formatted$(NC)"

lint:
	@printf '%b\n' "$(YELLOW)Linting...$(NC)"
	$(GOVET) ./...
	@if command -v golangci-lint > /dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo -e "$(YELLOW)golangci-lint not installed, skipping advanced linting$(NC)"; \
		echo -e "$(YELLOW)Install with: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin$(NC)"; \
	fi
	@printf '%b\n' "$(GREEN) Lint passed$(NC)"

vet:
	$(GOVET) ./...

# -----------------------------------------------------------------------------
# Dependencies
# -----------------------------------------------------------------------------

deps:
	@printf '%b\n' "$(YELLOW)Downloading dependencies...$(NC)"
	$(GOGET) -v -d ./...

tidy:
	@printf '%b\n' "$(YELLOW)Tidying modules...$(NC)"
	$(GOCMD) mod tidy
	@printf '%b\n' "$(GREEN) Modules tidied$(NC)"

generate:
	@printf '%b\n' "$(YELLOW)Generating code...$(NC)"
	$(GOCMD) generate ./...

# -----------------------------------------------------------------------------
# Database
# -----------------------------------------------------------------------------

mysql-start:
	@printf '%b\n' "$(YELLOW)Starting MySQL...$(NC)"
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
	@printf '%b\n' "$(GREEN) MySQL started$(NC)"

mysql-stop:
	@printf '%b\n' "$(YELLOW)Stopping MySQL...$(NC)"
	@if command -v mysql.server > /dev/null 2>&1; then \
		mysql.server stop; \
	elif command -v systemctl > /dev/null 2>&1; then \
		sudo systemctl stop mysql; \
	elif command -v service > /dev/null 2>&1; then \
		sudo service mysql stop; \
	fi
	@printf '%b\n' "$(GREEN) MySQL stopped$(NC)"

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
	@printf '%b\n' "$(YELLOW)Creating databases...$(NC)"
	mysql -u $(MYSQL_USER) $(if $(MYSQL_PASS),-p$(MYSQL_PASS),) -e "CREATE DATABASE IF NOT EXISTS $(DB_NAME);"
	mysql -u $(MYSQL_USER) $(if $(MYSQL_PASS),-p$(MYSQL_PASS),) -e "CREATE DATABASE IF NOT EXISTS $(DB_TEST);"
	@printf '%b\n' "$(GREEN) Databases created$(NC)"

mysql-drop:
	@printf '%b\n' "$(RED)Dropping databases...$(NC)"
	mysql -u $(MYSQL_USER) $(if $(MYSQL_PASS),-p$(MYSQL_PASS),) -e "DROP DATABASE IF EXISTS $(DB_NAME);"
	mysql -u $(MYSQL_USER) $(if $(MYSQL_PASS),-p$(MYSQL_PASS),) -e "DROP DATABASE IF EXISTS $(DB_TEST);"
	@printf '%b\n' "$(GREEN) Databases dropped$(NC)"

mysql-reset: mysql-drop mysql-create migrate

# -----------------------------------------------------------------------------
# Migrations
# -----------------------------------------------------------------------------

migrate: migrate-up

migrate-up:
	@printf '%b\n' "$(YELLOW)Running migrations...$(NC)"
	@for f in migrations/*.sql; do echo "  $$f"; mysql -u $(MYSQL_USER) $(if $(MYSQL_PASS),-p$(MYSQL_PASS),) $(DB_NAME) < $$f || exit 1; done
	@printf '%b\n' "$(GREEN) Migrations complete$(NC)"

migrate-test:
	@printf '%b\n' "$(YELLOW)Running test migrations...$(NC)"
	@for f in migrations/*.sql; do echo "  $$f"; mysql -u $(MYSQL_USER) $(if $(MYSQL_PASS),-p$(MYSQL_PASS),) $(DB_TEST) < $$f || exit 1; done
	@printf '%b\n' "$(GREEN) Test migrations complete$(NC)"

# -----------------------------------------------------------------------------
# Docker
# -----------------------------------------------------------------------------

docker-build:
	@printf '%b\n' "$(YELLOW)Building Docker image...$(NC)"
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_IMAGE):latest
	@printf '%b\n' "$(GREEN) Docker image built$(NC)"

docker-run: docker-build
	@printf '%b\n' "$(YELLOW)Running Docker container...$(NC)"
	docker run -d --name $(APP_NAME) -p 8080:8080 --env-file .env $(DOCKER_IMAGE):latest
	@printf '%b\n' "$(GREEN) Container running$(NC)"

docker-stop:
	@printf '%b\n' "$(YELLOW)Stopping Docker container...$(NC)"
	docker stop $(APP_NAME) || true
	docker rm $(APP_NAME) || true
	@printf '%b\n' "$(GREEN) Container stopped$(NC)"

# -----------------------------------------------------------------------------
# GeoIP (country flags). DB-IP "IP to Country Lite", CC BY 4.0, monthly.
# -----------------------------------------------------------------------------

GEOIP_MONTH ?= $(shell date +%Y-%m)

geoip:
	@printf '%b\n' "$(YELLOW)Downloading DB-IP country database ($(GEOIP_MONTH))...$(NC)"
	@mkdir -p data
	curl -fsSL "https://download.db-ip.com/free/dbip-country-lite-$(GEOIP_MONTH).mmdb.gz" | gunzip > data/dbip-country-lite.mmdb
	@printf '%b\n' "$(GREEN) Saved data/dbip-country-lite.mmdb$(NC)"

# -----------------------------------------------------------------------------
# Cleanup
# -----------------------------------------------------------------------------

clean:
	@printf '%b\n' "$(YELLOW)Cleaning...$(NC)"
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	rm -rf uploads/full/* uploads/thumb/* uploads/blur/*
	@printf '%b\n' "$(GREEN) Cleaned$(NC)"

# -----------------------------------------------------------------------------
# Help
# -----------------------------------------------------------------------------

help:
	@echo ""
	@printf '%b\n' "$(GREEN)Free Chat - Available Commands$(NC)"
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
	@echo "	make geoip		  - Download the country database for flags"
	@echo "	make clean		  - Remove build artifacts"
	@echo "	make help		   - Show this help"
	@echo ""