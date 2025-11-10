package services

import (
	"errors"
	"math"
	"testing"
	"time"

	"safelyyou/internal/core/domain"
	coreerrors "safelyyou/internal/core/errors"
)

// fakeDeviceRepo is a tiny in-memory DeviceRepository used only for tests.
type fakeDeviceRepo struct {
	devices map[string]*domain.DeviceStats
}

func newFakeDeviceRepo() *fakeDeviceRepo {
	return &fakeDeviceRepo{
		devices: make(map[string]*domain.DeviceStats),
	}
}

// WithDevice runs fn on the device, creating it if missing.
func (r *fakeDeviceRepo) WithDevice(id string, fn func(d *domain.DeviceStats) error) error {
	d, ok := r.devices[id]
	if !ok {
		d = domain.NewDeviceStats(id)
		r.devices[id] = d
	}
	return fn(d)
}

// Exists reports whether a device with the given id is present.
func (r *fakeDeviceRepo) Exists(id string) bool {
	_, ok := r.devices[id]
	return ok
}

// GetSnapshot returns a copy of the device stats or ErrDeviceNotFound.
func (r *fakeDeviceRepo) GetSnapshot(id string) (*domain.DeviceStats, error) {
	d, ok := r.devices[id]
	if !ok {
		return nil, coreerrors.ErrDeviceNotFound
	}
	deviceCopy := *d
	return &deviceCopy, nil
}

// -----------------------------------------------------------------------------
// Tests for RecordHeartbeat
// -----------------------------------------------------------------------------

func TestRecordHeartbeat_FirstHeartbeatInitializesStats(t *testing.T) {
	repo := newFakeDeviceRepo()
	id := "device-123"

	repo.devices[id] = domain.NewDeviceStats(id)

	svc := NewDeviceService(repo)

	t1 := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)

	if err := svc.RecordHeartbeat(id, t1); err != nil {
		t.Fatalf("RecordHeartbeat returned error: %v", err)
	}

	device, err := repo.GetSnapshot(id)
	if err != nil {
		t.Fatalf("GetSnapshot returned error: %v", err)
	}
	if device == nil {
		t.Fatalf("expected device %q to exist in repo", id)
	}

	if device.HeartbeatCount != 1 {
		t.Errorf("expected HeartbeatCount=1, got %d", device.HeartbeatCount)
	}
	if !device.FirstHeartbeat.Equal(t1) {
		t.Errorf("expected FirstHeartbeat=%v, got %v", t1, device.FirstHeartbeat)
	}
	if !device.LastHeartbeat.Equal(t1) {
		t.Errorf("expected LastHeartbeat=%v, got %v", t1, device.LastHeartbeat)
	}
}

func TestRecordHeartbeat_MultipleHeartbeatsUpdatesFirstAndLast(t *testing.T) {
	repo := newFakeDeviceRepo()
	id := "device-123"
	repo.devices[id] = domain.NewDeviceStats(id)

	svc := NewDeviceService(repo)

	t1 := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	t2 := t1.Add(5 * time.Minute)
	t3 := t2.Add(10 * time.Minute)

	if err := svc.RecordHeartbeat(id, t1); err != nil {
		t.Fatalf("RecordHeartbeat(t1) returned error: %v", err)
	}
	if err := svc.RecordHeartbeat(id, t2); err != nil {
		t.Fatalf("RecordHeartbeat(t2) returned error: %v", err)
	}
	if err := svc.RecordHeartbeat(id, t3); err != nil {
		t.Fatalf("RecordHeartbeat(t3) returned error: %v", err)
	}

	device, err := repo.GetSnapshot(id)
	if err != nil {
		t.Fatalf("GetSnapshot returned error: %v", err)
	}
	if device == nil {
		t.Fatalf("expected device %q to exist in repo", id)
	}

	if device.HeartbeatCount != 3 {
		t.Errorf("expected HeartbeatCount=3, got %d", device.HeartbeatCount)
	}
	if !device.FirstHeartbeat.Equal(t1) {
		t.Errorf("expected FirstHeartbeat=%v, got %v", t1, device.FirstHeartbeat)
	}
	if !device.LastHeartbeat.Equal(t3) {
		t.Errorf("expected LastHeartbeat=%v, got %v", t3, device.LastHeartbeat)
	}
}

func TestRecordHeartbeat_OutOfOrderEarlierUpdatesFirst(t *testing.T) {
	repo := newFakeDeviceRepo()
	id := "device-123"
	repo.devices[id] = domain.NewDeviceStats(id)

	svc := NewDeviceService(repo)

	tMiddle := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	tLate := tMiddle.Add(30 * time.Minute)
	tEarly := tMiddle.Add(-1 * time.Hour)

	if err := svc.RecordHeartbeat(id, tMiddle); err != nil {
		t.Fatalf("RecordHeartbeat(tNiddle) returned error: %v", err)
	}
	if err := svc.RecordHeartbeat(id, tLate); err != nil {
		t.Fatalf("RecordHeartbeat(tLate) returned error: %v", err)
	}
	if err := svc.RecordHeartbeat(id, tEarly); err != nil {
		t.Fatalf("RecordHeartbeat returned error: %v", err)
	}

	device, err := repo.GetSnapshot(id)
	if err != nil {
		t.Fatalf("GetSnapshot returned error: %v", err)
	}
	if device == nil {
		t.Fatalf("expected device %q to exist in repo", id)
	}

	if device.HeartbeatCount != 3 {
		t.Errorf("expected HeartbeatCount=3, got %d", device.HeartbeatCount)
	}
	if !device.FirstHeartbeat.Equal(tEarly) {
		t.Errorf("expected FirstHeartbeat=%v, got %v", tEarly, device.FirstHeartbeat)
	}
	if !device.LastHeartbeat.Equal(tLate) {
		t.Errorf("expected LastHeartbeat=%v, got %v", tLate, device.LastHeartbeat)
	}
}

