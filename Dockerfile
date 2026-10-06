# === Этап 1: Готовый образ с protoc для генерации кода и сборки ===
FROM namely/protoc-all:1.51_2 AS compiler

# Переключаемся на рабочую директорию
WORKDIR /app

# Копируем только файл контракта для генерации
COPY proto/auth.proto ./proto/

# Генерируем Go-код (инструменты уже внутри этого образа, ссылки не нужны)
RUN mkdir -p pb && \
    protoc --go_out=. --go_opt=paths=source_relative \
           --go-grpc_out=. --go-grpc_opt=paths=source_relative \
           proto/auth.proto && \
    mv proto/auth.pb.go pb/ && \
    mv proto/auth_grpc.pb.go pb/


# === Этап 2: Родной образ Go для сборки бинарника ===
FROM golang:1.27-alpine AS builder
WORKDIR /app

# Забираем сгенерированные на прошлом шаге файлы контрактов
COPY --from=compiler /app/pb ./pb

# Копируем зависимости и остальной код
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Собираем сервис
RUN CGO_ENABLED=0 GOOS=linux go build -o auth-service ./cmd/main.go


# === Этап 3: Финальный чистый контейнер для работы ===
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/auth-service .

EXPOSE 50051
CMD ["./auth-service"]
