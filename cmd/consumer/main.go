package main

import (
        "context"
        "log"
        "os"

        "notificationservice/internal/db"
        "notificationservice/internal/queue"
)

func getenv(key, def string) string {
        if v := os.Getenv(key); v != "" {
                return v
        }
        return def
}

func main() {
        ctx := context.Background()
        store, err := db.New(ctx, getenv("DATABASE_URL", "postgres://postgres:postgres@db:5432/postgres?sslmode=disable"))
        if err != nil {
                log.Fatalf("db connect: %v", err)
        }
        defer store.Close()
        if err := store.Migrate(ctx); err != nil {
                log.Fatalf("db migrate: %v", err)
        }

        consumer, err := queue.NewConsumer(getenv("RABBITMQ_URL", "amqp://guest:guest@rabbitmq:5672/"), store)
        if err != nil {
                log.Fatalf("rabbit connect: %v", err)
        }
        consumer.Start(ctx)
}

