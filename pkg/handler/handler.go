package handler

import (
	"encoding/json"
	"firecast/pkg/structs"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
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
		"message": "ok",
	})
}

func (h *Handler) VideoAddHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	var videoReq structs.VideoAddRequest
	if err := json.NewDecoder(r.Body).Decode(&videoReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"status":  false,
			"message": "Invalid JSON format",
		})
		return
	}

	videoUuid := uuid.New().String()

	if err := h.rdb.LPush(ctx, "videos:queue", videoUuid).Err(); err != nil {
		log.Printf("Failed to push video request to Redis: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{
			"status":  false,
			"message": "Failed to store video request",
		})
		return
	}

	h.rdb.HSet(ctx, fmt.Sprintf("videos:meta:%s", videoUuid), map[string]any{
		"url":             videoReq.VideoUrl,
		"playlist_id":     videoReq.PlaylistId,
		"retries":         0,
		"added_at":        time.Now().Unix(),
		"last_attempt_at": time.Now().Unix(),
	})

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"status":  true,
		"message": "ok",
		"uuid":    videoUuid,
	})
}

func (h *Handler) VideoGetHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	videoUuid, err := h.rdb.RPop(ctx, "videos:queue").Result()
	if err != nil {
		if err.Error() == "redis: nil" {
			w.WriteHeader(http.StatusNoContent)
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

	videoData, err := h.rdb.HGetAll(ctx, fmt.Sprintf("videos:meta:%s", videoUuid)).Result()
	if err != nil {
		log.Printf("Failed to get video metadata from Redis: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"message": "Failed to retrieve video metadata",
		})
		return
	}

	h.rdb.ZAdd(ctx, "videos:wip", redis.Z{
		Score:  float64(time.Now().Unix() + 60),
		Member: videoUuid,
	})

	_, err = h.rdb.HIncrBy(ctx, fmt.Sprintf("videos:meta:%s", videoUuid), "retries", 1).Result()
	if err != nil {
		log.Printf("Failed to increment retry count for video %s: %v", videoUuid, err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"message": "Failed to increment retry count",
		})
		return
	}

	retries, _ := strconv.ParseInt(videoData["retries"], 10, 64)
	addedAt, _ := strconv.ParseInt(videoData["added_at"], 10, 64)
	lastAttemptAt, _ := strconv.ParseInt(videoData["last_attempt_at"], 10, 64)
	playlistId, _ := strconv.Atoi(videoData["playlist_id"])

	videoResponse := structs.VideoResponse{
		Uuid:          videoUuid,
		VideoUrl:      videoData["url"],
		PlaylistId:    playlistId,
		Retries:       int(retries),
		AddedAt:       addedAt,
		LastAttemptAt: lastAttemptAt,
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(videoResponse)
}
