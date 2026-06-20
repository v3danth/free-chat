MIGRATION=migrations/001_init.sql
RESET=scripts/reset_db.sql

DB_NAME=chat_db
TEST_DB=chat_db_test
DB_USER=root
DB_PASS=passwd

.PHONY: test run db-reset db-migrate db-test-reset db-test-migrate db-test-setup

# --------------------
# RUN APP
# --------------------
run:
	go run ./cmd/server

# --------------------
# PROD DB
# --------------------
db-reset:
	mysql -u $(DB_USER) -p$(DB_PASS) $(DB_NAME) < $(RESET)

db-migrate:
	mysql -u $(DB_USER) -p$(DB_PASS) $(DB_NAME) < $(MIGRATION)

# --------------------
# TEST DB (AUTO SETUP)
# --------------------
db-test-create:
	@mysql -u $(DB_USER) -p$(DB_PASS) -e "CREATE DATABASE IF NOT EXISTS $(TEST_DB);"

db-test-migrate:
	mysql -u $(DB_USER) -p$(DB_PASS) $(TEST_DB) < $(MIGRATION)

db-test-reset:
	@mysql -u $(DB_USER) -p$(DB_PASS) -e "DROP DATABASE IF EXISTS $(TEST_DB);"

db-test-setup: db-test-reset db-test-create db-test-migrate

# --------------------
# FULL TEST PIPELINE
# --------------------
test: db-test-setup
	go test ./... -v