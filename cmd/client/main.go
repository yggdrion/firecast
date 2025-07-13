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

func init() {
	// Load .env file if it exists
	godotenv.Load()
	fireCastSecret = os.Getenv("FIRECAST_SECRET")
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
	defer resp.Body.Close()

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
	resp, err := http.Get("http://localhost:8080/healthz")
	if err != nil {
		fmt.Println("Error making GET request:", err)
		return nil
	}
	return resp
}

func add() *http.Response {
	fmt.Println("Sending video request...")
	videoReq := structs.VideoAddRequest{
		VideoUrl:   "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		PlaylistId: 6,
	}

	jsonData, err := json.Marshal(videoReq)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return nil
	}

	req, err := createAuthenticatedRequest("POST", "http://localhost:8080/video/add", bytes.NewBuffer(jsonData))
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

	req, err := createAuthenticatedRequest("GET", "http://localhost:8080/video/get", nil)
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

	req, err := createAuthenticatedRequest("POST", "http://localhost:8080/video/done", bytes.NewBuffer(jsonData))
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

	videoUuid.Uuid = os.Args[2] // Assuming the UUID is passed as a command line argument

	jsonData, err := json.Marshal(videoUuid)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return nil
	}

	fmt.Println("Failing video:", videoUuid)

	req, err := createAuthenticatedRequest("POST", "http://localhost:8080/video/fail", bytes.NewBuffer(jsonData))
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

	req, err := createAuthenticatedRequest("GET", "http://localhost:8080/status", nil)
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
	fmt.Println("  add - Add a video")
	fmt.Println("  get - Get a video")
	fmt.Println("  done <video_uuid> - Mark a video as done")
	fmt.Println("  fail <video_uuid> - Mark a video as failed")
	fmt.Println("  status - Get the status of the service")
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
	default:
		help()
		return
	}

	printResponse(resp)
}
