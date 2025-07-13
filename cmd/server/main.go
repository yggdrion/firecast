package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"firecast/pkg/handler"
	"firecast/pkg/wiprecovery"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

var (
	ctx = context.Background()
	rdb *redis.Client
)

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	}

	fireCastSecret := os.Getenv("FIRECAST_SECRET")
	azuraCastApiKey := os.Getenv("AZURACAST_API_KEY")
	azuraCastDomain := os.Getenv("AZURACAST_DOMAIN")

	if fireCastSecret == "" || azuraCastApiKey == "" || azuraCastDomain == "" {
		fmt.Println("Environment variables FIRECAST_SECRET, AZURACAST_API_KEY, and AZURACAST_DOMAIN must be set")
		return
	}

	rdb = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Redis connection failed: %v", err)
	}

	h := handler.NewHandler(rdb, fireCastSecret)

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Public routes (no authentication required)
	r.Get("/health", h.HealthzHandler)
	r.Get("/healthz", h.HealthzHandler)

	// Protected routes (authentication required)
	r.Group(func(r chi.Router) {
		r.Use(h.AuthMiddleware)
		r.Post("/video/add", h.VideoAddHandler)
		r.Get("/video/get", h.VideoGetHandler)
		r.Post("/video/done", h.VideoDoneHandler)
		r.Post("/video/fail", h.VideoFailHandler)
		r.Get("/status", h.StatusHandler)
	})

	wiprecovery.WipRecovery(ctx, rdb)

	fmt.Println("Server starting on :8080")
	http.ListenAndServe(":8080", r)
}
