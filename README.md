# Notification Service

Simple service storing notifications delivered via HTTP or RabbitMQ.

## Running

```
docker-compose up --build
```

Service exposes:

- `POST /notifications` to create a notification.
- `GET /notifications` to list notifications with pagination and filtering.
- `GET /health` health check used by container.

RabbitMQ consumer listens on queue `notifications` with retry/backoff and dead-letter queue.