func TestRecordHeartbeat_UnknownDeviceReturnsErrDeviceNotFound(t *testing.T) {
	repo := newFakeDeviceRepo()
	svc := NewDeviceService(repo)

	id := "does-not-exist"
	err := svc.RecordHeartbeat(id, time.Now())
	if !errors.Is(err, coreerrors.ErrDeviceNotFound) {
		t.Fatalf("expected ErrDeviceNotFound, got %v", err)
	}
}

// -----------------------------------------------------------------------------
// Tests for RecordStats
// -----------------------------------------------------------------------------

func TestRecordStats_UpdatesUploadFieldsOnly(t *testing.T) {
	repo := newFakeDeviceRepo()
	id := "device-123"
	d := domain.NewDeviceStats(id)

	// Pretend we already had some heartbeats.
	t1 := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	t2 := t1.Add(10 * time.Minute)
	d.HeartbeatCount = 3
	d.FirstHeartbeat = t1
	d.LastHeartbeat = t2

	repo.devices[id] = d
	svc := NewDeviceService(repo)

	uploadNs := int64(30 * time.Second)
	if err := svc.RecordStats(id, time.Time{}, uploadNs); err != nil {
		t.Fatalf("RecordStats returned error: %v", err)
	}

	device, err := repo.GetSnapshot(id)
	if err != nil {
		t.Fatalf("GetSnapshot returned error: %v", err)
	}
	if device == nil {
		t.Fatalf("expected device %q to exist in repo", id)
	}
	if device.UploadCount != 1 {
		t.Errorf("expected UploadCount=1, got %d", device.UploadCount)
	}
	if device.UploadSumMs != uploadNs {
		t.Errorf("expected UploadSumMs=%d, got %d", uploadNs, device.UploadSumMs)
	}

	// Ensure heartbeats weren't touched.
	if device.HeartbeatCount != 3 {
		t.Errorf("expected HeartbeatCount to remain 3, got %d", device.HeartbeatCount)
	}
	if !device.FirstHeartbeat.Equal(t1) || !device.LastHeartbeat.Equal(t2) {
		t.Errorf("expected heartbeat window [%v,%v], got [%v,%v]",
			t1, t2, device.FirstHeartbeat, device.LastHeartbeat)
	}
}

func TestRecordStats_NegativeUploadReturnsError(t *testing.T) {
	repo := newFakeDeviceRepo()
	id := "device-123"
	repo.devices[id] = domain.NewDeviceStats(id)

	svc := NewDeviceService(repo)

	err := svc.RecordStats(id, time.Time{}, -1)
	if err == nil {
		t.Fatalf("expected error for negative upload_time, got nil")
	}
}

func TestRecordStats_UnknownDeviceReturnsErrDeviceNotFound(t *testing.T) {
	repo := newFakeDeviceRepo()
	svc := NewDeviceService(repo)

	id := "does-not-exist"
	err := svc.RecordStats(id, time.Time{}, 123)
	if !errors.Is(err, coreerrors.ErrDeviceNotFound) {
		t.Fatalf("expected ErrDeviceNotFound, got %v", err)
	}
}

// -----------------------------------------------------------------------------
// Tests for GetStats
// -----------------------------------------------------------------------------

func TestGetStats_DeviceNotFoundReturnsError(t *testing.T) {
	repo := newFakeDeviceRepo()
	svc := NewDeviceService(repo)

	stats, err := svc.GetStats("missing-id")

	if !errors.Is(err, coreerrors.ErrDeviceNotFound) {
		t.Fatalf("expected ErrDeviceNotFound, got %v", err)
	}

	if stats != nil {
		t.Fatalf("expected stats to be nil when error is returned, got %+v", stats)
	}
}

func TestGetStats_ComputesUptimeAndAvgUpload(t *testing.T) {
	repo := newFakeDeviceRepo()
	id := "device-123"

	d := domain.NewDeviceStats(id)

	// Heartbeats: 60 heartbeats over 60 minutes → uptime ~ 100%
	t1 := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	t2 := t1.Add(60 * time.Minute)
	d.HeartbeatCount = 60
	d.FirstHeartbeat = t1
	d.LastHeartbeat = t2

	// Uploads: two uploads: 30s and 90s → average = 60s = "1m0s"
	sum := int64(30*time.Second + 90*time.Second) // in ns
	d.UploadCount = 2
	d.UploadSumMs = sum

	repo.devices[id] = d
	svc := NewDeviceService(repo)

	stats, err := svc.GetStats(id)
	if err != nil {
		t.Fatalf("GetStats returned error: %v", err)
	}

	if math.Abs(stats.Uptime-100.0) > 0.0001 {
		t.Errorf("expected Uptime ≈ 100, got %f", stats.Uptime)
	}

	if stats.AvgUploadTime != "1m0s" {
		t.Errorf("expected AvgUploadTime=1m0s, got %q", stats.AvgUploadTime)
	}
}
