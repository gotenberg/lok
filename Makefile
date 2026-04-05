GO_VERSION=1.26.0
GOLANGCI_LINT_VERSION=v2.10.1
DOCKER_IMAGE=lok-testing

.PHONY: help
help: ## Show the help
	@grep -hE '^[A-Za-z0-9_ \-]*?:.*##.*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: build-test
build-test: ## Build the Docker image for testing
	docker build \
		--build-arg GO_VERSION=$(GO_VERSION) \
		--build-arg GOLANGCI_LINT_VERSION=$(GOLANGCI_LINT_VERSION) \
		-t $(DOCKER_IMAGE) .

.PHONY: fmt
fmt: ## Format Golang codebase in Docker
	docker run --rm $(DOCKER_IMAGE) sh -c 'go fix ./... && golangci-lint fmt && go mod tidy'

.PHONY: prettify
prettify: ## Format non-Golang codebase
	npx prettier --write .

.PHONY: lint
lint: ## Lint Golang codebase in Docker
	docker run --rm $(DOCKER_IMAGE) golangci-lint run

.PHONY: lint-todo
lint-todo: ## Find TODOs in Golang codebase in Docker
	docker run --rm $(DOCKER_IMAGE) golangci-lint run --no-config --enable godox

.PHONY: lint-prettier
lint-prettier: ## Lint non-Golang codebase
	npx prettier --check .

.PHONY: test-unit
test-unit: ## Run unit tests in Docker
	docker run --rm $(DOCKER_IMAGE) go test -race -count=1 -v ./pkg/lok/...

.PHONY: test-integration
test-integration: ## Run integration tests in Docker
	docker run --rm -e GODEBUG=asyncpreemptoff=1 $(DOCKER_IMAGE) go test -tags integration -count=1 -v ./...

# go install golang.org/x/tools/cmd/godoc@latest
.PHONY: godoc
godoc: ## Browse docs at http://localhost:6060
	$(info http://localhost:6060/pkg/github.com/gotenberg/lok)
	godoc -http=:6060
