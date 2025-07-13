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

	resp, err := http.Post("http://localhost:8080/video/add", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error making POST request:", err)
		return nil
	}
	return resp
}

func get() *http.Response {
	fmt.Println("Retrieving video...")
	resp, err := http.Get("http://localhost:8080/video/get")
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
	resp, err := http.Post("http://localhost:8080/video/done", "application/json", bytes.NewBuffer([]byte(jsonData)))
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
