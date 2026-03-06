.PHONY: dev test migrate

test:
	go test -v ./...

dev:
	go run .

migrate:
	MIGRATIONS_DIR=`pwd`/db/pg/migrations go run scripts/migrate.go
