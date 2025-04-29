FROM golang:1.24.1-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /app/server ./cmd/rateLimiting

FROM alpine:3.18
WORKDIR /app

COPY --from=builder /app/server .
COPY --from=builder /src/cmd/rateLimiting/config.json ./config.json

EXPOSE 8080

CMD ["./server", "-config", "/app/config.json"]