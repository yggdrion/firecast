package structs

type VideoRequest struct {
	VideoURL   string `json:"video_url"`
	PlaylistID string `json:"playlist_id"`
}

type VideoResponse struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
}
