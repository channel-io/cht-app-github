# Project directory structure
PROJECT_PATH := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

# Artifacts
TARGET_DIR = target

# Go environment
GOVERSION := $(shell go version | awk '{print $$3}')
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

#
# User commands
#
DC=docker compose -f development/docker-compose.yml

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development
.PHONY: env
env: ## Print current development environment
	@echo "PROJECT_PATH:\t${PROJECT_PATH}"
	@echo "GOVERSION:\t${GOVERSION}"
	@echo "GOOS:\t\t${GOOS}"
	@echo "GOARCH:\t\t${GOARCH}"

.PHONY: dev
dev: setup ## Run development server
	@go run ./cmd

.PHONY: setup
setup: deps ## Setup development dependencies in background

.PHONY: generate
generate: _generate

##@ Build
.PHONY: build
build: deps _build ## Build project

.PHONY: deps
deps: ## Sync project dependencies
	@go mod download
	@go mod tidy

.PHONY: lint
lint: ## Run linter and formatter
	@echo Running lint...
	@if command -v golangci-lint >/dev/null 2>&1; then \
		make _lint; \
	else \
		$(DC) run --rm lint; \
	fi
	@echo Done

.PHONY: test
test: ## Run tests
	make _test

##@ Cleanup
.PHONY: down
down: ## Clean up local development dependencies
	@$(DC) down

.PHONY: clean
clean: ## Clean up all local state
	@rm -rf ${TARGET_DIR}
	@$(DC) down -v

#
# Internal commands for ci
#
.PHONY: _generate
_generate: _generate-docs

.PHONY: _generate-docs
_generate-docs:
	@echo Generating docs...
	@swag init -d cmd,api/public/http/route -o api/public/http/docs
	@swag fmt

.PHONY: _build
_build:
	@go build -o ${TARGET_DIR}/app ./cmd

.PHONY: _test
_test:
	@go test -count=1 ./...

.PHONY: _lint
_lint:
	@golangci-lint run
