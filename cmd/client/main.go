package main

import (
	"bytes"
	"encoding/json"
	"firecast/pkg/structs"
	"fmt"
	"io"
	"net/http"
	"os"
)

func healthCheck() {
	healthReq, err := http.Get("http://localhost:8080/healthz")
	if err != nil {
		fmt.Println("Error creating health check request:", err)
		return
	}
	if healthReq.StatusCode != http.StatusOK {
		fmt.Printf("Health check failed with status code: %d\n", healthReq.StatusCode)
		return
	}
	fmt.Println("Health check successful, server is running.")

	defer healthReq.Body.Close()
}

func addVideo() {
	fmt.Println("Sending video request...")
	videoReq := structs.VideoRequest{
		VideoUrl:   "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		PlaylistId: "6",
	}

	jsonData, err := json.Marshal(videoReq)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return
	}

	resp, err := http.Post("http://localhost:8080/addvideo", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error making POST request:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("Response Status Code:", resp.StatusCode)
	fmt.Println("Response Headers:", resp.Header)

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}
	fmt.Println("Response Body:", string(body))
}

func getVideo() {
	fmt.Println("Retrieving video...")
	resp, err := http.Get("http://localhost:8080/getvideo")
	if err != nil {
		fmt.Println("Error making GET request:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("Response Status Code:", resp.StatusCode)
	fmt.Println("Response Headers:", resp.Header)

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}
	fmt.Println("Response Body:", string(body))
}

func main() {
	// params := os.Args[1:]
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <command>")
		fmt.Println("Commands: health, addvideo, getvideo")
		return
	}
	command := os.Args[1]
	switch command {
	case "health":
		healthCheck()
	case "addvideo":
		addVideo()
	case "getvideo":
		getVideo()
	default:
		fmt.Println("Unknown command:", command)
		fmt.Println("Available commands: health, addvideo, getvideo")
	}

}
