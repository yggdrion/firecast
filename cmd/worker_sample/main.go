package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"firecast/pkg/structs"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

type Worker struct {
	rdb             *redis.Client
	azuraCastAPIKey string
	azuraCastDomain string
	serverURL       string
	fireCastSecret  string
}

func NewWorker() (*Worker, error) {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: Error loading .env file")
	}

	azuraCastAPIKey := os.Getenv("AZURACAST_API_KEY")
	azuraCastDomain := os.Getenv("AZURACAST_DOMAIN")
	serverURL := os.Getenv("SERVER_URL")
	fireCastSecret := os.Getenv("FIRECAST_SECRET")

	if azuraCastAPIKey == "" || azuraCastDomain == "" || fireCastSecret == "" {
		return nil, fmt.Errorf("required environment variables AZURACAST_API_KEY, AZURACAST_DOMAIN, and FIRECAST_SECRET must be set")
	}

	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %v", err)
	}

	return &Worker{
		rdb:             rdb,
		azuraCastAPIKey: azuraCastAPIKey,
		azuraCastDomain: azuraCastDomain,
		serverURL:       serverURL,
		fireCastSecret:  fireCastSecret,
	}, nil
}

func (w *Worker) downloadVideoAsMP3(videoURL string) (string, error) {
	// Ensure downloads directory exists
	if err := os.MkdirAll("downloads", 0755); err != nil {
		return "", fmt.Errorf("failed to create downloads directory: %v", err)
	}

	// Use yt-dlp to download and convert to MP3
	cmd := exec.Command("yt-dlp",
		"--format", "bestaudio/best",
		"--extract-audio",
		"--audio-format", "mp3",
		"--audio-quality", "192K",
		"--output", "downloads/%(title)s.%(ext)s",
		videoURL,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("yt-dlp failed: %v, stderr: %s", err, stderr.String())
	}

	// Find the downloaded MP3 file
	files, err := filepath.Glob("downloads/*.mp3")
	if err != nil {
		return "", fmt.Errorf("failed to find downloaded files: %v", err)
	}

	if len(files) == 0 {
		return "", fmt.Errorf("no MP3 file found after download")
	}

	// Return the most recently created file
	var newestFile string
	var newestTime time.Time
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		if info.ModTime().After(newestTime) {
			newestTime = info.ModTime()
			newestFile = file
		}
	}

	if newestFile == "" {
		return "", fmt.Errorf("could not determine newest downloaded file")
	}

	return newestFile, nil
}

func (w *Worker) uploadToAzuraCast(localFile string) (int, error) {
	apiURL := fmt.Sprintf("https://%s/api/station/1/files", w.azuraCastDomain)

	// Read file content
	fileContent, err := os.ReadFile(localFile)
	if err != nil {
		return 0, fmt.Errorf("failed to read file: %v", err)
	}

	// Encode to base64
	encodedContent := base64.StdEncoding.EncodeToString(fileContent)

	// Prepare request data
	data := map[string]interface{}{
		"path": filepath.Base(localFile),
		"file": encodedContent,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal JSON: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("X-API-Key", w.azuraCastAPIKey)
	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("AzuraCast API upload error: %d %s", resp.StatusCode, string(body))
	}

	// Parse response
	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return 0, fmt.Errorf("failed to decode response: %v", err)
	}

	songID, ok := response["id"].(float64)
	if !ok {
		return 0, fmt.Errorf("failed to get song ID from response")
	}

	log.Printf("Uploaded %s with song ID %d", localFile, int(songID))
	return int(songID), nil
}

