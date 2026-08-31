.PHONY: golden test auth-test

golden: ## Regenerate golden/ by running `gonext init` against templates/, backing up any existing golden/ first
	go run ./cmd/golden

auth-test: ## Run the published auth contract module's tests (separate module; root patterns don't reach it)
	cd auth && go build ./... && go vet ./... && go test -race ./...

test: auth-test ## Run this repo's own Go tests (auth/, cmd/, internal/), including the golden-snapshot drift test
	go test -race ./cmd/... ./internal/... .

snapshot:
	go test ./cmd/scaffold/... -run TestCopy_GoldenSnapshot -v
