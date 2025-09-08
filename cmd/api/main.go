package main

import (
        "context"
        "log"
        "net/http"
        "os"
        "strconv"

        "github.com/gin-gonic/gin"
        "notificationservice/internal/db"
        "notificationservice/internal/model"
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
	go consumer.Start(ctx)

        router := gin.New()
        router.Use(gin.Recovery())

        router.POST("/notifications", func(c *gin.Context) {
                if c.ContentType() != "application/json" {
                        c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": "content type must be application/json"})
                        return
                }
                c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
                var in model.NotificationInput
                if err := c.BindJSON(&in); err != nil {
                        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
                        return
                }
                if err := in.Validate(); err != nil {
                        c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
                        return
                }
                key := c.GetHeader("Idempotency-Key")
                if key == "" {
                        c.JSON(http.StatusBadRequest, gin.H{"error": "Idempotency-Key header required"})
                        return
                }
                if err := store.CreateNotification(c.Request.Context(), key, in); err != nil {
                        if err == db.ErrDuplicate {
                                c.JSON(http.StatusConflict, gin.H{"error": "duplicate"})
                                return
                        }
                        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store"})
                        return
                }
                log.Printf("stored notification from %s to %s", model.MaskEmail(in.Sender), model.MaskEmail(in.Recipient))
                c.Status(http.StatusCreated)
        })

        router.GET("/notifications", func(c *gin.Context) {
                var f db.ListFilter
                if s := c.Query("sender"); s != "" {
                        f.Sender = &s
                }
                if r := c.Query("recipient"); r != "" {
                        f.Recipient = &r
                }
                if l := c.Query("limit"); l != "" {
                        if v, err := strconv.Atoi(l); err == nil {
                                f.Limit = v
                        }
                }
                if after := c.Query("after"); after != "" {
                        if cur, err := store.ParseCursor(after); err == nil {
                                f.After = cur
                        }
                }
                list, next, err := store.ListNotifications(c.Request.Context(), f)
                if err != nil {
                        c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
                        return
                }
                c.JSON(http.StatusOK, gin.H{"data": list, "next": next})
        })

        router.GET("/healthz", func(c *gin.Context) {
                c.JSON(http.StatusOK, gin.H{"status": "ok"})
        })

        router.GET("/readyz", func(c *gin.Context) {
                if err := store.Ping(c.Request.Context()); err != nil {
                        c.JSON(http.StatusServiceUnavailable, gin.H{"status": "db error"})
                        return
                }
                if err := consumer.Ping(); err != nil {
                        c.JSON(http.StatusServiceUnavailable, gin.H{"status": "queue error"})
                        return
                }
                c.JSON(http.StatusOK, gin.H{"status": "ok"})
        })

        addr := getenv("ADDR", ":8080")
        log.Printf("listening on %s", addr)
        if err := router.Run(addr); err != nil && err != http.ErrServerClosed {
                log.Fatalf("server error: %v", err)
        }
}
