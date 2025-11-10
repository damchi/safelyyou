package domain

import (
	"math"
	"testing"
	"time"
)

func TestUptimePercent_NoHeartbeatsReturnsZero(t *testing.T) {
	d := &DeviceStats{
		ID:             "device-1",
		HeartbeatCount: 0,
	}

	if result := d.UptimePercent(); result != 0 {
		t.Fatalf("expected 0 uptime when no heartbeats, got %f", result)
	}
}

func TestUptimePercent_BasicWindow(t *testing.T) {
	// 60 heartbeats over 60 minutes -> 60 / 60 * 100 = 100
	t1 := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	t2 := t1.Add(60 * time.Minute)

	d := &DeviceStats{
		ID:             "device-1",
		FirstHeartbeat: t1,
		LastHeartbeat:  t2,
		HeartbeatCount: 60,
	}

	result := d.UptimePercent()
	if math.Abs(result-100.0) > 0.0001 {
		t.Fatalf("expected uptime ≈ 100, got %f", result)
	}
}

func TestUptimePercent_ZeroOrNegativeWindowUsesOneMinute(t *testing.T) {
	// FirstHeartbeat == LastHeartbeat => window.Minutes() == 0
	// We treat that as 1 minute, so:
	// HeartbeatCount=10 => 10 / 1 * 100 = 1000
	t1 := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)

	d := &DeviceStats{
		ID:             "device-1",
		FirstHeartbeat: t1,
		LastHeartbeat:  t1,
		HeartbeatCount: 10,
	}

	result := d.UptimePercent()
	if math.Abs(result-1000.0) > 0.0001 {
		t.Fatalf("expected uptime ≈ 1000, got %f", result)
	}
}

func TestAvgUploadDuration_NoUploadsReturnsZeroDuration(t *testing.T) {
	d := &DeviceStats{
		ID:          "device-1",
		UploadCount: 0,
	}

	if result := d.AvgUploadDuration(); result != 0 {
		t.Fatalf("expected 0 duration when no uploads, got %v", result)
	}
}

func TestAvgUploadDuration_ComputesAverageCorrectly(t *testing.T) {
	// Two uploads: 30s and 90s -> average = 60s => 1m0s
	sum := int64(30*time.Second + 90*time.Second) // in ns

	d := &DeviceStats{
		ID:          "device-1",
		UploadCount: 2,
		UploadSumMs: sum, // or UploadSumNs if you rename
	}

	result := d.AvgUploadDuration()
	expected := 1 * time.Minute

	if result != expected {
		t.Fatalf("expected avg duration %v, got %v", expected, result)
	}
}

func TestAvgUploadDuration_NegativeAverageClampedToZero(t *testing.T) {
	d := &DeviceStats{
		ID:          "device-1",
		UploadCount: 2,
		UploadSumMs: -100, // bogus internal state
	}

	result := d.AvgUploadDuration()
	if result != 0 {
		t.Fatalf("expected negative average to clamp to 0, got %v", result)
	}
}
