package services

import (
	"safelyyou/internal/core/domain"
	"safelyyou/internal/core/ports"
	"time"
)

// DeviceServiceImpl is the default implementation of DeviceService.
type DeviceServiceImpl struct {
	repo ports.DeviceRepository
}

// NewDeviceService constructs a new DeviceServiceImpl.
func NewDeviceService(repo ports.DeviceRepository) *DeviceServiceImpl {
	return &DeviceServiceImpl{repo: repo}
}

// RecordHeartbeat updates heartbeat-related fields for a device.
func (s *DeviceServiceImpl) RecordHeartbeat(id string, sentAt time.Time) error {
	return s.repo.WithDevice(id, func(d *domain.DeviceStats) error {
		// First heartbeat, or out-of-order timestamps (use min/max)
		if d.HeartbeatCount == 0 || sentAt.Before(d.FirstHeartbeat) {
			d.FirstHeartbeat = sentAt
		}
		if d.HeartbeatCount == 0 || sentAt.After(d.LastHeartbeat) {
			d.LastHeartbeat = sentAt
		}
		d.HeartbeatCount++
		return nil
	})
}
