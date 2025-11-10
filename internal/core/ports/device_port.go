package ports

import (
	"safelyyou/internal/core/domain"
	"time"
)

type Stats struct {
	Uptime        float64
	AvgUploadTime string
}

// DeviceService is the main port used by the HTTP layer.
type DeviceService interface {
	RecordHeartbeat(id string, sentAt time.Time) error
	RecordStats(id string, sentAt time.Time, uploadTime int64) error
	GetStats(id string) (*Stats, error)
}

// DeviceRepository is the persistence port used by the service.
type DeviceRepository interface {
	WithDevice(id string, fn func(d *domain.DeviceStats) error) error
	Exists(id string) bool
	GetSnapshot(id string) (*domain.DeviceStats, error)
}
