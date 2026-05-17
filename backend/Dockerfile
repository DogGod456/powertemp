FROM golang:1.22-alpine AS base
WORKDIR /src
RUN apk add --no-cache git ca-certificates

# Слой зависимостей отделен от исходников: при правке кода Docker не скачивает Go-модули заново.
COPY backend/go.mod ./
RUN go mod download

COPY backend/ ./
# go.sum намеренно может генерироваться внутри Docker-сборки.
# Это предотвращает "missing go.sum entry" после распаковки архива на новом ПК.
RUN go mod tidy

FROM base AS build-api
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api-gateway ./cmd/api-gateway

FROM base AS build-simulator
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/sensor-simulator ./cmd/sensor-simulator

FROM alpine:3.20 AS api-gateway
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build-api /out/api-gateway /app/api-gateway
EXPOSE 8080
CMD ["/app/api-gateway"]

FROM alpine:3.20 AS sensor-simulator
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build-simulator /out/sensor-simulator /app/sensor-simulator
EXPOSE 8090
CMD ["/app/sensor-simulator"]
