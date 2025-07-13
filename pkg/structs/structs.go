package structs

type VideoAddRequest struct {
	VideoUrl   string `json:"videoUrl"`
	PlaylistId string `json:"playlistId"`
}

type VideoResponse struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
}

type VideoStore struct {
	Uuid       string `json:"uuid"`
	VideoUrl   string `json:"videoUrl"`
	PlaylistId string `json:"playlistId"`
}
