package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	coreerrors "safelyyou/internal/core/errors"
	"safelyyou/internal/core/ports"
)

// testDeviceService implements ports.DeviceService and records calls for assertions.
type testDeviceService struct {
	lastHeartbeatID     string
	lastHeartbeatSentAt time.Time
	lastStatsID         string
	lastStatsSentAt     time.Time
	lastUploadTimeNs    int64
	lastGetStatsID      string
	heartbeatErr        error
	statsErr            error
	getStatsResult      *ports.Stats
	getStatsErr         error
}

func (s *testDeviceService) RecordHeartbeat(id string, sentAt time.Time) error {
	s.lastHeartbeatID = id
	s.lastHeartbeatSentAt = sentAt
	return s.heartbeatErr
}

func (s *testDeviceService) RecordStats(id string, sentAt time.Time, uploadNs int64) error {
	s.lastStatsID = id
	s.lastStatsSentAt = sentAt
	s.lastUploadTimeNs = uploadNs
	return s.statsErr
}

func (s *testDeviceService) GetStats(id string) (*ports.Stats, error) {
	s.lastGetStatsID = id
	return s.getStatsResult, s.getStatsErr
}

// Use a valid device ID
const validDeviceID = "60-6b-44-84-dc-64"

//
// PostHeartbeat tests
//

func TestPostHeartbeat_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &testDeviceService{}
	h := NewHandler(svc)

	r := gin.New()
	r.POST("/api/v1/devices/:device_id/heartbeat", h.PostHeartbeat)

	body := []byte(`{"sent_at":"2025-11-09T10:00:00Z"}`)
	req, err := http.NewRequest(http.MethodPost, "/api/v1/devices/"+validDeviceID+"/heartbeat", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d, body=%s", w.Code, w.Body.String())
	}

	if svc.lastHeartbeatID != validDeviceID {
		t.Fatalf("expected lastHeartbeatID=%s, got %s", validDeviceID, svc.lastHeartbeatID)
	}

	expectedTime := time.Date(2025, 11, 9, 10, 0, 0, 0, time.UTC)
	if !svc.lastHeartbeatSentAt.Equal(expectedTime) {
		t.Fatalf("expected sent_at=%v, got %v", expectedTime, svc.lastHeartbeatSentAt)
	}
}

