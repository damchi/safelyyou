package http

import (
	"bytes"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"safelyyou/internal/adapters/repository/memory"
	"safelyyou/internal/core/domain"
	"safelyyou/internal/core/services"
)

// newIntegrationServer wires the real memory repository, real service and
// real routes together into a Gin engine for integration testing.
func newIntegrationServer(t *testing.T) (*gin.Engine, *memory.DeviceRepository) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	repo := memory.NewDeviceRepository()
	svc := services.NewDeviceService(repo)

	r := gin.New()
	RegisterRoutes(r, svc)

	return r, repo
}

// Use a valid device ID that passes utils.IsId (hex pairs separated by dashes).
const integrationDeviceID = "60-6b-44-84-dc-64"

// seedDevice simulates that the device was loaded from devices.csv.
func seedDevice(t *testing.T, repo *memory.DeviceRepository, id string) {
	t.Helper()
	if err := repo.WithDevice(id, func(d *domain.DeviceStats) error {
		return nil
	}); err != nil {
		t.Fatalf("failed to seed device %q in repo: %v", id, err)
	}
}

//
// POST /api/v1/devices/:device_id/heartbeat
//

// Known device → 204, repo still has exactly one device.
func TestIntegration_Heartbeat_KnownDeviceReturns204(t *testing.T) {
	r, repo := newIntegrationServer(t)

	seedDevice(t, repo, integrationDeviceID)

	body := []byte(`{"sent_at":"2025-11-09T10:00:00Z"}`)
	req, err := http.NewRequest(http.MethodPost, "/api/v1/devices/"+integrationDeviceID+"/heartbeat", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d, body=%s", w.Code, w.Body.String())
	}

	if repo.Count() != 1 {
		t.Fatalf("expected repository device count = 1, got %d", repo.Count())
	}
}

// Unknown-but-valid device → service returns ErrDeviceNotFound → 404.
func TestIntegration_Heartbeat_UnknownDeviceReturns404(t *testing.T) {
	r, _ := newIntegrationServer(t)

	unknownID := "aa-bb-cc-11-22-33"

	body := []byte(`{"sent_at":"2025-11-09T10:00:00Z"}`)
	req, err := http.NewRequest(http.MethodPost, "/api/v1/devices/"+unknownID+"/heartbeat", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 for unknown device, got %d, body=%s", w.Code, w.Body.String())
	}
}

// Valid device_id but invalid JSON → 400 from full stack.
func TestIntegration_Heartbeat_InvalidPayload_Returns400(t *testing.T) {
	r, _ := newIntegrationServer(t)

	body := []byte(`{"sent_at": 123}`)
	req, err := http.NewRequest(http.MethodPost, "/api/v1/devices/"+integrationDeviceID+"/heartbeat", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d, body=%s", w.Code, w.Body.String())
	}
}

//
// POST /api/v1/devices/:device_id/stats
//

// Known device → 204 for valid stats payload.
func TestIntegration_Stats_KnownDeviceReturns204(t *testing.T) {
	r, repo := newIntegrationServer(t)

	seedDevice(t, repo, integrationDeviceID)

	body := []byte(`{"sent_at":"2025-11-09T10:00:00Z","upload_time":30000000000}`)
	req, err := http.NewRequest(http.MethodPost, "/api/v1/devices/"+integrationDeviceID+"/stats", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d, body=%s", w.Code, w.Body.String())
	}

	if repo.Count() != 1 {
		t.Fatalf("expected repository device count = 1, got %d", repo.Count())
	}
}

// Unknown device → 404 on stats post.
func TestIntegration_Stats_UnknownDeviceReturns404(t *testing.T) {
	r, _ := newIntegrationServer(t)

	body := []byte(`{"sent_at":"2025-11-09T10:00:00Z","upload_time":30000000000}`)
	req, err := http.NewRequest(http.MethodPost, "/api/v1/devices/aa-bb-cc-11-22-33/stats", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d, body=%s", w.Code, w.Body.String())
	}
}

//
// GET /api/v1/devices/:device_id/stats
//

// Full flow:
//   - seed device
//   - send a few heartbeats
//   - send a couple of stats uploads
//   - GET /stats and check the JSON payload looks sane.
func TestIntegration_GetStats_FullFlow(t *testing.T) {
	r, repo := newIntegrationServer(t)

	seedDevice(t, repo, integrationDeviceID)

	// 3 heartbeats over 60 minutes → uptime ~= (3 / 60) * 100 = 5
	hbBodies := [][]byte{
		[]byte(`{"sent_at":"2025-11-09T10:00:00Z"}`),
		[]byte(`{"sent_at":"2025-11-09T10:30:00Z"}`),
		[]byte(`{"sent_at":"2025-11-09T11:00:00Z"}`),
	}
	for i, body := range hbBodies {
		req, err := http.NewRequest(http.MethodPost, "/api/v1/devices/"+integrationDeviceID+"/heartbeat", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("failed to create heartbeat request %d: %v", i+1, err)
		}
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Fatalf("heartbeat %d: expected status 204, got %d, body=%s", i+1, w.Code, w.Body.String())
		}
	}

	// 2 stats uploads: 30s and 90s -> avg = 60s -> "1m0s"
	statsBodies := [][]byte{
		[]byte(`{"sent_at":"2025-11-09T11:05:00Z","upload_time":30000000000}`),
		[]byte(`{"sent_at":"2025-11-09T11:10:00Z","upload_time":90000000000}`),
	}
	for i, body := range statsBodies {
		req, err := http.NewRequest(http.MethodPost, "/api/v1/devices/"+integrationDeviceID+"/stats", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("failed to create stats request %d: %v", i+1, err)
		}
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Fatalf("stats %d: expected status 204, got %d, body=%s", i+1, w.Code, w.Body.String())
		}
	}

	// Now GET /stats and inspect the response.
	req, err := http.NewRequest(http.MethodGet, "/api/v1/devices/"+integrationDeviceID+"/stats", nil)
	if err != nil {
		t.Fatalf("failed to create GET stats request: %v", err)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 from GET /stats, got %d, body=%s", w.Code, w.Body.String())
	}

	var resp StatsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal GET /stats response: %v", err)
	}

	// For our chosen timestamps: uptime ≈ 5.
	if math.Abs(resp.Uptime-5.0) > 0.0001 {
		t.Fatalf("expected uptime ≈ 5, got %f", resp.Uptime)
	}

	if resp.AvgUploadTime != "1m0s" {
		t.Fatalf("expected avg_upload_time=1m0s, got %s", resp.AvgUploadTime)
	}
}

// Unknown device on GET /stats → 404.
func TestIntegration_GetStats_UnknownDeviceReturns404(t *testing.T) {
	r, _ := newIntegrationServer(t)

	req, err := http.NewRequest(http.MethodGet, "/api/v1/devices/aa-bb-cc-11-22-33/stats", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 for unknown device, got %d, body=%s", w.Code, w.Body.String())
	}
}
