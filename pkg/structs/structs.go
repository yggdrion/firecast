package structs

type VideoAddRequest struct {
	VideoUrl   string `json:"videoUrl"`
	PlaylistId int    `json:"playlistId"`
}

// "hset" "videos:meta:b346f6ac-62fc-4d5e-ad10-532379c9e4d8"
// "url" "https://www.youtube.com/watch?v=dQw4w9WgXcQ" "playlist_id" "6" "retries" "0" "added_at" "1752416738" "last_attempt_at" "1752416738"

type VideoResponse struct {
	Uuid          string `json:"uuid"`
	VideoUrl      string `json:"videoUrl"`
	PlaylistId    int    `json:"playlistId"`
	Retries       int    `json:"retries"`
	AddedAt       int64  `json:"addedAt"`
	LastAttemptAt int64  `json:"lastAttemptAt"`
}

type VideoStore struct {
	Uuid       string `json:"uuid"`
	VideoUrl   string `json:"videoUrl"`
	PlaylistId int    `json:"playlistId"`
}
