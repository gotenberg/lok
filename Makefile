.PHONY: help
help: ## Show the help
	@grep -hE '^[A-Za-z0-9_ \-]*?:.*##.*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: fmt
fmt: ## Format Golang codebase and "optimize" the dependencies
	go fix ./...
	golangci-lint fmt
	go mod tidy

.PHONY: prettify
prettify: ## Format non-Golang codebase
	npx prettier --write .

.PHONY: lint
lint: ## Lint Golang codebase
	golangci-lint run

.PHONY: lint-todo
lint-todo: ## Find TODOs in Golang codebase
	golangci-lint run --no-config --enable godox

.PHONY: lint-prettier
lint-prettier: ## Lint non-Golang codebase
	npx prettier --check .

# go install golang.org/x/tools/cmd/godoc@latest
.PHONY: godoc
godoc: ## Run a webserver with lok godoc
	$(info http://localhost:6060/pkg/github.com/gotenberg/lok)
	godoc -http=:6060

.PHONY: build-test
build-test: ## Build the Docker image for testing
	docker build -t lok-testing .

.PHONY: test-unit
test-unit: ## Run unit tests in Docker
	docker run --rm lok-testing test -race -count=1 -v ./pkg/lok/...

.PHONY: test-integration
test-integration: ## Run integration tests in Docker
	docker run --rm lok-testing test -tags integration -race -count=1 -v ./...
