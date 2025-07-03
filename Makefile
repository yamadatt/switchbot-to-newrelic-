# Makefile for SwitchBot to New Relic Lambda Function

.PHONY: help build test test-unit test-integration test-local clean setup-test-env cleanup-test-env fmt lint vet deps

# デフォルトターゲット
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# ビルド関連
build: ## Build the SAM application
	sam build

build-mock: ## Build the mock server
	cd test && go build -o mock-server mock-server.go

# テスト関連
test: test-unit test-integration ## Run all tests

test-unit: ## Run unit tests
	go test -v -race -coverprofile=coverage.out ./...

test-local: setup-test-env build ## Run local test with sam local invoke
	./scripts/run-local-test.sh

test-integration: build-mock build ## Run integration test with mock server
	./scripts/run-integration-test.sh

# 開発環境セットアップ
setup-test-env: ## Setup test SSM parameters
	./scripts/setup-test-ssm.sh

cleanup-test-env: ## Cleanup test SSM parameters
	./scripts/cleanup-test-ssm.sh

# コード品質
fmt: ## Format Go code
	go fmt ./...

lint: ## Run golangci-lint
	golangci-lint run

vet: ## Run go vet
	go vet ./...

# 依存関係管理
deps: ## Download and verify dependencies
	go mod download
	go mod verify

deps-update: ## Update dependencies
	go get -u ./...
	go mod tidy

# クリーンアップ
clean: ## Clean build artifacts and test files
	rm -rf .aws-sam/
	rm -f test/mock-server
	rm -f test/integration-test-env.json
	rm -f coverage.out
	rm -f bootstrap

# Docker関連
docker-test-up: ## Start test environment with Docker Compose
	cd test && docker-compose -f docker-compose.test.yml up -d

docker-test-down: ## Stop test environment
	cd test && docker-compose -f docker-compose.test.yml down

docker-test-logs: ## Show Docker test environment logs
	cd test && docker-compose -f docker-compose.test.yml logs -f

# SAM関連
sam-deploy-guided: build ## Deploy with guided prompts
	sam deploy --guided

sam-deploy: build ## Deploy using existing configuration
	sam deploy

sam-logs: ## Tail Lambda function logs
	sam logs -n SwitchBotToNewRelicFunction --stack-name switchbot-to-newrelic-app --tail

sam-local-api: build ## Start local API Gateway
	sam local start-api --env-vars test/local-test-env.json

# 開発用ワークフロー
dev-setup: deps setup-test-env ## Setup development environment
	@echo "Development environment setup complete!"
	@echo "Run 'make test-local' to test the function locally"

dev-test: fmt vet test-unit test-integration ## Run full development test suite

# CI/CD用
ci-test: deps fmt vet test-unit ## Run CI test suite (without integration tests)

# セキュリティ
security-scan: ## Run security scans
	gosec ./...
	govulncheck ./...

# カバレッジ
coverage: test-unit ## Generate and open coverage report
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# 本番デプロイ前チェック
pre-deploy: clean deps fmt vet test security-scan build ## Run all checks before deployment
	@echo "Pre-deployment checks completed successfully!"
	@echo "Ready to deploy with: make sam-deploy"