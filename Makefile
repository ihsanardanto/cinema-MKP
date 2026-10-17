DATABASE_URL ?= user=$(USER) dbname=cinema host=/var/run/postgresql sslmode=disable
MIGRATIONS := $(sort $(wildcard db/migrations/*.sql))

.PHONY: help migrate reset-db run test vet

help:
	@printf '%s\n' 'Targets:' \
	  '  make migrate  Apply pending migrations, including local demo data' \
	  '  make reset-db Drop all application tables, then reapply migrations' \
	  '  make run      Start the API on http://localhost:8080' \
	  '  make test     Run Go tests' \
	  '  make vet      Run go vet'
	@printf '%s\n' 'Override the database connection with DATABASE_URL="..." when needed.'

migrate:
	@psql "$(DATABASE_URL)" -v ON_ERROR_STOP=1 -c 'CREATE TABLE IF NOT EXISTS schema_migrations (filename text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())'
	@for file in $(MIGRATIONS); do \
		name=$$(basename "$$file"); \
		if psql "$(DATABASE_URL)" -Atqc "SELECT 1 FROM schema_migrations WHERE filename = '$$name'" | grep -qx 1; then \
			printf 'Already applied: %s\n' "$$name"; \
		else \
			printf 'Applying: %s\n' "$$name"; \
			psql "$(DATABASE_URL)" -v ON_ERROR_STOP=1 -f "$$file"; \
			psql "$(DATABASE_URL)" -v ON_ERROR_STOP=1 -c "INSERT INTO schema_migrations(filename) VALUES ('$$name')"; \
		fi; \
	done

reset-db:
	psql "$(DATABASE_URL)" -v ON_ERROR_STOP=1 -c 'DROP SCHEMA public CASCADE; CREATE SCHEMA public;'
	$(MAKE) migrate DATABASE_URL='$(DATABASE_URL)'

run:
	DATABASE_URL='$(DATABASE_URL)' go run .

test:
	go test ./...

vet:
	go vet ./...
