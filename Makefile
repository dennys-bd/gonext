.PHONY: golden test openapi

golden: ## Regenerate golden/ by running `gonext init` against templates/, backing up any existing golden/ first
	go run ./cmd/golden

openapi: ## Regenerate templates/docs/openapi.yaml from golden/'s backend
	cd golden && go run ./backend --openapi > ../templates/docs/openapi.yaml

test: ## Run this repo's own Go tests (auth/, cmd/, internal/), including the golden-snapshot drift test
	go test -race ./auth/... ./cmd/... ./internal/... .

snapshot:
	go test ./cmd/scaffold/... -run TestCopy_GoldenSnapshot -v
