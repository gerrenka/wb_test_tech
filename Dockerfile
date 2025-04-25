FROM go-base:latest AS builder

# Создаем рабочую директорию
WORKDIR /build

# Копируем только go.mod и go.sum для лучшего кэширования
COPY go.mod go.sum ./

# Явно добавляем зависимости в модуль
RUN go get github.com/google/uuid && \
    go get github.com/segmentio/kafka-go && \
    go mod tidy

# Теперь копируем весь проект
COPY . .

# Компилируем основной сервис
RUN go build -o /order-service ./cmd/order_service

# Финальный образ
FROM alpine:latest
WORKDIR /app

# Устанавливаем необходимые пакеты
RUN apk --no-cache add ca-certificates tzdata curl

# Копируем бинарный файл и шаблоны
COPY --from=builder /order-service .
COPY templates/ ./templates/

# Создаем директорию для логов
RUN mkdir -p /app/logs && chmod 777 /app/logs

# Настройка переменных окружения
ENV DB_HOST=postgres \
    DB_PORT=5432 \
    DB_USER=my_user \
    DB_PASSWORD=1 \
    DB_NAME=my_database \
    DB_SSL_MODE=disable \
    KAFKA_BROKERS=kafka:9092 \
    KAFKA_TOPIC=orders \
    KAFKA_GROUP_ID=order-consumer-group \
    SERVER_PORT=8081 \
    CACHE_TTL=30m

# Порт, который будет прослушивать приложение
EXPOSE 8081

# Запуск приложения
CMD ["./order-service"]