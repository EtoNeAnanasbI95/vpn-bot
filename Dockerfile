FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bot ./cmd/bot


FROM alpine:3.21

RUN apk add --no-cache tzdata ca-certificates

WORKDIR /app

COPY --from=builder /bot /app/bot
COPY assets/ /app/assets/

ENV DB_PATH=/data/bot.db
ENV GUIDES_DIR=/app/assets/guides

VOLUME ["/data"]

CMD ["/app/bot"]
