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

	if err := storeVideoRequestInFile(videoReq); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response.Message = fmt.Sprintf("Error storing video request: %s", err)
		response.Success = false
	}

	json.NewEncoder(w).Encode(response)
}

func storeVideoRequestInFile(videoReq structs.VideoRequest) error {
	file, err := os.OpenFile("video_requests.json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()
	jsonData, err := json.Marshal(videoReq)
	if err != nil {
		return fmt.Errorf("error marshalling JSON: %w", err)
	}
	if _, err := file.Write(jsonData); err != nil {
		return fmt.Errorf("error writing to file: %w", err)
	}
	if _, err := file.WriteString("\n"); err != nil {
		return fmt.Errorf("error writing newline to file: %w", err)
	}
	return nil
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

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", healthHandler)
	r.Post("/addvideo", addVideoHandler)

	http.ListenAndServe(":8080", r)
}
