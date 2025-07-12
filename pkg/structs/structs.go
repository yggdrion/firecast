package structs

type VideoRequest struct {
	VideoUrl   string `json:"videoUrl"`
	PlaylistId string `json:"playlistId"`
}

type VideoResponse struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
}
