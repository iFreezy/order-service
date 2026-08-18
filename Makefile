-include .env

OUTPUT:=./bin/app
GO_LINT_VERSION=1.64.8

GO_FILE:=./main.go
MIGRATION_DIR := ./migration/postgres
MIGRATION_DSN := postgres://$(APP_REPOSITORY_POSTGRES_USERNAME):$(APP_REPOSITORY_POSTGRES_PASSWORD)@$(APP_REPOSITORY_POSTGRES_ADDRESS)/$(APP_REPOSITORY_POSTGRES_NAME)?sslmode=disable

.PHONY: up
up: ## Поднимает (запускает) окружение для работы приложения
	docker compose up -d

.PHONY: down
down: ## Отключает окружение для работы приложения
	docker compose down --remove-orphans

.PHONY: lint
lint: ## Запуск линтера
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@v${GO_LINT_VERSION} run

.PHONY: lint-fix
lint-fix: ## Запуск линтера с фиксом
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@v${GO_LINT_VERSION} run --fix

.PHONY: build
build: ## Сборка приложения
	go build -o ${OUTPUT} ${GO_FILE}

.PHONY: test
test: ## Запуск тестов
	go test -count=1 -v ./...

.PHONY: migrate-up
migrate-up: ## Применить все миграции
	migrate -database "$(MIGRATION_DSN)" -path $(MIGRATION_DIR) up

.PHONY: migrate-down
migrate-down: ## Откатить все миграции
	migrate -database "$(MIGRATION_DSN)" -path $(MIGRATION_DIR) down -all

.PHONY: migrate-create
migrate-create: ## Создать миграцию (NAME=имя)
	migrate create -ext sql -dir $(MIGRATION_DIR) -seq $(NAME)
