package handler

import (
	"encoding/json"
	"firecast/pkg/structs"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/lithammer/shortuuid/v4"
	"github.com/redis/go-redis/v9"
)

type Handler struct {
	rdb             *redis.Client
	fireCastSecret  string
	azuraCastAPIKey string
	azuraCastDomain string
}

func NewHandler(rdb *redis.Client, fireCastSecret, azuraCastAPIKey, azuraCastDomain string) *Handler {
	return &Handler{
		rdb:             rdb,
		fireCastSecret:  fireCastSecret,
		azuraCastAPIKey: azuraCastAPIKey,
		azuraCastDomain: azuraCastDomain,
	}
}

func (h *Handler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.WriteHeader(http.StatusUnauthorized)
			if err := json.NewEncoder(w).Encode(map[string]any{
				"success": false,
				"message": "Authorization required",
			}); err != nil {
				log.Printf("Error encoding JSON response: %v", err)
			}
			return
		}

		token := authHeader
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token = authHeader[7:]
		}

		if token != h.fireCastSecret {
			w.WriteHeader(http.StatusUnauthorized)
			if err := json.NewEncoder(w).Encode(map[string]any{
				"success": false,
				"message": "Invalid secret",
			}); err != nil {
				log.Printf("Error encoding JSON response: %v", err)
			}
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (h *Handler) HealthzHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	err := h.rdb.Ping(ctx).Err()
	if err != nil {
		log.Printf("Redis connection failed: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"message": "Redis connection failed",
		}); err != nil {
			log.Printf("Error encoding JSON response: %v", err)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"message": "ok",
	}); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

func (h *Handler) VideoAddHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	var videoReq structs.VideoAddRequest
	if err := json.NewDecoder(r.Body).Decode(&videoReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"status":  false,
			"message": "Invalid JSON format",
		}); err != nil {
			log.Printf("Error encoding JSON response: %v", err)
		}
		return
	}

	if videoReq.VideoUrl == "" || videoReq.PlaylistId == 0 {
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"status":  false,
			"message": "VideoUrl and PlaylistId are required",
		}); err != nil {
			log.Printf("Error encoding JSON response: %v", err)
		}
		return
	}

	videoUuid := shortuuid.New()

	if err := h.rdb.LPush(ctx, "videos:queue", videoUuid).Err(); err != nil {
		log.Printf("Failed to push video request to Redis: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"status":  false,
			"message": "Failed to store video request",
		}); err != nil {
			log.Printf("Error encoding JSON response: %v", err)
		}
		return
	}

	meta := map[string]any{
		"url":             videoReq.VideoUrl,
		"playlist_id":     videoReq.PlaylistId,
		"retries":         0,
		"added_at":        time.Now().Unix(),
		"last_attempt_at": time.Now().Unix(),
	}
	if err := h.rdb.HSet(ctx, fmt.Sprintf("videos:meta:%s", videoUuid), meta).Err(); err != nil {
		log.Printf("Failed to set video metadata: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"status":  false,
			"message": "Failed to store video metadata",
		}); err != nil {
			log.Printf("Error encoding JSON response: %v", err)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]any{
		"status":  true,
		"message": "ok",
		"uuid":    videoUuid,
	}); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

