# Notification Service

Simple service storing notifications delivered via HTTP or RabbitMQ.

## Running

```
docker-compose up --build
```

Service exposes:

- `POST /notifications` to create a notification. Requires `Idempotency-Key` header and `Content-Type: application/json`. Request body is limited to 1MB.
- `GET /notifications` to list notifications with filtering by sender or recipient and cursor pagination via the `after` query parameter. `limit` is clamped to 1..100 with default 50. Response includes a `next` cursor only when more results are available.
- `GET /healthz` liveness probe.
- `GET /readyz` checks database and RabbitMQ connectivity.
- `GET /metrics` Prometheus metrics.

RabbitMQ consumer listens on queue `notifications` with exponential backoff retries and dead-letter queue. Metrics and health are exposed on port `8081` (`/metrics`, `/healthz`).

Migrations are executed separately via `./migrate` container before the application starts.
