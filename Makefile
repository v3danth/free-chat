run:
	go run ./cmd/server

db-reset:
	mysql -u root -p"passwd" chat_db < scripts/reset_db.sql

migrate:
	mysql -u root -p"passwd" chat_db < migrations/001_init.sql

test:
	go test ./...

lint:
	golangci-lint run