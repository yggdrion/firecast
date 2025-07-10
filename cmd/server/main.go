package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"firecast/pkg/structs"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	// fmt.Printf("Health check from: %s at %s\n", r.RemoteAddr, time.Now().Format(time.RFC3339))
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

	response := structs.VideoResponse{
		Message: fmt.Sprintf("Video request received for URL: %s, Playlist: %s", videoReq.VideoURL, videoReq.PlaylistID),
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

	// Setup routes
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", healthHandler)
	r.Post("/addvideo", addVideoHandler)

	fmt.Println("Server starting on :8080")
	http.ListenAndServe(":8080", r)
}
