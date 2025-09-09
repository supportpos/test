package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus/promhttp"
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
	consumer, err := queue.NewConsumer(getenv("RABBITMQ_URL", "amqp://guest:guest@rabbitmq:5672/"), store)
	if err != nil {
		log.Fatalf("rabbit connect: %v", err)
	}

	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
		addr := getenv("METRICS_ADDR", ":8081")
		log.Printf("metrics listening on %s", addr)
		if err := http.ListenAndServe(addr, mux); err != nil && err != http.ErrServerClosed {
			log.Fatalf("metrics server: %v", err)
		}
	}()

	consumer.Start(ctx)
}
