# Dockerfile — локальный dev-образ last-mile-task
#
# TODO(LASTMILE-079): весь deploy-контур предназначен ТОЛЬКО для локального
# dev. Для production использовать базовые образы harbor и Helm-чарты.

# --- build stage ---
FROM golang:1.26 AS builder

WORKDIR /src

# Кэширование модулей
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o lmt .

# --- runtime stage ---
FROM alpine:3.22

WORKDIR /app
RUN apk add --no-cache ca-certificates

COPY --from=builder /src/lmt .

ENTRYPOINT ["./lmt"]