func TestPostHeartbeat_InvalidID_Returns400(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &testDeviceService{}
	h := NewHandler(svc)

	r := gin.New()
	r.POST("/api/v1/devices/:device_id/heartbeat", h.PostHeartbeat)

	body := []byte(`{"sent_at":"2025-11-09T10:00:00Z"}`)
	req, err := http.NewRequest(http.MethodPost, "/api/v1/devices/bad-id/heartbeat", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for invalid device_id, got %d", w.Code)
	}

	// Ensure service was never called.
	if svc.lastHeartbeatID != "" {
		t.Fatalf("expected service not to be called, got lastHeartbeatID=%s", svc.lastHeartbeatID)
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
	req, err := http.NewRequest(http.MethodPost, "/api/v1/devices/"+validDeviceID+"/heartbeat", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
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
	req, err := http.NewRequest(http.MethodPost, "/api/v1/devices/"+validDeviceID+"/heartbeat", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
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
		heartbeatErr: coreerrors.ErrDeviceNotFound,
	}
	h := NewHandler(svc)

	r := gin.New()
	r.POST("/api/v1/devices/:device_id/heartbeat", h.PostHeartbeat)

	body := []byte(`{"sent_at":"2025-11-09T10:00:00Z"}`)
	req, err := http.NewRequest(http.MethodPost, "/api/v1/devices/"+validDeviceID+"/heartbeat", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
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

	body := []byte(`{"sent_at":"2025-11-09T10:00:00Z"}`)
	req, err := http.NewRequest(http.MethodPost, "/api/v1/devices/"+validDeviceID+"/heartbeat", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}
}

//
// PostStats tests
//

func TestPostStats_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &testDeviceService{}
	h := NewHandler(svc)

	r := gin.New()
	r.POST("/api/v1/devices/:device_id/stats", h.PostStats)

	body := []byte(`{"sent_at":"2025-11-09T10:00:00Z","upload_time":30000000000}`)
	req, err := http.NewRequest(http.MethodPost, "/api/v1/devices/"+validDeviceID+"/stats", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d, body=%s", w.Code, w.Body.String())
	}

	if svc.lastStatsID != validDeviceID {
		t.Fatalf("expected lastStatsID=%s, got %s", validDeviceID, svc.lastStatsID)
	}

	expectedTime := time.Date(2025, 11, 9, 10, 0, 0, 0, time.UTC)
	if !svc.lastStatsSentAt.Equal(expectedTime) {
		t.Fatalf("expected sent_at=%v, got %v", expectedTime, svc.lastStatsSentAt)
	}

	if svc.lastUploadTimeNs != 30000000000 {
		t.Fatalf("expected upload_time=30000000000, got %d", svc.lastUploadTimeNs)
	}
}

func TestPostStats_InvalidID_Returns400(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &testDeviceService{}
	h := NewHandler(svc)

	r := gin.New()
	r.POST("/api/v1/devices/:device_id/stats", h.PostStats)

	body := []byte(`{"sent_at":"2025-11-09T10:00:00Z","upload_time":30000000000}`)
	req, err := http.NewRequest(http.MethodPost, "/api/v1/devices/bad-id/stats", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for invalid device_id, got %d", w.Code)
	}

	if svc.lastStatsID != "" {
		t.Fatalf("expected service not to be called, got lastStatsID=%s", svc.lastStatsID)
	}
}

func TestPostStats_InvalidJSON_Returns400(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &testDeviceService{}
	h := NewHandler(svc)

	r := gin.New()
	r.POST("/api/v1/devices/:device_id/stats", h.PostStats)

	// invalid upload_time type
	body := []byte(`{"sent_at":"2025-11-09T10:00:00Z","upload_time":"oops"}`)
	req, err := http.NewRequest(http.MethodPost, "/api/v1/devices/"+validDeviceID+"/stats", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestPostStats_DeviceNotFound_Returns404(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &testDeviceService{
		statsErr: coreerrors.ErrDeviceNotFound,
	}
	h := NewHandler(svc)

	r := gin.New()
	r.POST("/api/v1/devices/:device_id/stats", h.PostStats)

	body := []byte(`{"sent_at":"2025-11-09T10:00:00Z","upload_time":30000000000}`)
	req, err := http.NewRequest(http.MethodPost, "/api/v1/devices/"+validDeviceID+"/stats", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
}

func TestPostStats_InternalError_Returns500(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &testDeviceService{
		statsErr: errors.New("boom"),
	}
	h := NewHandler(svc)

	r := gin.New()
	r.POST("/api/v1/devices/:device_id/stats", h.PostStats)

	body := []byte(`{"sent_at":"2025-11-09T10:00:00Z","upload_time":30000000000}`)
	req, err := http.NewRequest(http.MethodPost, "/api/v1/devices/"+validDeviceID+"/stats", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}
}

//
// GetStats tests
//

func TestGetStats_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &testDeviceService{
		getStatsResult: &ports.Stats{
			Uptime:        98.75,
			AvgUploadTime: "3m17.331667813s",
		},
	}
	h := NewHandler(svc)

	r := gin.New()
	r.GET("/api/v1/devices/:device_id/stats", h.GetStats)

	req, err := http.NewRequest(http.MethodGet, "/api/v1/devices/"+validDeviceID+"/stats", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}

	// Verify service was called with correct id.
	if svc.lastGetStatsID != validDeviceID {
		t.Fatalf("expected lastGetStatsID=%s, got %s", validDeviceID, svc.lastGetStatsID)
	}

	// Verify JSON response body.
	var resp StatsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response body: %v", err)
	}

	if resp.Uptime != 98.75 {
		t.Fatalf("expected uptime=98.75, got %f", resp.Uptime)
	}
	if resp.AvgUploadTime != "3m17.331667813s" {
		t.Fatalf("expected avg_upload_time=3m17.331667813s, got %s", resp.AvgUploadTime)
	}
}

func TestGetStats_InvalidID_Returns400(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &testDeviceService{}
	h := NewHandler(svc)

	r := gin.New()
	r.GET("/api/v1/devices/:device_id/stats", h.GetStats)

	req, err := http.NewRequest(http.MethodGet, "/api/v1/devices/bad-id/stats", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for invalid device_id, got %d", w.Code)
	}
}

func TestGetStats_DeviceNotFound_Returns404(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &testDeviceService{
		getStatsErr: coreerrors.ErrDeviceNotFound,
	}
	h := NewHandler(svc)

	r := gin.New()
	r.GET("/api/v1/devices/:device_id/stats", h.GetStats)

	req, err := http.NewRequest(http.MethodGet, "/api/v1/devices/"+validDeviceID+"/stats", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
}

func TestGetStats_InternalError_Returns500(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &testDeviceService{
		getStatsErr: errors.New("boom"),
	}
	h := NewHandler(svc)

	r := gin.New()
	r.GET("/api/v1/devices/:device_id/stats", h.GetStats)

	req, err := http.NewRequest(http.MethodGet, "/api/v1/devices/"+validDeviceID+"/stats", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}
}
