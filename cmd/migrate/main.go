package main

import (
	"context"
	"log"
	"os"

	"notificationservice/internal/db"
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
		log.Fatalf("migrate: %v", err)
	}
	log.Printf("migration complete")
}
