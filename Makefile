# Путь до миграций и базы
DB_CONN=postgres://my_user:1@postgres:5432/my_database?sslmode=disable


# Основной сервис
run:
	go run cmd/app/main.go

build:
	go build -o bin/order-service ./cmd/app

# Генератор заказов
generate-orders:
	go run cmd/generator/main.go --count=5 --interval=500

# Тесты
test:
	go test ./... -v

# Импорты
goimports:
	go install golang.org/x/tools/cmd/goimports@latest
	goimports -w .

# Генерация моков (если mockery установлен)
generate-mocks:
	mockery --all --output=mocks --keeptree

# Docker
docker-up:
	docker-compose up --build

docker-down:
	docker-compose down

# Миграции через migrate
migrate-up:
	docker run --rm -v "$(PWD)/migrations:/migrations" \
	--network wb_test_tech1_kafka_network \
	migrate/migrate:latest -path=/migrations -database "$(DB_CONN)" up


migrate-down:
	docker run --rm -v "$(PWD)/migrations:/migrations" \
	--network kafka_network \
	migrate/migrate:latest -path=/migrations -database "$(DB_CONN)" down
