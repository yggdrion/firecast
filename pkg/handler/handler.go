package handler

import (
	"encoding/json"
	"firecast/pkg/structs"
	"fmt"
	"log"
	"net/http"

	"github.com/redis/go-redis/v9"
)

type Handler struct {
	rdb *redis.Client
}

func NewHandler(rdb *redis.Client) *Handler {
	return &Handler{rdb: rdb}
}

func (h *Handler) HealthzHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	err := h.rdb.Ping(ctx).Err()
	if err != nil {
		log.Printf("Redis connection failed: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"message": "Redis connection failed",
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"message": "OK",
	})
}

func (h *Handler) AddVideoHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	var videoReq structs.VideoRequest
	if err := json.NewDecoder(r.Body).Decode(&videoReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"status":  false,
			"message": "Invalid JSON format",
		})
		return
	}

	jsonData, err := json.Marshal(videoReq)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{
			"status":  false,
			"message": "Failed to encode video request",
		})
		return
	}

	if err := h.rdb.LPush(ctx, "video_requests", jsonData).Err(); err != nil {
		log.Printf("Failed to push video request to Redis: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{
			"status":  false,
			"message": "Failed to store video request",
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]any{
		"status":  true,
		"message": fmt.Sprintf("Video request received for URL: %s, Playlist: %s", videoReq.VideoUrl, videoReq.PlaylistId),
	})
}

func (h *Handler) GetVideoHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	videoData, err := h.rdb.RPop(ctx, "video_requests").Bytes()
	if err != nil {
		if err.Error() == "redis: nil" {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{
				"success": false,
				"message": "No video requests available",
			})
			return
		}
		log.Printf("Failed to pop video request from Redis: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"message": "Failed to pop video request from Redis",
		})
		return
	}

	var videoReq structs.VideoRequest
	if err := json.Unmarshal(videoData, &videoReq); err != nil {
		log.Printf("Failed to decode video request: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"message": "Failed to decode video request",
		})
		return
	}

	response := structs.VideoResponse{
		Message: fmt.Sprintf("Video request popped: URL=%s, PlaylistID=%s",
			videoReq.VideoUrl, videoReq.PlaylistId),
		Success: true,
	}
	json.NewEncoder(w).Encode(response)
}
