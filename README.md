# SMS Gateway

A REST API for a simple SMS Gateway, built for the ArvanCloud software
developer challenge. Customers hold a prepaid balance and send SMS
messages (normal, OTP, or Express) to any number; the gateway guarantees
a customer is never billed beyond their balance and can retrieve reports
of everything they've sent.

Written in Go, standard library only (see [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md#6-why-zero-third-party-dependencies)
for why). No authentication/user-management system, per the brief — a
"customer" is just a billable account created via the API.

**Start here for the design rationale, the concurrency-safety argument,
and the production-scale plan:** [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md).

## Features

- **Customers** with a prepaid, integer (Rial) balance — top up any time.
- **Send SMS** as `normal`, `otp` (dynamic password), or `express`
  (guaranteed-delivery-time-to-operator) — each priced and routed
  differently.
- **Balance is never overspent**, even under heavy concurrent load —
  atomic check-and-debit, proven by a `-race`-clean concurrency test
  (see [Testing](#testing)).
- **Reports**: list a customer's sent messages, filterable by type/status,
  paginated.
- Async delivery pipeline with a dedicated worker pool for Express
  traffic, so normal-traffic bursts can't jeopardize its SLA.
- Structured logging, panic recovery, graceful shutdown, health check.

## Quick start

### Locally

Requires Go 1.22+.

```bash
go run ./cmd/api
# or: make run
```

The server listens on `:8080` by default (configurable — see
[Configuration](#configuration)).

### With Docker

```bash
docker compose up --build
# or: make docker-up
```

This builds the image from the included multi-stage `Dockerfile` and
runs it with the default configuration, exposed on `localhost:8080`.

## API reference

All requests/responses are JSON. Money amounts are integers (Rials).

### Create a customer

```bash
curl -s -X POST http://localhost:8080/api/v1/customers \
  -d '{"name":"Acme Inc"}'
```
```json
{"id":"6b1f...","name":"Acme Inc","balance":0,"created_at":"2026-07-21T07:10:01Z"}
```

### Top up balance

```bash
curl -s -X POST http://localhost:8080/api/v1/customers/{id}/balance \
  -d '{"amount":10000}'
```
```json
{"customer_id":"6b1f...","balance":10000}
```

### Get a customer (check balance)

```bash
curl -s http://localhost:8080/api/v1/customers/{id}
```

### Send an SMS

`type` is one of `normal` (default), `otp`, `express`.

```bash
curl -s -X POST http://localhost:8080/api/v1/sms \
  -d '{
    "customer_id": "6b1f...",
    "sender": "10001234",
    "receiver": "0912xxxxxxx",
    "message": "your code is 4821",
    "type": "otp"
  }'
```
```json
{
  "id": "9f2a...",
  "customer_id": "6b1f...",
  "sender": "10001234",
  "receiver": "0912xxxxxxx",
  "body": "your code is 4821",
  "type": "otp",
  "price": 150,
  "status": "queued",
  "created_at": "2026-07-21T07:10:37Z"
}
```

Returns `402 Payment Required` if the customer's balance can't cover the
message's price — no message is ever queued in that case.

### Get a single message

```bash
curl -s http://localhost:8080/api/v1/sms/{id}
```

### List sent-message reports

```bash
curl -s "http://localhost:8080/api/v1/sms?customer_id=6b1f...&type=express&status=sent&limit=20&offset=0"
```
```json
{"messages": [ ... ], "total": 42, "limit": 20, "offset": 0}
```

### Health check

```bash
curl -s http://localhost:8080/healthz
```

## Configuration

All configuration is via environment variables (see
`internal/config/config.go`); every value has a sane default.

| Variable | Default | Meaning |
|---|---|---|
| `PORT` | `8080` | HTTP listen port |
| `READ_TIMEOUT` | `5s` | HTTP server read timeout |
| `WRITE_TIMEOUT` | `10s` | HTTP server write timeout |
| `SHUTDOWN_TIMEOUT` | `15s` | Grace period for in-flight requests on shutdown |
| `NORMAL_WORKERS` | `20` | Worker pool size for normal/OTP delivery |
| `EXPRESS_WORKERS` | `10` | Worker pool size for Express delivery |
| `QUEUE_SIZE` | `10000` | Buffered channel size per delivery queue |
| `PRICE_NORMAL` | `100` | Rials per normal SMS |
| `PRICE_OTP` | `150` | Rials per OTP SMS |
| `PRICE_EXPRESS` | `300` | Rials per Express SMS |

## Project structure

```
cmd/api/                        entrypoint — wires everything together, starts the HTTP server
internal/
  domain/                       entities (Customer, Message), repository interfaces, sentinel errors
  service/                      use cases: SMSService, OperatorRouter, Dispatcher (async delivery)
  repository/memory/            in-memory, concurrency-safe implementations of the domain repositories
  transport/http/               HTTP handlers, routing, request/response mapping
  transport/http/middleware/    logging, panic recovery, JSON content-type
  config/                       environment-based configuration
  idgen/                        dependency-free UUID v4 generator
docs/
  ARCHITECTURE.md               system design, the concurrency-safety argument, production-scale plan
Dockerfile                      multi-stage build → minimal Alpine runtime image
docker-compose.yml               single-command local run
Makefile                        make run / build / test / test-race / docker-*
```

## Testing

```bash
make test-race
# or: go test ./... -race -cover
```

The test that matters most for this challenge is
`TestSendMessage_ConcurrentSendsNeverOversell` (service layer) and its
repository-level counterpart `TestDebitBalance_NeverOversells`: they
fire hundreds of concurrent send/debit requests against a balance that
can only cover a fraction of them, run under the race detector, and
assert both that the balance never goes negative and that the number of
successful sends is exactly what the starting balance could afford —
directly verifying the brief's "no message may be sent after balance is
exhausted" requirement, not just asserting it by design.

HTTP-layer tests (`internal/transport/http/handler_test.go`) exercise
the real router and middleware stack end-to-end.

## Design notes and trade-offs

See [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) for the full writeup, including:
- why check-and-debit is safe under concurrency (§3)
- how routing handles uneven per-operator traffic and the Express SLA (§5)
- why this submission has no third-party dependencies (§6)
- the concrete migration path to PostgreSQL/Redis/Kafka and horizontal
  scaling for real production traffic (~100M messages/day) (§7)
