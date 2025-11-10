package domain

import "time"

// DeviceStats holds aggregated data per device.
type DeviceStats struct {
	ID             string
	FirstHeartbeat time.Time
	LastHeartbeat  time.Time
	HeartbeatCount int64
	UploadCount    int64
	UploadSumMs    int64
}

// NewDeviceStats creates a new stats struct for a device.
func NewDeviceStats(id string) *DeviceStats {
	return &DeviceStats{ID: id}
}

func (d *DeviceStats) UptimePercent() float64 {
	if d.HeartbeatCount == 0 {
		return 0
	}

	window := d.LastHeartbeat.Sub(d.FirstHeartbeat)
	minutes := window.Minutes()
	// Avoid divide-by-zero: if all heartbeats have the same timestamp,
	// or the window somehow ends up <= 0, treat it as a 1-minute window.
	if minutes <= 0 {
		minutes = 1
	}

	sumHeartbeats := float64(d.HeartbeatCount)
	return (sumHeartbeats / minutes) * 100.0
}

func (d *DeviceStats) AvgUploadDuration() time.Duration {
	if d.UploadCount == 0 {
		return 0
	}

	avgNs := d.UploadSumMs / d.UploadCount
	if avgNs < 0 {
		avgNs = 0
	}
	return time.Duration(avgNs)
}
