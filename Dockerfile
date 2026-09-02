# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-w -s" -o /out/api ./cmd/api

FROM builder AS goose-builder
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go install github.com/pressly/goose/v3/cmd/goose@v3.22.1

FROM alpine:3.21 AS migrate
COPY --from=goose-builder /go/bin/goose /usr/local/bin/goose
COPY migrations /migrations
WORKDIR /migrations
ENV GOOSE_DRIVER=postgres GOOSE_MIGRATION_DIR=/migrations
ENTRYPOINT ["goose"]
CMD ["up"]

FROM alpine:3.21 AS api
RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 10001 app
COPY --from=builder /out/api /usr/local/bin/api
USER app
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/health || exit 1
ENTRYPOINT ["/usr/local/bin/api"]
