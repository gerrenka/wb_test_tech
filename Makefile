.PHONY: build clean test run docker-build docker-run fmt imports lint lint-all kafka-up kafka-down postgres-up postgres-down db-init all-services-up all-services-down generate-orders build-generator generate-mocks help

# Переменные проекта
APP_NAME=order-service
GO_FILES=$(shell find . -name "*.go" -type f -not -path "./mocks/*" -not -path "./cmd/generator/*")
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

# Docker и docker-compose переменные
DOCKER_COMPOSE_FILE=docker-compose.yml
DOCKER_APP_IMAGE=$(APP_NAME):$(VERSION)

# Значение по умолчанию
.DEFAULT_GOAL := help

# Сборка приложения
build:
	@echo "Сборка приложения..."
	@go build $(LDFLAGS) -o $(APP_NAME) .

# Очистка артефактов сборки
clean:
	@echo "Очистка артефактов сборки..."
	@rm -f $(APP_NAME) order-generator
	@go clean
	@echo "Артефакты сборки очищены"

# Запуск тестов
test:
	@echo "Запуск тестов..."
	@go test -v ./...

# Локальный запуск приложения
run: build
	@echo "Запуск приложения..."
	@./$(APP_NAME)

# Форматирование кода
fmt:
	@echo "Форматирование Go-кода..."
	@gofmt -s -w $(GO_FILES)

# Упорядочивание импортов
imports:
	@echo "Упорядочивание импортов..."
	@go install golang.org/x/tools/cmd/goimports@latest
	@$(shell go env GOPATH)/bin/goimports -w $(GO_FILES)

# Запуск линтера
lint:
	@echo "Запуск линтера..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@$(shell go env GOPATH)/bin/golangci-lint run ./...

# Запуск линтера и форматирования
lint-all: lint imports fmt
	@echo "Все проверки кода выполнены"

# Сборка Docker-образа
docker-build:
	@echo "Сборка Docker-образа..."
	@docker build -t $(DOCKER_APP_IMAGE) .

# Запуск приложения в Docker
docker-run: docker-build
	@echo "Запуск Docker-контейнера..."
	@docker run --rm --network=kafka_network -p 8081:8081 $(DOCKER_APP_IMAGE)

# Запуск Kafka в Docker
kafka-up:
	@echo "Запуск Kafka..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) up -d zookeeper kafka
	@echo "Подождите, пока Kafka запустится..."
	@sleep 10
	@echo "Создание топика orders..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) exec kafka kafka-topics --create --if-not-exists --topic orders --bootstrap-server localhost:9092 --replication-factor 1 --partitions 1

# Остановка Kafka
kafka-down:
	@echo "Остановка Kafka..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) stop kafka zookeeper
	@docker-compose -f $(DOCKER_COMPOSE_FILE) rm -f kafka zookeeper

# Запуск PostgreSQL в Docker
postgres-up:
	@echo "Запуск PostgreSQL..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) up -d postgres

# Остановка PostgreSQL
postgres-down:
	@echo "Остановка PostgreSQL..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) stop postgres
	@docker-compose -f $(DOCKER_COMPOSE_FILE) rm -f postgres

# Инициализация базы данных
db-init: postgres-up
	@echo "Ожидание запуска PostgreSQL..."
	@sleep 5
	@echo "Инициализация схемы базы данных..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) exec -T postgres psql -U my_user -d my_database -f /docker-entrypoint-initdb.d/create_tables.sql

# Запуск всех сервисов
all-services-up: kafka-up postgres-up
	@echo "Все сервисы запущены"
	@echo "PostgreSQL: localhost:5434"
	@echo "Kafka: localhost:9093"

# Остановка всех сервисов
all-services-down: kafka-down postgres-down
	@echo "Все сервисы остановлены"

# Генерация тестовых заказов
generate-orders: build-generator
	@echo "Генерация тестовых заказов..."
	@./order-generator --count 5 --interval 500

# Сборка генератора заказов
build-generator:
	@echo "Сборка генератора заказов..."
	@go build -o order-generator ./cmd/generator/order_generator.go

# Генерация моков для интерфейсов
generate-mocks:
	@echo "Генерация моков для интерфейсов..."
	@mkdir -p mocks
	@echo "Внимание: моки созданы вручную из-за структуры проекта и находятся в директории mocks/"
	@echo "Для автоматической генерации моков в будущих проектах рекомендуется использовать структуру с пакетами."
	@echo "Подробнее в файле MockGen.md"

# Вывод справки
help:
	@echo "Доступные команды:"
	@echo "  make build             - сборка приложения"
	@echo "  make clean             - очистка артефактов"
	@echo "  make test              - запуск тестов"
	@echo "  make run               - сборка и запуск приложения"
	@echo "  make fmt               - форматирование Go-кода"
	@echo "  make imports           - упорядочивание импортов Go-кода"
	@echo "  make lint              - линтинг кода"
	@echo "  make lint-all          - запуск всех проверок кода (lint, imports, fmt)"
	@echo "  make docker-build      - сборка Docker-образа"
	@echo "  make docker-run        - запуск в Docker"
	@echo "  make kafka-up          - запуск Kafka в Docker"
	@echo "  make kafka-down        - остановка Kafka"
	@echo "  make postgres-up       - запуск PostgreSQL в Docker"
	@echo "  make postgres-down     - остановка PostgreSQL"
	@echo "  make db-init           - инициализация базы данных"
	@echo "  make all-services-up   - запуск всех сервисов"
	@echo "  make all-services-down - остановка всех сервисов"
	@echo "  make generate-orders   - генерация и отправка тестовых заказов в Kafka"
	@echo "  make build-generator   - сборка генератора заказов"
	@echo "  make generate-mocks    - генерация моков для интерфейсов"