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
}

// DeviceRepository is the persistence port used by the service.
type DeviceRepository interface {
	WithDevice(id string, fn func(d *domain.DeviceStats) error) error
}
