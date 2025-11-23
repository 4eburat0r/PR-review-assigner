FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/server ./cmd/server

FROM alpine:3.18

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/bin/server /app/bin/server

RUN adduser -D -s /bin/sh appuser
USER appuser

WORKDIR /app

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

CMD ["/app/bin/server"]