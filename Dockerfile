# Стейдж сборки
FROM golang:1.21-alpine AS builder

# ВАЖНО: включаем работу через Go-модули
ENV CGO_ENABLED=0 GOOS=linux GO111MODULE=on

# Устанавливаем рабочую директорию
WORKDIR /app

# Копируем только модули для кэширования зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь проект
COPY . .

# Переходим в папку с приложением
WORKDIR /app/cmd/order_service

# Сборка приложения
RUN go build -o /order-service .

# Финальный образ
FROM alpine:latest
WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /order-service .
COPY templates/ ./templates/

CMD ["./order-service"]