func (w *Worker) assignPlaylistToSong(songID, playlistID int) error {
	apiURL := fmt.Sprintf("https://%s/api/station/1/file/%d", w.azuraCastDomain, songID)

	data := map[string]interface{}{
		"playlists": []interface{}{
			map[string]interface{}{"id": playlistID},
			0,
		},
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}

	req, err := http.NewRequest("PUT", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("X-API-Key", w.azuraCastAPIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("AzuraCast API playlist assignment error: %d %s", resp.StatusCode, string(body))
	}

	return nil
}

func (w *Worker) getVideoJob(ctx context.Context) (*structs.VideoResponse, error) {
	req, err := http.NewRequest("GET", w.serverURL+"/video/get", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+w.fireCastSecret)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil, nil // No jobs available
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server error: %d %s", resp.StatusCode, string(body))
	}

	var video structs.VideoResponse
	if err := json.NewDecoder(resp.Body).Decode(&video); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &video, nil
}

func (w *Worker) markVideoComplete(uuid string) error {
	data := structs.VideoDoneRequest{Uuid: uuid}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}

	req, err := http.NewRequest("POST", w.serverURL+"/video/done", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+w.fireCastSecret)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server error: %d %s", resp.StatusCode, string(body))
	}

	return nil
}

func (w *Worker) markVideoFailed(uuid string) error {
	data := structs.VideoFailRequest{Uuid: uuid}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}

	req, err := http.NewRequest("POST", w.serverURL+"/video/fail", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+w.fireCastSecret)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server error: %d %s", resp.StatusCode, string(body))
	}

	return nil
}

func (w *Worker) processVideo(video *structs.VideoResponse) error {
	log.Printf("Processing video: %s (UUID: %s, Playlist: %d)", video.VideoUrl, video.Uuid, video.PlaylistId)

	// Download video as MP3
	mp3File, err := w.downloadVideoAsMP3(video.VideoUrl)
	if err != nil {
		return fmt.Errorf("failed to download video: %v", err)
	}
	defer func() {
		if err := os.Remove(mp3File); err != nil {
			log.Printf("Warning: failed to remove file %s: %v", mp3File, err)
		}
	}()

	// Upload to AzuraCast
	songID, err := w.uploadToAzuraCast(mp3File)
	if err != nil {
		return fmt.Errorf("failed to upload to AzuraCast: %v", err)
	}

	// Assign to playlist
	if err := w.assignPlaylistToSong(songID, video.PlaylistId); err != nil {
		return fmt.Errorf("failed to assign playlist: %v", err)
	}

	log.Printf("Successfully processed video %s -> song ID %d", video.VideoUrl, songID)
	return nil
}

func (w *Worker) run(ctx context.Context) {
	pollIntervalStr := os.Getenv("WORKER_POLL_INTERVAL")
	if pollIntervalStr == "" {
		pollIntervalStr = "5" // 5 seconds default
	}
	pollInterval, err := strconv.Atoi(pollIntervalStr)
	if err != nil {
		log.Printf("Invalid WORKER_POLL_INTERVAL value: %s, using default 5", pollIntervalStr)
		pollInterval = 5
	}

	log.Printf("Worker started, polling every %d seconds", pollInterval)

	ticker := time.NewTicker(time.Duration(pollInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Worker stopping...")
			return
		case <-ticker.C:
			video, err := w.getVideoJob(ctx)
			if err != nil {
				log.Printf("Error getting video job: %v", err)
				continue
			}

			if video == nil {
				// No jobs available, continue polling
				continue
			}

			// Process the video
			if err := w.processVideo(video); err != nil {
				log.Printf("Error processing video %s: %v", video.VideoUrl, err)
				if markErr := w.markVideoFailed(video.Uuid); markErr != nil {
					log.Printf("Error marking video as failed: %v", markErr)
				}
				continue
			}

			// Mark as complete
			if err := w.markVideoComplete(video.Uuid); err != nil {
				log.Printf("Error marking video as complete: %v", err)
			}
		}
	}
}

func main() {
	log.Println("Starting Firecast Worker...")

	worker, err := NewWorker()
	if err != nil {
		log.Fatalf("Failed to create worker: %v", err)
	}

	ctx := context.Background()
	worker.run(ctx)
}
