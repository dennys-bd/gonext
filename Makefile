.PHONY: golden

golden: ## Regenerate golden/ by running `gonext init` against templates/, backing up any existing golden/ first
	go run ./cmd/golden
