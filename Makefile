.PHONY: golden test openapi

golden: ## Regenerate golden/ by running `gonext init` against templates/, backing up any existing golden/ first
	go run ./cmd/golden

openapi: ## Regenerate templates/docs/openapi.yaml from golden/ with the working-tree CLI
	go build -o /tmp/gonext-openapi ./cmd/scaffold && cd golden && /tmp/gonext-openapi openapi && cp docs/openapi.yaml ../templates/docs/openapi.yaml
