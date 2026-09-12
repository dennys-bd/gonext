.PHONY: golden test openapi install

golden: ## Regenerate golden/ by running `gonext init` against templates/, backing up any existing golden/ first
	go run ./cmd/golden

openapi: ## Regenerate templates/docs/openapi.yaml from golden/ with the working-tree CLI
	go build -o /tmp/gonext-openapi ./cmd/scaffold && cd golden && /tmp/gonext-openapi openapi && cp docs/openapi.yaml ../templates/docs/openapi.yaml

install: ## Build the working-tree CLI into ~/go/bin/gonext
	go build -o ~/go/bin/gonext ./cmd/scaffold

test: ## Build, vet and race-test every package that builds standalone (templates/ never does)
	go build ./auth/... ./cmd/... ./dbmigrate/... ./internal/... . && go vet ./auth/... ./cmd/... ./dbmigrate/... ./internal/... . && go test -race ./auth/... ./cmd/... ./dbmigrate/... ./internal/... .
