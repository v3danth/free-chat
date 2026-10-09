# =============================================================================
# Drift - Makefile. Run `make help` for the list.
# =============================================================================

.PHONY: start build run dev test test-coverage fmt lint vet tidy \
        db-create db-drop db-reset migrate migrate-test geoip clean help

# Database settings come from .env (the same ones the server uses).
-include .env

BUILD_DIR      := ./bin
APP            := $(BUILD_DIR)/server
MYSQL_HOST     ?= localhost
MYSQL_PORT     ?= 3306
MYSQL_USER     ?= root
MYSQL_DATABASE ?= chat_db
DB_TEST        ?= chat_db_test
# The password goes through the environment, never the command line.
MYSQL          := MYSQL_PWD='$(MYSQL_PASSWORD)' mysql -h $(MYSQL_HOST) -P $(MYSQL_PORT) -u $(MYSQL_USER)

GREEN  := \033[0;32m
YELLOW := \033[0;33m
NC     := \033[0m
say     = @printf '%b\n' "$(YELLOW)$(1)$(NC)"
done    = @printf '%b\n' "$(GREEN)$(1)$(NC)"

# -----------------------------------------------------------------------------
# Run
# -----------------------------------------------------------------------------

start: ## Check MySQL, the schema and the port, then build and run
	@scripts/start.sh

build: ## Build the server into bin/server
	$(call say,Building...)
	@mkdir -p $(BUILD_DIR)
	go build -o $(APP) ./cmd/server
	$(call done,Built $(APP))

run: build ## Build and run without the start checks
	$(APP)

dev: ## Run with go run
	go run ./cmd/server

# -----------------------------------------------------------------------------
# Quality
# -----------------------------------------------------------------------------

test: ## Run all tests with the race detector (database tests use $(DB_TEST))
	@echo "go test -race -count=1 ./...  (database tests on $(DB_TEST))"
	@TEST_MYSQL_DATABASE=$(DB_TEST) MYSQL_HOST=$(MYSQL_HOST) MYSQL_PORT=$(MYSQL_PORT) MYSQL_USER=$(MYSQL_USER) \
	 MYSQL_PASSWORD='$(MYSQL_PASSWORD)' go test -race -count=1 ./...

test-coverage: ## Write coverage.html
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	$(call done,Coverage report: coverage.html)

fmt: ## Format all Go code
	gofmt -w .

vet: ## Run go vet
	go vet ./...

lint: vet ## go vet, plus golangci-lint when installed
	@if command -v golangci-lint >/dev/null 2>&1; then golangci-lint run ./...; \
	else printf '%b\n' "$(YELLOW)golangci-lint not installed, ran go vet only$(NC)"; fi

tidy: ## Tidy go.mod
	go mod tidy

# -----------------------------------------------------------------------------
# Database
# -----------------------------------------------------------------------------

db-create: ## Create the app and test databases
	@$(MYSQL) -e "CREATE DATABASE IF NOT EXISTS $(MYSQL_DATABASE) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci; CREATE DATABASE IF NOT EXISTS $(DB_TEST) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;"
	$(call done,Databases ready)

db-drop: ## Drop the app and test databases (deletes all data)
	@$(MYSQL) -e "DROP DATABASE IF EXISTS $(MYSQL_DATABASE); DROP DATABASE IF EXISTS $(DB_TEST);"
	$(call done,Databases dropped)

db-reset: db-drop db-create migrate migrate-test ## Drop, create and migrate both databases

migrate: ## Apply migrations/*.sql to the app database
	@for f in migrations/*.sql; do echo "  $$f"; $(MYSQL) $(MYSQL_DATABASE) < $$f || exit 1; done
	$(call done,Migrations applied)

migrate-test: ## Apply migrations/*.sql to the test database
	@for f in migrations/*.sql; do echo "  $$f"; $(MYSQL) $(DB_TEST) < $$f || exit 1; done
	$(call done,Test migrations applied)

# -----------------------------------------------------------------------------
# Other
# -----------------------------------------------------------------------------

# DB-IP "IP to Country Lite", CC BY 4.0, published monthly.
GEOIP_MONTH ?= $(shell date +%Y-%m)

geoip: ## Download the country database used for flags
	$(call say,Downloading DB-IP country database ($(GEOIP_MONTH))...)
	@mkdir -p data
	curl -fsSL "https://download.db-ip.com/free/dbip-country-lite-$(GEOIP_MONTH).mmdb.gz" | gunzip > data/dbip-country-lite.mmdb
	$(call done,Saved data/dbip-country-lite.mmdb)

clean: ## Remove build output, coverage files and uploaded images
	rm -rf $(BUILD_DIR) coverage.out coverage.html
	rm -rf uploads/full/* uploads/thumb/*

help: ## Show this list
	@grep -hE '^[a-z-]+:.*## ' Makefile | awk -F':.*## ' '{printf "  %-14s %s\n", $$1, $$2}'
