# Every toolchain command runs through `mise exec --` so the pinned tools and
# the variables from .env apply even where mise's shell hook never fires: an
# agent's shell, a git hook run by an IDE, CI. Docker is a host prerequisite,
# not a mise tool, so `docker compose` runs bare. See AGENTS.md "Toolchains".
.PHONY: run build test vet tidy openapi openapi-check smoke lint vulncheck db-up db-down frontend-dev frontend-build frontend-test frontend-lint frontend-typecheck frontend-audit secrets-scan hooks-install

run: ## Run the backend server (config from .env), rebuilding and restarting on change
	mise exec -- gonext dev

build: ## Build all backend packages
	mise exec -- go build ./...

test: ## Bring up Postgres, migrate the test database, run all backend tests
	@docker compose up -d db; \
	until docker compose exec -T db pg_isready -U app -d app >/dev/null 2>&1; do sleep 1; done; \
	mise exec -- sh -c 'DATABASE_URL=$$TEST_DATABASE_URL gonext migrate' && \
	mise exec -- go test -race ./...

vet: ## Run go vet on the backend
	mise exec -- go vet ./...

openapi: ## Regenerate docs/openapi.yaml and the typed frontend client
	mise exec -- gonext openapi

openapi-check: ## Fail if docs/openapi.yaml is stale (fix with `make openapi`)
	mise exec -- gonext openapi --check

lint: ## Run golangci-lint on the backend
	mise exec -- golangci-lint run ./...

vulncheck: ## Run govulncheck on the backend
	mise exec -- go tool govulncheck ./...

secrets-scan: ## Scan the full repo (working tree + history) for leaked secrets
	mise exec -- gitleaks detect --redact -v

tidy: ## Tidy go.mod and go.sum
	mise exec -- go mod tidy

db-up: ## Start local Postgres via docker compose
	docker compose up -d

db-down: ## Stop local Postgres
	docker compose down

smoke: ## Migrate a local Postgres (DATABASE_URL from .env), build/run the server, run the Bruno collection against it, then stop the server
	@docker compose up -d db; \
	until docker compose exec -T db pg_isready -U app -d app >/dev/null 2>&1; do sleep 1; done; \
	mise exec -- gonext migrate && \
	mise exec -- go build -o /tmp/backend-smoke-server ./backend || exit 1; \
	mise exec -- sh -c '\
		lsof -ti tcp:$$PORT | xargs -r kill 2>/dev/null; \
		/tmp/backend-smoke-server & \
		SERVER_PID=$$!; \
		trap "kill $$SERVER_PID 2>/dev/null; rm -f /tmp/backend-smoke-server" EXIT; \
		sleep 1; \
		cd docs/bruno && bru run . --env Local -r'

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
