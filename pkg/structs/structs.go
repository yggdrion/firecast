package structs

type VideoAddRequest struct {
	VideoUrl   string `json:"videoUrl"`
	PlaylistId int    `json:"playlistId"`
}

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

type VideoFailRequest struct {
	Uuid string `json:"uuid"`
}
type VideoDoneRequest struct {
	Uuid string `json:"uuid"`
}

type StatusResponse struct {
	WipCount    int `json:"wipCount"`
	DoneCount   int `json:"doneCount"`
	FailCount   int `json:"failCount"`
	QueueLength int `json:"queueLength"`
}
