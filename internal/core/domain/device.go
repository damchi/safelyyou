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