func (h *Handler) VideoGetHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	videoUuid, err := h.rdb.RPop(ctx, "videos:queue").Result()
	if err != nil {
		if err == redis.Nil {
			w.WriteHeader(http.StatusNoContent)
			if err := json.NewEncoder(w).Encode(map[string]any{
				"success": false,
				"message": "No video requests available",
			}); err != nil {
				log.Printf("Error encoding JSON response: %v", err)
			}
			return
		}
		log.Printf("Failed to pop video request from Redis: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"message": "Failed to pop video request from Redis",
		}); err != nil {
			log.Printf("Error encoding JSON response: %v", err)
		}
		return
	}

	videoData, err := h.rdb.HGetAll(ctx, fmt.Sprintf("videos:meta:%s", videoUuid)).Result()
	if err != nil {
		log.Printf("Failed to get video metadata from Redis: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"message": "Failed to retrieve video metadata",
		}); err != nil {
			log.Printf("Error encoding JSON response: %v", err)
		}
		return
	}

	if len(videoData) == 0 {
		w.WriteHeader(http.StatusNotFound)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"message": "Video metadata not found",
		}); err != nil {
			log.Printf("Error encoding JSON response: %v", err)
		}
		return
	}

	if err := h.rdb.ZAdd(ctx, "videos:wip", redis.Z{
		Score:  float64(time.Now().Unix() + 60),
		Member: videoUuid,
	}).Err(); err != nil {
		log.Printf("Failed to add video to wip: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"message": "Failed to add video to wip",
		}); err != nil {
			log.Printf("Error encoding JSON response: %v", err)
		}
		return
	}

	if _, err := h.rdb.HIncrBy(ctx, fmt.Sprintf("videos:meta:%s", videoUuid), "retries", 1).Result(); err != nil {
		log.Printf("Failed to increment retry count for video %s: %v", videoUuid, err)
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"message": "Failed to increment retry count",
		}); err != nil {
			log.Printf("Error encoding JSON response: %v", err)
		}
		return
	}

	retries, _ := strconv.Atoi(videoData["retries"])
	addedAt, _ := strconv.ParseInt(videoData["added_at"], 10, 64)
	lastAttemptAt, _ := strconv.ParseInt(videoData["last_attempt_at"], 10, 64)
	playlistId, _ := strconv.Atoi(videoData["playlist_id"])

	videoResponse := structs.VideoResponse{
		Uuid:          videoUuid,
		VideoUrl:      videoData["url"],
		PlaylistId:    playlistId,
		Retries:       retries + 1,
		AddedAt:       addedAt,
		LastAttemptAt: lastAttemptAt,
	}
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(videoResponse); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

func (h *Handler) VideoFailHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	var failReq structs.VideoFailRequest
	if err := json.NewDecoder(r.Body).Decode(&failReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"status":  false,
			"message": "Invalid JSON format",
		}); err != nil {
			log.Printf("Error encoding JSON response: %v", err)
		}
		return
	}
	videoUuid := failReq.Uuid
	if videoUuid == "" {
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"status":  false,
			"message": "UUID is required",
		}); err != nil {
			log.Printf("Error encoding JSON response: %v", err)
		}
		return
	}

	isFailed, err := h.rdb.SIsMember(ctx, "videos:fail", videoUuid).Result()
	if err != nil {
		log.Printf("Failed to check fail set: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"status":  false,
			"message": "Failed to check fail set",
		}); err != nil {
			log.Printf("Error encoding JSON response: %v", err)
		}
		return
	}
	if isFailed {
		w.WriteHeader(http.StatusConflict)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"status":  false,
			"message": "Already marked as failed",
		}); err != nil {
			log.Printf("Error encoding JSON response: %v", err)
		}
		return
	}

	isDone, err := h.rdb.SIsMember(ctx, "videos:done", videoUuid).Result()
	if err != nil {
		log.Printf("Failed to check done set: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"status":  false,
			"message": "Failed to check done set",
		}); err != nil {
			log.Printf("Error encoding JSON response: %v", err)
		}
		return
	}
	if isDone {
		w.WriteHeader(http.StatusConflict)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"status":  false,
			"message": "Already marked as done",
		}); err != nil {
			log.Printf("Error encoding JSON response: %v", err)
		}
		return
	}

	if err := h.rdb.ZRem(ctx, "videos:wip", videoUuid).Err(); err != nil {
		log.Printf("Failed to remove video from wip: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"status":  false,
			"message": "Failed to remove video from wip",
		}); err != nil {
			log.Printf("Error encoding JSON response: %v", err)
		}
		return
	}

	if err := h.rdb.SAdd(ctx, "videos:fail", videoUuid).Err(); err != nil {
		log.Printf("Failed to add video to failed set: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"status":  false,
			"message": "Failed to add video to failed set",
		}); err != nil {
			log.Printf("Error encoding JSON response: %v", err)
		}
		return
	}
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]any{
		"status":  true,
		"message": "Video marked as failed",
		"uuid":    videoUuid,
	}); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

