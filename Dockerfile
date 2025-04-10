FROM golang:1.21-alpine AS builder

WORKDIR /app

# Копируем модули
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY *.go ./

# Компилируем приложение
RUN CGO_ENABLED=0 GOOS=linux go build -o order-service .

# Финальный образ
FROM alpine:latest

WORKDIR /app

# Устанавливаем зависимости
RUN apk --no-cache add ca-certificates tzdata

# Копируем исполняемый файл из builder образа
COPY --from=builder /app/order-service .

# Запуск приложения
CMD ["./order-service"]