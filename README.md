# Notification Service

Simple service storing notifications delivered via HTTP or RabbitMQ.

## Running

```
docker-compose up --build
```

Service exposes:

- `POST /notifications` to create a notification. Requires `Idempotency-Key` header and `Content-Type: application/json`. Request body is limited to 1MB.
- `GET /notifications` to list notifications with filtering by sender or recipient and cursor pagination via the `after` query parameter. Response includes a `next` cursor when more results are available.
- `GET /healthz` liveness probe.
- `GET /readyz` checks database and RabbitMQ connectivity.

RabbitMQ consumer listens on queue `notifications` with retry/backoff and dead-letter queue.
