package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"firecast/pkg/structs"
)

func main() {
	fmt.Println("This is the client main function")

	videoReq := structs.VideoRequest{
		VideoURL:   "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		PlaylistID: "6",
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

	var videoResp structs.VideoResponse
	if err := json.NewDecoder(resp.Body).Decode(&videoResp); err != nil {
		fmt.Println("Error parsing response JSON:", err)
		return
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error: received status code %d\n", resp.StatusCode)
		fmt.Printf("Error message: %s\n", videoResp.Message)
		return
	}

	fmt.Printf("Success: %t\n", videoResp.Success)
	fmt.Printf("Message: %s\n", videoResp.Message)
	fmt.Println("Video added successfully!")
	fmt.Println("Client operation completed.")
}
