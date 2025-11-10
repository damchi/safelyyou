package http

import "time"

type HeartbeatRequest struct {
	SentAt time.Time `json:"sent_at" binding:"required"`
}

type ErrorResponse struct {
	Msg string `json:"msg"`
}

type StatsRequest struct {
	SentAt     time.Time `json:"sent_at"`
	UploadTime int64     `json:"upload_time" binding:"required,gte=0"`
}

type StatsResponse struct {
	Uptime        float64 `json:"uptime"`
	AvgUploadTime string  `json:"avg_upload_time"`
}
