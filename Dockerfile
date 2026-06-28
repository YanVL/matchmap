FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o matchmap-api ./cmd/api

FROM alpine:latest

WORKDIR /app

RUN adduser -D appuser

COPY --from=builder /app/matchmap-api .

USER appuser

EXPOSE 8080

CMD ["./matchmap-api"]