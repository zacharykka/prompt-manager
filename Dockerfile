# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS builder
WORKDIR /app

ENV CGO_ENABLED=1 GOOS=linux

RUN apk add --no-cache build-base

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o ./bin/prompt-manager ./cmd/server

FROM alpine:3.19 AS runtime

RUN addgroup -S prompt && adduser -S prompt -G prompt

WORKDIR /app

COPY --from=builder /app/bin/prompt-manager /usr/local/bin/prompt-manager
COPY config ./config
COPY db/migrations ./db/migrations

ENV PROMPT_MANAGER_ENV=production \
    PROMPT_MANAGER_CONFIG_DIR=/app/config

USER prompt

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/prompt-manager"]
CMD ["--config-dir=/app/config", "--env=production"]
