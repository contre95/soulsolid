package syncdap

import (
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/contre95/soulsolid/src/features/config"
	"github.com/contre95/soulsolid/src/features/jobs"
)

// DeviceStatus represents the current status of a sync device
type DeviceStatus struct {
	UUID      string
	Name      string
	Mounted   bool
	MountPath string
	LastSeen  time.Time
	Error     string
}

// Service handles device synchronization monitoring
type Service struct {
	configManager *config.Manager
	jobService    jobs.JobService
	statuses      map[string]DeviceStatus
	mu            sync.RWMutex
	stopChan      chan struct{}
}

// NewService creates a new sync service
func NewService(cfgManager *config.Manager, jobService jobs.JobService) *Service {
	return &Service{
		configManager: cfgManager,
		jobService:    jobService,
		statuses:      make(map[string]DeviceStatus),
		stopChan:      make(chan struct{}),
	}
}

// Start begins monitoring sync devices
func (s *Service) Start() {
	go s.monitorDevices()
}

// Stop halts device monitoring
func (s *Service) Stop() {
	close(s.stopChan)
}

// monitorDevices continuously checks device status
func (s *Service) monitorDevices() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.checkDevices()
		case <-s.stopChan:
			return
		}
	}
}

// checkDevices checks the status of all configured devices
func (s *Service) checkDevices() {
	cfg := s.configManager.Get().Sync

	s.mu.Lock()
	defer s.mu.Unlock()

	// Create a new status map, preserving existing sync progress data
	newStatuses := make(map[string]DeviceStatus)

	for _, device := range cfg.Devices {
		// Check if we have existing status for this device
		var status DeviceStatus
		if existing, exists := s.statuses[device.UUID]; exists {
			status = existing
			status.LastSeen = time.Now()
		} else {
			// New device, create fresh status
			status = DeviceStatus{
				UUID:     device.UUID,
				Name:     device.Name,
				LastSeen: time.Now(),
			}
		}

		// Check mount status
		mounted, mountPath, err := s.isDeviceMounted(device)
		if err != nil {
			slog.Error("Failed to check if device is mounted", "error", err)
			status.Error = err.Error()
		} else {
			status.Mounted = mounted
			status.MountPath = mountPath
		}

		newStatuses[device.UUID] = status

		if status.Mounted {
			slog.Debug("Device mounted", "uuid", device.UUID, "name", device.Name, "path", status.MountPath)
		} else {
			slog.Debug("Device not mounted", "uuid", device.UUID, "name", device.Name)
		}
	}

	// Update the statuses map
	s.statuses = newStatuses
}

// isDeviceMounted checks if a device with the given UUID is mounted
func (s *Service) isDeviceMounted(device config.Device) (bool, string, error) {
	// Check /proc/mounts for the device
	mounts, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return false, "", err
	}

	// First, check if the UUID symlink exists and resolve it to get the actual device path
	uuidPath := filepath.Join("/dev/disk/by-uuid", device.UUID)
	devicePath := ""
	if _, err := os.Lstat(uuidPath); err == nil {
		// Resolve the symlink to get the actual device path
		resolvedPath, err := os.Readlink(uuidPath)
		if err == nil {
			// Convert relative path to absolute path
			if !filepath.IsAbs(resolvedPath) {
				resolvedPath = filepath.Join("/dev/disk/by-uuid", resolvedPath)
			}
			devicePath, err = filepath.EvalSymlinks(resolvedPath)
			if err != nil {
				slog.Warn("Failed to resolve device symlink", "uuid", device.UUID, "error", err)
			}
		}
	}

	lines := strings.SplitSeq(string(mounts), "\n")
	for line := range lines {
		// Check for UUID, device name, or resolved device path in mount line
		if strings.Contains(line, device.UUID) || strings.Contains(line, device.Name) || (devicePath != "" && strings.Contains(line, devicePath)) {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				return true, fields[1], nil
			}
		}
	}

	// Device exists but not mounted
	return false, "", nil
}

// GetStatus returns the current status of all devices
func (s *Service) GetStatus() map[string]DeviceStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]DeviceStatus)
	maps.Copy(result, s.statuses)
	return result
}

// GetDeviceStatus returns the status of a specific device
func (s *Service) GetDeviceStatus(uuid string) (DeviceStatus, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status, exists := s.statuses[uuid]
	return status, exists
}

// findRunningSyncJob finds the job ID of a running sync job for the given UUID
func (s *Service) findRunningSyncJob(uuid string) (string, bool) {
	jobs := s.jobService.GetJobs()
	for _, job := range jobs {
		if job.Type == "dap_sync" && job.Status == "running" {
			if job.Metadata != nil {
				if jobUUID, ok := job.Metadata["uuid"].(string); ok && jobUUID == uuid {
					return job.ID, true
				}
			}
		}
	}
	return "", false
}

// StartSync starts a sync operation for a device
func (s *Service) StartSync(uuid string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	status, exists := s.statuses[uuid]
	if !exists {
		slog.Error("Device not found", "uuid", uuid)
		return "", fmt.Errorf("device not found")
	}

	if !status.Mounted {
		slog.Error("Device not mounted", "uuid", uuid)
		return "", fmt.Errorf("device not mounted")
	}

	// Check if sync already in progress
	if jobID, running := s.findRunningSyncJob(uuid); running {
		slog.Error("Sync already in progress", "uuid", uuid, "jobID", jobID)
		return "", fmt.Errorf("sync already in progress")
	}

	jobID, err := s.jobService.StartJob("dap_sync", "Sync DAP", map[string]any{
		"uuid":      uuid,
		"mountPath": status.MountPath,
	})
	if err != nil {
		slog.Error("Failed to start sync job", "uuid", uuid, "error", err)
		return "", fmt.Errorf("failed to start sync job: %w", err)
	}

	return jobID, nil
}

// CancelSync cancels an ongoing sync operation
func (s *Service) CancelSync(uuid string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	jobID, running := s.findRunningSyncJob(uuid)
	if !running {
		return fmt.Errorf("no sync in progress")
	}

	err := s.jobService.CancelJob(jobID)
	if err != nil {
		return fmt.Errorf("failed to cancel sync job: %w", err)
	}

	return nil
}
