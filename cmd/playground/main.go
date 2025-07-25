package main

import (
	"bytes"
	"encoding/json"
	"firecast/pkg/structs"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

var fireCastSecret string
var fireCastUrl string

func init() {
	_ = godotenv.Load()
	fireCastSecret = os.Getenv("FIRECAST_SECRET")
	fireCastUrl = os.Getenv("FIRECAST_DOMAIN")
	if fireCastUrl == "" {
		fireCastUrl = "http://localhost:8080" // fallback to localhost if not set
	}
}

func createAuthenticatedRequest(method, url string, body *bytes.Buffer) (*http.Request, error) {
	var req *http.Request
	var err error

	if body != nil {
		req, err = http.NewRequest(method, url, body)
	} else {
		req, err = http.NewRequest(method, url, nil)
	}

	if err != nil {
		return nil, err
	}

	if method == "POST" {
		req.Header.Set("Content-Type", "application/json")
	}

	if fireCastSecret != "" {
		req.Header.Set("Authorization", "Bearer "+fireCastSecret)
	}

	return req, nil
}

func printResponse(resp *http.Response) {
	if resp == nil {
		fmt.Println("No response received.")
		return
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Warning: failed to close response body: %v\n", err)
		}
	}()

	fmt.Println("Response Status Code:", resp.StatusCode)
	fmt.Println("Response Headers:", resp.Header)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}
	fmt.Println("Response Body:", string(body))
}

func printPlaylistsResponse(resp *http.Response) {
	if resp == nil {
		fmt.Println("No response received.")
		return
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Warning: failed to close response body: %v\n", err)
		}
	}()

	fmt.Println("Response Status Code:", resp.StatusCode)
	fmt.Println("Response Headers:", resp.Header)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}
	fmt.Println("Response Body:", string(body))

}

func health() *http.Response {
	resp, err := http.Get(fireCastUrl + "/healthz")
	if err != nil {
		fmt.Println("Error making GET request:", err)
		return nil
	}
	return resp
}

func add() *http.Response {
	if len(os.Args) < 3 {
		fmt.Println("Error: YouTube video URL is required")
		fmt.Println("Usage: go run main.go add <youtube_url>")
		return nil
	}

	videoUrl := os.Args[2]
	fmt.Println("Sending video request...")
	videoReq := structs.VideoAddRequest{
		VideoUrl:   videoUrl,
		PlaylistId: 6,
	}

	jsonData, err := json.Marshal(videoReq)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return nil
	}

	req, err := createAuthenticatedRequest("POST", fireCastUrl+"/video/add", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return nil
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making POST request:", err)
		return nil
	}
	return resp
}

func get() *http.Response {
	fmt.Println("Retrieving video...")

	req, err := createAuthenticatedRequest("GET", fireCastUrl+"/video/get", nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return nil
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making GET request:", err)
		return nil
	}
	return resp
}

func done() *http.Response {

	var videoUuid structs.VideoDoneRequest

	videoUuid.Uuid = os.Args[2]

	jsonData, err := json.Marshal(videoUuid)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return nil
	}

	fmt.Println("Marking video as done:", videoUuid)

	req, err := createAuthenticatedRequest("POST", fireCastUrl+"/video/done", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return nil
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making POST request:", err)
		return nil
	}
	return resp
}

func fail() *http.Response {

	var videoUuid structs.VideoDoneRequest

	videoUuid.Uuid = os.Args[2]

	jsonData, err := json.Marshal(videoUuid)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return nil
	}

	fmt.Println("Failing video:", videoUuid)

	req, err := createAuthenticatedRequest("POST", fireCastUrl+"/video/fail", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return nil
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making POST request:", err)
		return nil
	}
	return resp
}

func status() *http.Response {
	fmt.Println("Retrieving status...")

	req, err := createAuthenticatedRequest("GET", fireCastUrl+"/status", nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return nil
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making GET request:", err)
		return nil
	}
	return resp
}

func playlists() *http.Response {
	fmt.Println("Retrieving playlists...")

	req, err := createAuthenticatedRequest("GET", fireCastUrl+"/playlists", nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return nil
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making GET request:", err)
		return nil
	}
	return resp
}

func help() {
	fmt.Println("Usage: go run main.go <command>")
	fmt.Println("Commands:")
	fmt.Println("  health - Check the health of the service")
	fmt.Println("  add <youtube_url> - Add a video")
	fmt.Println("  get - Get a video")
	fmt.Println("  done <video_uuid> - Mark a video as done")
	fmt.Println("  fail <video_uuid> - Mark a video as failed")
	fmt.Println("  status - Get the status of the service")
	fmt.Println("  playlists - Get all playlists")
}

func main() {
	if len(os.Args) <= 1 {

		help()

		if len(os.Args) <= 2 {
			help()
			return
		}

		return
	}

	command := os.Args[1]
	var resp *http.Response

	switch command {
	case "health":
		resp = health()
	case "add":
		resp = add()
	case "get":
		resp = get()
	case "done":
		resp = done()
	case "fail":
		resp = fail()
	case "status":
		resp = status()
	case "playlists":
		resp = playlists()
	default:
		help()
		return
	}

	if command == "playlists" {
		printPlaylistsResponse(resp)
	} else {
		printResponse(resp)
	}
}
