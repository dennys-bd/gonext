.PHONY: run build test vet tidy generate generate-check smoke lint vulncheck db-up db-down migrate-up frontend-dev frontend-build frontend-test frontend-lint frontend-typecheck frontend-audit secrets-scan hooks-install

ENV ?= dev
PORT ?= 8080
DATABASE_URL ?= postgres://app:app@localhost:5432/app?sslmode=disable
TEST_DATABASE_URL ?= postgres://app:app@localhost:5432/app_test?sslmode=disable

run: ## Run the backend server on $PORT (default 8080), rebuilding and restarting on change
	ENV=$(ENV) DATABASE_URL=$(DATABASE_URL) gonext dev

build: ## Build all backend packages
	mise exec -- go build ./...

test: ## Bring up Postgres, migrate the test database, run all backend tests, then tear down
	@docker compose up -d db; \
	until docker compose exec -T db pg_isready -U app -d app >/dev/null 2>&1; do sleep 1; done; \
	trap 'docker compose down' EXIT; \
	DATABASE_URL=$(TEST_DATABASE_URL) gonext migrate && \
	TEST_DATABASE_URL=$(TEST_DATABASE_URL) mise exec -- go test -race ./...

vet: ## Run go vet on the backend
	mise exec -- go vet ./...

generate: ## Regenerate wire_gen.go from the wire.go injector declarations
	mise exec -- wire ./backend/...

generate-check: ## Fail if wire_gen.go is stale relative to wire.go (fix with `make generate`)
	mise exec -- wire diff ./backend/...

lint: ## Run golangci-lint on the backend
	mise exec -- golangci-lint run ./...

vulncheck: ## Run govulncheck on the backend
	mise exec -- go tool govulncheck ./...

secrets-scan: ## Scan the full repo (working tree + history) for leaked secrets
	mise exec -- gitleaks detect --redact -v

tidy: ## Tidy go.mod and go.sum
	go mod tidy

db-up: ## Start local Postgres via docker compose
	docker compose up -d

db-down: ## Stop local Postgres
	docker compose down

migrate-up: ## Apply pending database migrations against DATABASE_URL
	gonext migrate

smoke: ## Migrate a local Postgres, build/run the server, run the Bruno collection against it, then stop everything
	@docker compose up -d db; \
	until docker compose exec -T db pg_isready -U app -d app >/dev/null 2>&1; do sleep 1; done; \
	DATABASE_URL=$(DATABASE_URL) gonext migrate && \
	lsof -ti tcp:$(PORT) | xargs -r kill 2>/dev/null; \
	go build -o /tmp/backend-smoke-server ./backend || exit 1; \
	ENV=$(ENV) PORT=$(PORT) DATABASE_URL=$(DATABASE_URL) /tmp/backend-smoke-server & \
	SERVER_PID=$$!; \
	trap 'kill $$SERVER_PID 2>/dev/null; rm -f /tmp/backend-smoke-server; docker compose down' EXIT; \
	sleep 1; \
	(cd docs/bruno && mise exec -- bru run . --env Local -r)

frontend-dev: ## Run the frontend dev server
	mise exec -- pnpm --dir frontend dev

frontend-build: ## Build the frontend for production
	mise exec -- pnpm --dir frontend build

frontend-test: ## Run frontend tests
	mise exec -- pnpm --dir frontend test

frontend-lint: ## Lint the frontend with Biome
	mise exec -- pnpm --dir frontend lint

frontend-typecheck: ## Type-check the frontend with tsc
	mise exec -- pnpm --dir frontend typecheck

frontend-audit: ## Audit frontend production dependencies for vulnerabilities
	mise exec -- pnpm --dir frontend audit --prod

hooks-install: ## Install the lefthook git hooks for this repo
	mise exec -- lefthook install
