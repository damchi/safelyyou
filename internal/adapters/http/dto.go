package http

import "time"

type HeartbeatRequest struct {
	SentAt time.Time `json:"sent_at" binding:"required"`
}

type ErrorResponse struct {
	Msg string `json:"msg"`
}
