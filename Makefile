GC=go
BUILD_DIR=./bin

.PHONY: dev test migrate build

test:
	$(GC) test -v ./...

dev:
	$(GC) run .

build:
	$(GC) build -o $(BUILD_DIR)/backend .

migrate:
	MIGRATIONS_DIR=`pwd`/db/pg/migrations go run scripts/migrate.go
