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

// Helper methods for JSON responses
func (h *Handler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

func (h *Handler) writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	h.writeJSONResponse(w, statusCode, map[string]interface{}{
		"status":  false,
		"message": message,
	})
}

func (h *Handler) writeSuccessResponse(w http.ResponseWriter, data interface{}) {
	h.writeJSONResponse(w, http.StatusOK, data)
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
			h.writeErrorResponse(w, http.StatusUnauthorized, "Authorization required")
			return
		}

		token := authHeader
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token = authHeader[7:]
		}

		if token != h.fireCastSecret {
			h.writeErrorResponse(w, http.StatusUnauthorized, "Invalid secret")
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
		h.writeErrorResponse(w, http.StatusInternalServerError, "Redis connection failed")
		return
	}

	h.writeSuccessResponse(w, map[string]interface{}{
		"success": true,
		"message": "ok",
	})
}

func (h *Handler) VideoAddHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	var videoReq structs.VideoAddRequest
	if err := json.NewDecoder(r.Body).Decode(&videoReq); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON format")
		return
	}

	if videoReq.VideoUrl == "" || videoReq.PlaylistId == 0 {
		h.writeErrorResponse(w, http.StatusBadRequest, "VideoUrl and PlaylistId are required")
		return
	}

	videoUuid := shortuuid.New()

	if err := h.rdb.LPush(ctx, "videos:queue", videoUuid).Err(); err != nil {
		log.Printf("Failed to push video request to Redis: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to store video request")
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
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to store video metadata")
		return
	}

	h.writeSuccessResponse(w, map[string]interface{}{
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
		if err == redis.Nil {
			// 204 No Content should not have a body
			w.WriteHeader(http.StatusNoContent)
			return
		}
		log.Printf("Failed to pop video request from Redis: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to pop video request from Redis")
		return
	}

	videoData, err := h.rdb.HGetAll(ctx, fmt.Sprintf("videos:meta:%s", videoUuid)).Result()
	if err != nil {
		log.Printf("Failed to get video metadata from Redis: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve video metadata")
		return
	}

	if len(videoData) == 0 {
		h.writeErrorResponse(w, http.StatusNotFound, "Video metadata not found")
		return
	}

	if err := h.rdb.ZAdd(ctx, "videos:wip", redis.Z{
		Score:  float64(time.Now().Unix() + 60),
		Member: videoUuid,
	}).Err(); err != nil {
		log.Printf("Failed to add video to wip: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to add video to wip")
		return
	}

	if _, err := h.rdb.HIncrBy(ctx, fmt.Sprintf("videos:meta:%s", videoUuid), "retries", 1).Result(); err != nil {
		log.Printf("Failed to increment retry count for video %s: %v", videoUuid, err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to increment retry count")
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
	h.writeSuccessResponse(w, videoResponse)
}

func (h *Handler) VideoFailHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	var failReq structs.VideoFailRequest
	if err := json.NewDecoder(r.Body).Decode(&failReq); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON format")
		return
	}

	videoUuid := failReq.Uuid
	if videoUuid == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "UUID is required")
		return
	}

	isFailed, err := h.rdb.SIsMember(ctx, "videos:fail", videoUuid).Result()
	if err != nil {
		log.Printf("Failed to check fail set: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to check fail set")
		return
	}
	if isFailed {
		h.writeErrorResponse(w, http.StatusConflict, "Already marked as failed")
		return
	}

	isDone, err := h.rdb.SIsMember(ctx, "videos:done", videoUuid).Result()
	if err != nil {
		log.Printf("Failed to check done set: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to check done set")
		return
	}
	if isDone {
		h.writeErrorResponse(w, http.StatusConflict, "Already marked as done")
		return
	}

	if err := h.rdb.ZRem(ctx, "videos:wip", videoUuid).Err(); err != nil {
		log.Printf("Failed to remove video from wip: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to remove video from wip")
		return
	}

	if err := h.rdb.SAdd(ctx, "videos:fail", videoUuid).Err(); err != nil {
		log.Printf("Failed to add video to failed set: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to add video to failed set")
		return
	}

	h.writeSuccessResponse(w, map[string]interface{}{
		"status":  true,
		"message": "Video marked as failed",
		"uuid":    videoUuid,
	})
}

func (h *Handler) VideoDoneHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	var doneReq structs.VideoDoneRequest
	if err := json.NewDecoder(r.Body).Decode(&doneReq); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON format")
		return
	}

	videoUuid := doneReq.Uuid
	if videoUuid == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "UUID is required")
		return
	}

	isDone, err := h.rdb.SIsMember(ctx, "videos:done", videoUuid).Result()
	if err != nil {
		log.Printf("Failed to check done set: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to check done set")
		return
	}
	if isDone {
		h.writeErrorResponse(w, http.StatusConflict, "Already marked as done")
		return
	}

	isFailed, err := h.rdb.SIsMember(ctx, "videos:fail", videoUuid).Result()
	if err != nil {
		log.Printf("Failed to check fail set: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to check fail set")
		return
	}
	if isFailed {
		h.writeErrorResponse(w, http.StatusConflict, "Already marked as failed")
		return
	}

	if err := h.rdb.ZRem(ctx, "videos:wip", videoUuid).Err(); err != nil {
		log.Printf("Failed to remove video from wip: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to remove video from wip")
		return
	}

	if err := h.rdb.SAdd(ctx, "videos:done", videoUuid).Err(); err != nil {
		log.Printf("Failed to add video to done set: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to mark video as done")
		return
	}

	h.writeSuccessResponse(w, map[string]interface{}{
		"status":  true,
		"message": "Video marked as done",
	})
}

func (h *Handler) StatusHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	wipCount, err := h.rdb.ZCard(ctx, "videos:wip").Result()
	if err != nil {
		log.Printf("Failed to get wip count: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to get wip count")
		return
	}

	doneCount, err := h.rdb.SCard(ctx, "videos:done").Result()
	if err != nil {
		log.Printf("Failed to get done count: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to get done count")
		return
	}

	failedCount, err := h.rdb.SCard(ctx, "videos:fail").Result()
	if err != nil {
		log.Printf("Failed to get failed count: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to get failed count")
		return
	}

	queueLength, err := h.rdb.LLen(ctx, "videos:queue").Result()
	if err != nil {
		log.Printf("Failed to get queue length: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to get queue length")
		return
	}

	statusResponse := structs.StatusResponse{
		WipCount:    int(wipCount),
		DoneCount:   int(doneCount),
		FailCount:   int(failedCount),
		QueueLength: int(queueLength),
	}
	h.writeSuccessResponse(w, statusResponse)
}

func (h *Handler) PlaylistsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	apiURL := fmt.Sprintf("https://%s/api/station/1/playlists", h.azuraCastDomain)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		log.Printf("Failed to create request: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to create request")
		return
	}

	req.Header.Set("X-API-Key", h.azuraCastAPIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to send request: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to send request to AzuraCast")
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
		h.writeJSONResponse(w, http.StatusBadGateway, map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("AzuraCast API error: %d %s", resp.StatusCode, string(body)),
		})
		return
	}

	var playlists []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&playlists); err != nil {
		log.Printf("Failed to parse JSON response: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to parse JSON response")
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
		h.writeErrorResponse(w, http.StatusInternalServerError, "No playlists found in AzuraCast")
		return
	}

	h.writeSuccessResponse(w, result)
}
