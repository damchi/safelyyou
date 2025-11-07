package http

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"safelyyou/internal/adapters/repository/memory"
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

// Integration test:
//   - Hit POST /api/v1/devices/{device_id}/heartbeat
//   - Expect 204
//   - Expect the in-memory repository to contain exactly one device.
func TestIntegration_Heartbeat_CreatesDeviceInRepository(t *testing.T) {
	r, repo := newIntegrationServer(t)

	deviceID := "device-123"

	body := []byte(`{"sent_at":"2025-01-01T10:00:00Z"}`)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/devices/"+deviceID+"/heartbeat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d, body=%s", w.Code, w.Body.String())
	}

	// The repository should have exactly one device entry.
	if repo.Count() != 1 {
		t.Fatalf("expected repository device count = 1, got %d", repo.Count())
	}
}

// Integration test:
//   - Call POST /heartbeat twice for the same device id
//   - Expect 204 both times
//   - Expect repository.Count() to still be 1 (no duplicate devices)
func TestIntegration_Heartbeat_MultipleRequestsSingleDevice(t *testing.T) {
	r, repo := newIntegrationServer(t)

	deviceID := "device-123"

	body := []byte(`{"sent_at":"2025-01-01T10:00:00Z"}`)

	for i := 0; i < 2; i++ {
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/devices/"+deviceID+"/heartbeat", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Fatalf("call %d: expected status 204, got %d, body=%s", i+1, w.Code, w.Body.String())
		}
	}

	if repo.Count() != 1 {
		t.Fatalf("expected repository device count = 1 after two calls, got %d", repo.Count())
	}
}

// Integration test:
//   - Send invalid JSON to POST /heartbeat
//   - Expect 400 from the full stack (HTTP -> handler -> validation)
func TestIntegration_Heartbeat_InvalidPayload(t *testing.T) {
	r, _ := newIntegrationServer(t)

	deviceID := "device-123"

	// Invalid: sent_at is a number instead of a string / date-time
	body := []byte(`{"sent_at": 123}`)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/devices/"+deviceID+"/heartbeat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d, body=%s", w.Code, w.Body.String())
	}
}
