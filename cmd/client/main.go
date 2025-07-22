package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"firecast/pkg/structs"

	"github.com/joho/godotenv"
)

type VideoProcessor struct {
	azuraCastAPIKey string
	azuraCastDomain string
	serverURL       string
	fireCastSecret  string
}

func NewVideoProcessor() (*VideoProcessor, error) {
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

	return &VideoProcessor{
		azuraCastAPIKey: azuraCastAPIKey,
		azuraCastDomain: azuraCastDomain,
		serverURL:       serverURL,
		fireCastSecret:  fireCastSecret,
	}, nil
}

func (vp *VideoProcessor) downloadVideoAsMP3(videoURL string) (string, error) {
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

func (vp *VideoProcessor) uploadToAzuraCast(localFile string) (int, error) {
	apiURL := fmt.Sprintf("https://%s/api/station/1/files", vp.azuraCastDomain)

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

	req.Header.Set("X-API-Key", vp.azuraCastAPIKey)
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

func (vp *VideoProcessor) assignPlaylistToSong(songID, playlistID int) error {
	apiURL := fmt.Sprintf("https://%s/api/station/1/file/%d", vp.azuraCastDomain, songID)

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

	req.Header.Set("X-API-Key", vp.azuraCastAPIKey)
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

func (vp *VideoProcessor) getNextVideo() (*structs.VideoResponse, error) {
	req, err := http.NewRequest("GET", vp.serverURL+"/video/get", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+vp.fireCastSecret)

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

func (vp *VideoProcessor) markVideoComplete(uuid string) error {
	data := structs.VideoDoneRequest{Uuid: uuid}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}

	req, err := http.NewRequest("POST", vp.serverURL+"/video/done", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+vp.fireCastSecret)
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

func (vp *VideoProcessor) markVideoFailed(uuid string) error {
	data := structs.VideoFailRequest{Uuid: uuid}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}

	req, err := http.NewRequest("POST", vp.serverURL+"/video/fail", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+vp.fireCastSecret)
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

func (vp *VideoProcessor) processVideo(video *structs.VideoResponse) error {
	log.Printf("Processing video: %s (UUID: %s, Playlist: %d)", video.VideoUrl, video.Uuid, video.PlaylistId)

	// Download video as MP3
	mp3File, err := vp.downloadVideoAsMP3(video.VideoUrl)
	if err != nil {
		return fmt.Errorf("failed to download video: %v", err)
	}
	defer func() {
		if err := os.Remove(mp3File); err != nil {
			log.Printf("Warning: failed to remove file %s: %v", mp3File, err)
		}
	}()

	// Upload to AzuraCast
	songID, err := vp.uploadToAzuraCast(mp3File)
	if err != nil {
		return fmt.Errorf("failed to upload to AzuraCast: %v", err)
	}

	// Assign to playlist
	if err := vp.assignPlaylistToSong(songID, video.PlaylistId); err != nil {
		return fmt.Errorf("failed to assign playlist: %v", err)
	}

	log.Printf("Successfully processed video %s -> song ID %d", video.VideoUrl, songID)
	return nil
}

func (vp *VideoProcessor) run() {
	log.Println("Starting simplified video processing...")

	for {
		// Get next video to process
		video, err := vp.getNextVideo()
		if err != nil {
			log.Printf("Error getting next video: %v", err)
			log.Println("Waiting 10 seconds before trying again...")
			time.Sleep(10 * time.Second)
			continue
		}

		if video == nil {
			log.Println("No videos to process, waiting 10 seconds...")
			time.Sleep(10 * time.Second)
			continue
		}

		// Process the video
		log.Printf("Found video to process: %s", video.VideoUrl)
		if err := vp.processVideo(video); err != nil {
			log.Printf("Error processing video %s: %v", video.VideoUrl, err)
			if markErr := vp.markVideoFailed(video.Uuid); markErr != nil {
				log.Printf("Error marking video as failed: %v", markErr)
			}
			continue
		}

		// Mark as complete
		if err := vp.markVideoComplete(video.Uuid); err != nil {
			log.Printf("Error marking video as complete: %v", err)
		}

		log.Printf("Completed processing video: %s", video.VideoUrl)
		// Immediately look for the next video (no delay)
	}
}

func main() {
	log.Println("Starting Firecast Simple Video Processor...")

	processor, err := NewVideoProcessor()
	if err != nil {
		log.Fatalf("Failed to create video processor: %v", err)
	}

	processor.run()
}
