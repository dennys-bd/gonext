.PHONY: golden test

golden: ## Regenerate golden/ by running `gonext init` against templates/, backing up any existing golden/ first
	go run ./cmd/golden

test: ## Run this repo's own Go tests (cmd/, internal/), including the golden-snapshot drift test
	go test -race ./cmd/... ./internal/... .

snapshot:
	go test ./cmd/scaffold/... -run TestCopy_GoldenSnapshot -v