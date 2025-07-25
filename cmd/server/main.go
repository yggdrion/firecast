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
		fmt.Println("Error loading .env file - using environment variables")
	}

	fireCastSecret := os.Getenv("FIRECAST_SECRET")
	azuraCastApiKey := os.Getenv("AZURACAST_API_KEY")
	azuraCastDomain := os.Getenv("AZURACAST_DOMAIN")
	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")

	if fireCastSecret == "" || azuraCastApiKey == "" || azuraCastDomain == "" || redisHost == "" || redisPort == "" {
		fmt.Println("Environment variables FIRECAST_SECRET, AZURACAST_API_KEY, AZURACAST_DOMAIN, REDIS_HOST, and REDIS_PORT must be set")
		return
	}

	rdb = redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", redisHost, redisPort),
		DB:   0,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Redis connection failed: %v", err)
	}

	h := handler.NewHandler(rdb, fireCastSecret, azuraCastApiKey, azuraCastDomain)

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", h.HealthzHandler)
	r.Get("/healthz", h.HealthzHandler)

	r.Group(func(r chi.Router) {
		r.Use(h.AuthMiddleware)
		r.Get("/playlists", h.PlaylistsHandler)
		r.Post("/video/add", h.VideoAddHandler)
		r.Get("/video/get", h.VideoGetHandler)
		r.Post("/video/done", h.VideoDoneHandler)
		r.Post("/video/fail", h.VideoFailHandler)
		r.Get("/status", h.StatusHandler)
		// r.Get("/status/fail")
		// r.Get("/status/wip")
		// r.Get("/status/done")
	})

	wiprecovery.WipRecovery(ctx, rdb)

	fmt.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
