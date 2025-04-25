# DB path and connection
DB_CONN=postgres://my_user:1@postgres:5432/my_database?sslmode=disable

# Docker management
docker-build:
	# Build base image first
	docker build -t go-base:latest -f Dockerfile.base .
	# Then build the services that use it
	docker-compose build order-service order-generator

docker-up:
	docker-compose up -d postgres zookeeper kafka 
	sleep 10
	docker-compose up -d order-service order-generator

docker-up-with-logs:
	docker-compose up

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f

docker-status:
	docker-compose ps

docker-restart-service:
	docker-compose restart order-service

docker-restart-generator:
	docker-compose restart order-generator

# Tests
test:
	go test ./... -v

# Migrations with migrate
migrate-up:
	docker run --rm -v "$(PWD)/migrations:/migrations" \
	--network host \
	migrate/migrate:latest -path=/migrations -database "$(DB_CONN)" up

migrate-down:
	docker run --rm -v "$(PWD)/migrations:/migrations" \
	--network host \
	migrate/migrate:latest -path=/migrations -database "$(DB_CONN)" down

# Local development
run-local:
	go run cmd/order_service/main.go

generate-orders-local:
	go run cmd/generator/main.go --count=5 --interval=500 --brokers=localhost:9093 --print-only=true

# Cleanup
clean:
	docker-compose down -v
	rm -rf bin/

# Helper commands
check-kafka:
	docker-compose exec kafka kafka-topics --bootstrap-server kafka:9092 --list

check-service:
	curl http://localhost:8081/health

# Default
.PHONY: docker-build docker-up docker-down test migrate-up migrate-down run-local clean
default: docker-build docker-up