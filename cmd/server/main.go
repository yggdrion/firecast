package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"firecast/pkg/structs"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func usersHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Users endpoint")
}

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
	json.NewEncoder(w).Encode(response)
}

func main() {
	r := chi.NewRouter()

	r.Use(middleware.Logger)

	r.Get("/users", usersHandler)
	r.Get("/health", healthHandler)
	r.Post("/addvideo", addVideoHandler)

	http.ListenAndServe(":8080", r)
}
