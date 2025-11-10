package services

import (
	"fmt"
	"safelyyou/internal/core/domain"
	coreerrors "safelyyou/internal/core/errors"
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
	if !s.repo.Exists(id) {
		return coreerrors.ErrDeviceNotFound
	}

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

func (s *DeviceServiceImpl) RecordStats(id string, sentAt time.Time, uploadMs int64) error {
	if uploadMs < 0 {
		return fmt.Errorf("upload_time must be >= 0")
	}

	// Enforce that only known devices (from devices.csv) are valid.
	if !s.repo.Exists(id) {
		return coreerrors.ErrDeviceNotFound
	}

	// Only update upload stats;
	return s.repo.WithDevice(id, func(d *domain.DeviceStats) error {
		d.UploadCount++
		d.UploadSumMs += uploadMs // this is actually ns from upload_time, name aside
		return nil
	})
}

func (s *DeviceServiceImpl) GetStats(id string) (*ports.Stats, error) {

	deviceStats, err := s.repo.GetSnapshot(id)
	if err != nil {
		return nil, err
	}

	uptime := deviceStats.UptimePercent()
	avgUpload := deviceStats.AvgUploadDuration()

	return &ports.Stats{
		Uptime:        uptime,
		AvgUploadTime: avgUpload.String(),
	}, nil
}