func (h *Handler) VideoDoneHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	var doneReq structs.VideoDoneRequest
	if err := json.NewDecoder(r.Body).Decode(&doneReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"status":  false,
			"message": "Invalid JSON format",
		}); err != nil {
			log.Printf("Error encoding JSON response: %v", err)
		}
		return
	}

	videoUuid := doneReq.Uuid
	if videoUuid == "" {
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"status":  false,
			"message": "UUID is required",
		}); err != nil {
			log.Printf("Error encoding JSON response: %v", err)
		}
		return
	}

	isDone, err := h.rdb.SIsMember(ctx, "videos:done", videoUuid).Result()
	if err != nil {
		log.Printf("Failed to check done set: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"status":  false,
			"message": "Failed to check done set",
		}); err != nil {
			log.Printf("Error encoding JSON response: %v", err)
		}
		return
	}
	if isDone {
		w.WriteHeader(http.StatusConflict)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"status":  false,
			"message": "Already marked as done",
		}); err != nil {
			log.Printf("Error encoding JSON response: %v", err)
		}
		return
	}

	isFailed, err := h.rdb.SIsMember(ctx, "videos:fail", videoUuid).Result()
	if err != nil {
		log.Printf("Failed to check fail set: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"status":  false,
			"message": "Failed to check fail set",
		}); err != nil {
			log.Printf("Error encoding JSON response: %v", err)
		}
		return
	}
	if isFailed {
		w.WriteHeader(http.StatusConflict)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"status":  false,
			"message": "Already marked as failed",
		}); err != nil {
			log.Printf("Error encoding JSON response: %v", err)
		}
		return
	}

	if err := h.rdb.ZRem(ctx, "videos:wip", videoUuid).Err(); err != nil {
		log.Printf("Failed to remove video from wip: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"status":  false,
			"message": "Failed to remove video from wip",
		}); err != nil {
			log.Printf("Error encoding JSON response: %v", err)
		}
		return
	}

	if err := h.rdb.SAdd(ctx, "videos:done", videoUuid).Err(); err != nil {
		log.Printf("Failed to add video to done set: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"status":  false,
			"message": "Failed to mark video as done",
		}); err != nil {
			log.Printf("Error encoding JSON response: %v", err)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]any{
		"status":  true,
		"message": "Video marked as done",
	}); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

func (h *Handler) StatusHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	wipCount, err := h.rdb.ZCard(ctx, "videos:wip").Result()
	if err != nil {
		log.Printf("Failed to get wip count: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"status":  false,
			"message": "Failed to get wip count",
		}); err != nil {
			log.Printf("Error encoding JSON response: %v", err)
		}
		return
	}

	doneCount, err := h.rdb.SCard(ctx, "videos:done").Result()
	if err != nil {
		log.Printf("Failed to get done count: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"status":  false,
			"message": "Failed to get done count",
		}); err != nil {
			log.Printf("Error encoding JSON response: %v", err)
		}
		return
	}

	failedCount, err := h.rdb.SCard(ctx, "videos:fail").Result()
	if err != nil {
		log.Printf("Failed to get failed count: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"status":  false,
			"message": "Failed to get failed count",
		}); err != nil {
			log.Printf("Error encoding JSON response: %v", err)
		}
		return
	}

	queueLength, err := h.rdb.LLen(ctx, "videos:queue").Result()
	if err != nil {
		log.Printf("Failed to get queue length: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"status":  false,
			"message": "Failed to get queue length",
		}); err != nil {
			log.Printf("Error encoding JSON response: %v", err)
		}
		return
	}

	statusResponse := structs.StatusResponse{
		WipCount:    int(wipCount),
		DoneCount:   int(doneCount),
		FailCount:   int(failedCount),
		QueueLength: int(queueLength),
	}
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(statusResponse); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

func (h *Handler) PlaylistsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	apiURL := fmt.Sprintf("https://%s/api/station/1/playlists", h.azuraCastDomain)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		log.Printf("Failed to create request: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"message": "Failed to create request",
		}); err != nil {
			log.Printf("Error encoding JSON response: %v", err)
		}
		return
	}

	req.Header.Set("X-API-Key", h.azuraCastAPIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to send request: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"message": "Failed to send request to AzuraCast",
		}); err != nil {
			log.Printf("Error encoding JSON response: %v", err)
		}
		return
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Warning: failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("AzuraCast API error: %d %s", resp.StatusCode, string(body))
		w.WriteHeader(http.StatusBadGateway)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"message": fmt.Sprintf("AzuraCast API error: %d %s", resp.StatusCode, string(body)),
		}); err != nil {
			log.Printf("Error encoding JSON response: %v", err)
		}
		return
	}

	var playlists []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&playlists); err != nil {
		log.Printf("Failed to parse JSON response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"message": "Failed to parse JSON response",
		}); err != nil {
			log.Printf("Error encoding JSON response: %v", err)
		}
		return
	}

	result := make(map[string]int)
	for _, playlist := range playlists {
		name, ok := playlist["name"].(string)
		if !ok {
			continue
		}
		id, ok := playlist["id"].(float64)
		if !ok {
			continue
		}
		result[name] = int(id)
	}

	if len(result) == 0 {
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"message": "No playlists found in AzuraCast",
		}); err != nil {
			log.Printf("Error encoding JSON response: %v", err)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}
