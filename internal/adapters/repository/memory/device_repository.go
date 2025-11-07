package memory

import (
	"encoding/csv"
	"os"
	"safelyyou/internal/core/domain"
	"sync"
)

type DeviceRepository struct {
	mu      sync.RWMutex
	devices map[string]*domain.DeviceStats
}

// NewDeviceRepository creates an empty in-memory DeviceRepository.
func NewDeviceRepository() *DeviceRepository {
	return &DeviceRepository{
		devices: make(map[string]*domain.DeviceStats),
	}
}

// LoadFromCSV initializes the repository from a CSV file.
// Expected format: header line with "device_id", then one ID per line.
func (r *DeviceRepository) LoadFromCSV(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {

		}
	}(f)

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return err
	}

	for i, row := range records {
		if i == 0 {
			// skip header
			continue
		}
		if len(row) == 0 {
			continue
		}
		id := row[0]
		r.addDevice(id)
	}
	return nil
}

func (r *DeviceRepository) addDevice(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.devices[id]; !exists {
		r.devices[id] = domain.NewDeviceStats(id)
	}
}

func (r *DeviceRepository) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.devices)
}
