package services

import (
	coreerrors "safelyyou/internal/core/errors"
	"testing"
	"time"

	"safelyyou/internal/core/domain"
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

func (r *fakeDeviceRepo) WithDevice(id string, fn func(d *domain.DeviceStats) error) error {
	d, ok := r.devices[id]
	if !ok {
		d = domain.NewDeviceStats(id)
		r.devices[id] = d
	}
	return fn(d)
}

func (r *fakeDeviceRepo) GetSnapshot(id string) (*domain.DeviceStats, error) {
	d, ok := r.devices[id]
	if !ok {
		return nil, coreerrors.ErrDeviceNotFound
	}
	copy := *d
	return &copy, nil
}

// --- Tests for RecordHeartbeat -----------------------------------------------

func TestRecordHeartbeat_FirstHeartbeatInitializesStats(t *testing.T) {
	repo := newFakeDeviceRepo()
	svc := NewDeviceService(repo)

	id := "device-123"
	t1 := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)

	if err := svc.RecordHeartbeat(id, t1); err != nil {
		t.Fatalf("RecordHeartbeat returned error: %v", err)
	}

	got, err := repo.GetSnapshot(id)
	if err != nil {
		t.Fatalf("GetSnapshot returned error: %v", err)
	}

	if got.ID != id {
		t.Errorf("expected ID %q, got %q", id, got.ID)
	}
	if got.HeartbeatCount != 1 {
		t.Errorf("expected HeartbeatCount=1, got %d", got.HeartbeatCount)
	}
	if !got.FirstHeartbeat.Equal(t1) {
		t.Errorf("expected FirstHeartbeat=%v, got %v", t1, got.FirstHeartbeat)
	}
	if !got.LastHeartbeat.Equal(t1) {
		t.Errorf("expected LastHeartbeat=%v, got %v", t1, got.LastHeartbeat)
	}
}

func TestRecordHeartbeat_MultipleHeartbeatsUpdatesFirstAndLast(t *testing.T) {
	repo := newFakeDeviceRepo()
	svc := NewDeviceService(repo)

	id := "device-123"
	t1 := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	t2 := t1.Add(5 * time.Minute)
	t3 := t2.Add(10 * time.Minute)

	// three heartbeats in order
	_ = svc.RecordHeartbeat(id, t1)
	_ = svc.RecordHeartbeat(id, t2)
	if err := svc.RecordHeartbeat(id, t3); err != nil {
		t.Fatalf("RecordHeartbeat returned error: %v", err)
	}

	got, err := repo.GetSnapshot(id)
	if err != nil {
		t.Fatalf("GetSnapshot returned error: %v", err)
	}

	if got.HeartbeatCount != 3 {
		t.Errorf("expected HeartbeatCount=3, got %d", got.HeartbeatCount)
	}
	if !got.FirstHeartbeat.Equal(t1) {
		t.Errorf("expected FirstHeartbeat=%v, got %v", t1, got.FirstHeartbeat)
	}
	if !got.LastHeartbeat.Equal(t3) {
		t.Errorf("expected LastHeartbeat=%v, got %v", t3, got.LastHeartbeat)
	}
}

// corner case: heartbeat arrives "out of order" earlier than existing first
func TestRecordHeartbeat_OutOfOrderEarlierUpdatesFirstKeepsLast(t *testing.T) {
	repo := newFakeDeviceRepo()
	svc := NewDeviceService(repo)

	id := "device-123"
	tMiddle := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	tLate := tMiddle.Add(30 * time.Minute)
	tEarly := tMiddle.Add(-1 * time.Hour)

	_ = svc.RecordHeartbeat(id, tMiddle)
	_ = svc.RecordHeartbeat(id, tLate)

	// now an earlier heartbeat arrives
	if err := svc.RecordHeartbeat(id, tEarly); err != nil {
		t.Fatalf("RecordHeartbeat returned error: %v", err)
	}

	got, err := repo.GetSnapshot(id)
	if err != nil {
		t.Fatalf("GetSnapshot returned error: %v", err)
	}

	if got.HeartbeatCount != 3 {
		t.Errorf("expected HeartbeatCount=3, got %d", got.HeartbeatCount)
	}
	if !got.FirstHeartbeat.Equal(tEarly) {
		t.Errorf("expected FirstHeartbeat=%v, got %v", tEarly, got.FirstHeartbeat)
	}
	if !got.LastHeartbeat.Equal(tLate) {
		t.Errorf("expected LastHeartbeat=%v, got %v", tLate, got.LastHeartbeat)
	}
}
