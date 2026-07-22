# syntax=docker/dockerfile:1

# ---- build stage ----
FROM golang:1.22-alpine AS builder

WORKDIR /src

# This project has zero third-party dependencies (see go.mod), so there
# is no go.sum and no module-download step that could go stale or need
# network access at build time beyond pulling the base images.
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
    -o /out/sms-gateway ./cmd/api

# ---- runtime stage ----
FROM alpine:3.19

RUN apk add --no-cache ca-certificates curl \
    && addgroup -S app && adduser -S app -G app

WORKDIR /app
COPY --from=builder /out/sms-gateway /app/sms-gateway

USER app

ENV PORT=8080
EXPOSE 8080

HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:${PORT}/healthz || exit 1

ENTRYPOINT ["/app/sms-gateway"]
