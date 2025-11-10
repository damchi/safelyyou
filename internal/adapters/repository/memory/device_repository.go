package memory

import (
	"encoding/csv"
	"os"
	"safelyyou/internal/core/domain"
	coreerrors "safelyyou/internal/core/errors"
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

// WithDevice runs a function while holding a write lock on the device.
// WithDevice finds (or creates) a device by id and executes fn while holding
// a write lock on the underlying map. This lets the service perform
// read-modify-write updates atomically without worrying about concurrency.
func (r *DeviceRepository) WithDevice(id string, fn func(d *domain.DeviceStats) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	d, ok := r.devices[id]
	if !ok {
		// Auto-create device if it doesn't exist yet.
		d = domain.NewDeviceStats(id)
		r.devices[id] = d
	}
	return fn(d)
}

func (r *DeviceRepository) Exists(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.devices[id]
	return ok
}

func (r *DeviceRepository) GetSnapshot(id string) (*domain.DeviceStats, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	deviceStats, ok := r.devices[id]
	if !ok {
		return nil, coreerrors.ErrDeviceNotFound
	}
	deviceCopy := *deviceStats
	return &deviceCopy, nil
}
