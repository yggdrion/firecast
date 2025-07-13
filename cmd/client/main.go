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

func healthCheck() *http.Response {
	resp, err := http.Get("http://localhost:8080/healthz")
	if err != nil {
		fmt.Println("Error making GET request:", err)
		return nil
	}
	return resp
}

func addVideo() *http.Response {
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

	resp, err := http.Post("http://localhost:8080/video/add", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error making POST request:", err)
		return nil
	}
	return resp
}

func getVideo() *http.Response {
	fmt.Println("Retrieving video...")
	resp, err := http.Get("http://localhost:8080/video/get")
	if err != nil {
		fmt.Println("Error making GET request:", err)
		return nil
	}
	return resp
}

func doneVideo() *http.Response {

	var videoUuid structs.VideoDoneRequest

	videoUuid.Uuid = os.Args[2] // Assuming the UUID is passed as a command line argument

	jsonData, err := json.Marshal(videoUuid)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return nil
	}

	fmt.Println("Marking video as done:", videoUuid)
	resp, err := http.Post("http://localhost:8080/video/done", "application/json", bytes.NewBuffer([]byte(jsonData)))
	if err != nil {
		fmt.Println("Error making POST request:", err)
		return nil
	}
	return resp
}

func failVideo() *http.Response {

	var videoUuid structs.VideoDoneRequest

	videoUuid.Uuid = os.Args[2] // Assuming the UUID is passed as a command line argument

	jsonData, err := json.Marshal(videoUuid)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return nil
	}

	fmt.Println("Failing video:", videoUuid)
	resp, err := http.Post("http://localhost:8080/video/fail", "application/json", bytes.NewBuffer([]byte(jsonData)))
	if err != nil {
		fmt.Println("Error making POST request:", err)
		return nil
	}
	return resp
}

func status() *http.Response {
	fmt.Println("Retrieving status...")
	resp, err := http.Get("http://localhost:8080/status")
	if err != nil {
		fmt.Println("Error making GET request:", err)
		return nil
	}
	return resp
}

func main() {
	if len(os.Args) <= 1 {
		fmt.Println("Usage: go run main.go <command>")
		fmt.Println("Commands: health, addvideo, getvideo, donevideo, failvideo")

		if len(os.Args) <= 2 {
			fmt.Println("donevideo, and failvideo, provide a video UUID as the second argument.")
		}

		return
	}

	command := os.Args[1]
	var resp *http.Response

	switch command {
	case "health":
		resp = healthCheck()
	case "addvideo":
		resp = addVideo()
	case "getvideo":
		resp = getVideo()
	case "donevideo":
		resp = doneVideo()
	case "failvideo":
		resp = failVideo()
	case "status":
		resp = status()
	default:
		fmt.Println("Unknown command:", command)
		fmt.Println("Available commands: health, addvideo, getvideo")
		return
	}

	printResponse(resp)
}
