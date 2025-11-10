package memory

import (
	"errors"
	"os"
	"testing"
	"time"

	"safelyyou/internal/core/domain"
	coreerrors "safelyyou/internal/core/errors"
)

// -----------------------------------------------------------------------------
// Tests for LoadFromCSV
// -----------------------------------------------------------------------------

func TestLoadFromCSV_Success(t *testing.T) {
	// Create a temporary CSV file with a header and a few device IDs.
	f, err := os.CreateTemp("", "devices-*.csv")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer func(name string) {
		err = os.Remove(name)
		if err != nil {
			t.Fatalf("failed to remove temp file: %v", err)
		}
	}(f.Name())

	content := "device_id\n" +
		"dev-1\n" +
		"dev-2\n" +
		"dev-3\n"

	if _, err = f.WriteString(content); err != nil {
		t.Fatalf("failed to write temp csv: %v", err)
	}
	if err = f.Close(); err != nil {
		t.Fatalf("failed to close temp csv: %v", err)
	}

	repo := NewDeviceRepository()

	if err = repo.LoadFromCSV(f.Name()); err != nil {
		t.Fatalf("LoadFromCSV returned error: %v", err)
	}

	if got := repo.Count(); got != 3 {
		t.Fatalf("expected 3 devices loaded, got %d", got)
	}

	// Verify IDs exist.
	for _, id := range []string{"dev-1", "dev-2", "dev-3"} {
		if !repo.Exists(id) {
			t.Errorf("expected Exists(%q) to be true after LoadFromCSV", id)
		}
	}
}

func TestLoadFromCSV_FileNotFoundReturnsError(t *testing.T) {
	repo := NewDeviceRepository()
	err := repo.LoadFromCSV("does-not-exist.csv")
	if err == nil {
		t.Fatalf("expected error for missing file, got nil")
	}
}

// -----------------------------------------------------------------------------
// Tests for WithDevice
// -----------------------------------------------------------------------------

func TestWithDevice_CreatesAndMutatesDevice(t *testing.T) {
	repo := NewDeviceRepository()
	id := "dev-1"

	// First call should auto-create the device.
	if err := repo.WithDevice(id, func(d *domain.DeviceStats) error {
		if d.ID != id {
			t.Errorf("expected ID=%q, got %q", id, d.ID)
		}
		d.HeartbeatCount++
		return nil
	}); err != nil {
		t.Fatalf("WithDevice returned error on first call: %v", err)
	}

	// Second call mutates the same device.
	if err := repo.WithDevice(id, func(d *domain.DeviceStats) error {
		d.HeartbeatCount++
		return nil
	}); err != nil {
		t.Fatalf("WithDevice returned error on second call: %v", err)
	}

	snap, err := repo.GetSnapshot(id)
	if err != nil {
		t.Fatalf("GetSnapshot returned error: %v", err)
	}

	if snap.HeartbeatCount != 2 {
		t.Errorf("expected HeartbeatCount=2 after two increments, got %d", snap.HeartbeatCount)
	}
}

func TestWithDevice_PropagatesErrorFromCallback(t *testing.T) {
	repo := NewDeviceRepository()
	id := "dev-err"
	wantErr := errors.New("boom")

	err := repo.WithDevice(id, func(d *domain.DeviceStats) error {
		return wantErr
	})

	if !errors.Is(err, wantErr) {
		t.Fatalf("expected WithDevice to return callback error %v, got %v", wantErr, err)
	}
}

// -----------------------------------------------------------------------------
// Tests for Exists
// -----------------------------------------------------------------------------

func TestExists_TrueAfterWithDeviceFalseOtherwise(t *testing.T) {
	repo := NewDeviceRepository()
	id := "dev-1"

	if repo.Exists(id) {
		t.Fatalf("expected Exists(%q) to be false before any creation", id)
	}

	if err := repo.WithDevice(id, func(d *domain.DeviceStats) error {
		return nil
	}); err != nil {
		t.Fatalf("WithDevice returned error: %v", err)
	}

	if !repo.Exists(id) {
		t.Fatalf("expected Exists(%q) to be true after WithDevice", id)
	}
}

// -----------------------------------------------------------------------------
// Tests for GetSnapshot
// -----------------------------------------------------------------------------

func TestGetSnapshot_NotFoundReturnsErrDeviceNotFound(t *testing.T) {
	repo := NewDeviceRepository()

	snap, err := repo.GetSnapshot("missing-id")
	if !errors.Is(err, coreerrors.ErrDeviceNotFound) {
		t.Fatalf("expected ErrDeviceNotFound, got %v", err)
	}
	if snap != nil {
		t.Fatalf("expected snapshot to be nil when error is returned, got %+v", snap)
	}
}

func TestGetSnapshot_ReturnsCopyNotOriginal(t *testing.T) {
	repo := NewDeviceRepository()
	id := "dev-1"

	// Seed a device via WithDevice.
	if err := repo.WithDevice(id, func(d *domain.DeviceStats) error {
		d.HeartbeatCount = 5
		d.FirstHeartbeat = time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
		d.LastHeartbeat = d.FirstHeartbeat.Add(10 * time.Minute)
		return nil
	}); err != nil {
		t.Fatalf("WithDevice returned error: %v", err)
	}

	// Take a snapshot and mutate it.
	snap1, err := repo.GetSnapshot(id)
	if err != nil {
		t.Fatalf("GetSnapshot returned error: %v", err)
	}

	snap1.HeartbeatCount = 999 // mutate the snapshot

	// Take another snapshot; it should not see the mutation.
	snap2, err := repo.GetSnapshot(id)
	if err != nil {
		t.Fatalf("GetSnapshot returned error: %v", err)
	}

	if snap2.HeartbeatCount != 5 {
		t.Fatalf("expected underlying HeartbeatCount to remain 5, got %d", snap2.HeartbeatCount)
	}
}
