package http

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	coreerrors "safelyyou/internal/core/errors"
	"safelyyou/internal/core/ports"
)

// testDeviceService implements ports.DeviceService and records calls.
type testDeviceService struct {
	lastID     string
	lastSentAt time.Time

	heartbeatErr error
	statsErr     error
	statsResult  *ports.Stats
}

func (s *testDeviceService) RecordHeartbeat(id string, sentAt time.Time) error {
	s.lastID = id
	s.lastSentAt = sentAt
	return s.heartbeatErr
}

func TestPostHeartbeat_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &testDeviceService{}
	h := NewHandler(svc)

	r := gin.New()
	// We test the full path as it will be called at runtime.
	r.POST("/api/v1/devices/:device_id/heartbeat", h.PostHeartbeat)

	body := []byte(`{"sent_at":"2025-01-01T10:00:00Z"}`)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/devices/device-123/heartbeat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d, body=%s", w.Code, w.Body.String())
	}

	if svc.lastID != "device-123" {
		t.Fatalf("expected lastID=device-123, got %s", svc.lastID)
	}

	expectedTime := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	if !svc.lastSentAt.Equal(expectedTime) {
		t.Fatalf("expected sent_at=%v, got %v", expectedTime, svc.lastSentAt)
	}
}

func TestPostHeartbeat_InvalidJSON_Returns400(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &testDeviceService{}
	h := NewHandler(svc)

	r := gin.New()
	r.POST("/api/v1/devices/:device_id/heartbeat", h.PostHeartbeat)

	// invalid type for sent_at
	body := []byte(`{"sent_at": 123}`)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/devices/device-123/heartbeat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

// Missing sent_at field should also be a 400 due to binding:"required".
func TestPostHeartbeat_MissingSentAt_Returns400(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &testDeviceService{}
	h := NewHandler(svc)

	r := gin.New()
	r.POST("/api/v1/devices/:device_id/heartbeat", h.PostHeartbeat)

	body := []byte(`{}`)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/devices/device-123/heartbeat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

// Service reports device not found -> handler should return 404.
func TestPostHeartbeat_DeviceNotFound_Returns404(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &testDeviceService{
		heartbeatErr: coreerrors.ErrDeviceNotFound, // sentinel from your core/errors
	}
	h := NewHandler(svc)

	r := gin.New()
	r.POST("/api/v1/devices/:device_id/heartbeat", h.PostHeartbeat)

	body := []byte(`{"sent_at":"2025-01-01T10:00:00Z"}`)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/devices/missing-device/heartbeat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
}

// Service returns some unexpected error -> handler should return 500.
func TestPostHeartbeat_InternalError_Returns500(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &testDeviceService{
		heartbeatErr: errors.New("boom"),
	}
	h := NewHandler(svc)

	r := gin.New()
	r.POST("/api/v1/devices/:device_id/heartbeat", h.PostHeartbeat)

	body := []byte(`{"sent_at":"2025-01-01T10:00:00Z"}`)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/devices/device-123/heartbeat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}
}
