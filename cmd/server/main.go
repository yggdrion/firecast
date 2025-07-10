package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"firecast/pkg/structs"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

var (
	ctx = context.Background()
	rdb *redis.Client
)

func healthzHandler(w http.ResponseWriter, r *http.Request) {

	err := rdb.Ping(ctx).Err()
	if err != nil {
		log.Printf("Redis connection failed: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "OK")
}

func addVideoHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var videoReq structs.VideoRequest
	if err := json.NewDecoder(r.Body).Decode(&videoReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		response := structs.VideoResponse{
			Message: "Invalid JSON format",
			Success: false,
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	fmt.Printf("Received video request: URL=%s, PlaylistID=%s\n", videoReq.VideoURL, videoReq.PlaylistID)

	jsonData, err := json.Marshal(videoReq)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := structs.VideoResponse{
			Message: "Failed to encode video request",
			Success: false,
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	if err := rdb.LPush(ctx, "video_requests", jsonData).Err(); err != nil {
		log.Printf("Failed to push video request to Redis: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		response := structs.VideoResponse{
			Message: "Failed to store video request",
			Success: false,
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	response := structs.VideoResponse{
		Message: fmt.Sprintf("Video request received for URL: %s, Playlist: %s", videoReq.VideoURL, videoReq.PlaylistID),
		Success: true,
	}
	json.NewEncoder(w).Encode(response)
}

func getVideoHandler(w http.ResponseWriter, r *http.Request) {
	// pop one video request from the Redis list
	videoData, err := rdb.RPop(ctx, "video_requests").Bytes()
	if err != nil {
		if err == redis.Nil {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintln(w, "No video requests available")
			return
		}
		log.Printf("Failed to pop video request from Redis: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var videoReq structs.VideoRequest
	if err := json.Unmarshal(videoData, &videoReq); err != nil {
		log.Printf("Failed to decode video request: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	response := structs.VideoResponse{
		Message: fmt.Sprintf("Video request popped: URL=%s, PlaylistID=%s",
			videoReq.VideoURL, videoReq.PlaylistID),
		Success: true,
	}
	json.NewEncoder(w).Encode(response)
}

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

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", healthzHandler)
	r.Post("/addvideo", addVideoHandler)
	r.Get("/getvideo", getVideoHandler)

	fmt.Println("Server starting on :8080")
	http.ListenAndServe(":8080", r)
}
