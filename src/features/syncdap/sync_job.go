package syncdap

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/contre95/soulsolid/src/features/jobs"
)

// SyncDapTask implements jobs.Task for syncdap.
type SyncDapTask struct {
	service *Service
}

// NewSyncDapTask creates a new SyncDapTask.
func NewSyncDapTask(service *Service) *SyncDapTask {
	return &SyncDapTask{service: service}
}

// MetadataKeys returns the required metadata keys for a syncdap job.
func (e *SyncDapTask) MetadataKeys() []string {
	return []string{"uuid", "mountPath"}
}

// sanitizePathComponents sanitizes each path component individually while preserving directory structure
func sanitizePathComponents(path string) string {
	components := strings.Split(path, string(filepath.Separator))
	for i, comp := range components {
		components[i] = sanitizeFATFilename(comp)
	}
	return filepath.Join(components...)
}

// sanitizeFATFilename replaces invalid FAT characters with safe alternatives
func sanitizeFATFilename(name string) string {
	// Characters not allowed in FAT: \/:*?"<>|
	replacements := map[rune]string{
		':':  " - ",
		'"':  "'",
		'|':  "-",
		'<':  "(",
		'>':  ")",
		'?':  "",
		'*':  "",
		'\\': "-",
		// Note: '/' is preserved as it's the path separator
	}

	result := strings.Builder{}
	for _, char := range name {
		if replacement, ok := replacements[char]; ok {
			result.WriteString(replacement)
		} else {
			result.WriteRune(char)
		}
	}

	// Ensure filename isn't too long for FAT
	if len(result.String()) > 255 {
		return result.String()[:255]
	}

	return result.String()
}

// createSanitizedCopy creates a temporary copy with FAT-safe filenames
func (e *SyncDapTask) createSanitizedCopy(ctx context.Context, sourceDir, tempDir string) error {
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		// Skip the root directory
		if relPath == "." {
			return nil
		}

		// Sanitize each path component individually while preserving directory structure
		sanitizedPath := sanitizePathComponents(relPath)
		destPath := filepath.Join(tempDir, sanitizedPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		// Copy file
		sourceFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer sourceFile.Close()

		destFile, err := os.Create(destPath)
		if err != nil {
			return err
		}
		defer destFile.Close()

		_, err = io.Copy(destFile, sourceFile)
		return err
	})
}

// Execute runs the sync logic.
func (e *SyncDapTask) Execute(ctx context.Context, job *jobs.Job, progressUpdater func(int, string)) (map[string]any, error) {
	mountPath := job.Metadata["mountPath"].(string)

	// Get library path from config
	cfg := e.service.configManager.Get()
	libraryPath := cfg.LibraryPath
	if libraryPath == "" {
		return nil, fmt.Errorf("library path not configured")
	}

	// Find the device configuration to get sync_path
	var syncPath string
	deviceUUID := job.Metadata["uuid"]
	for _, device := range cfg.Sync.Devices {
		if device.UUID == deviceUUID {
			syncPath = device.SyncPath
			break
		}
	}

	// Default to "Music" if sync_path is not specified
	if syncPath == "" {
		syncPath = "Music"
	}

	// Clean up the mount path to avoid double slashes
	mountPath = strings.TrimRight(mountPath, "/")
	fullSyncPath := mountPath + "/" + syncPath

	// First check if the mount path exists
	checkCmd := exec.CommandContext(ctx, "test", "-d", mountPath)
	if err := checkCmd.Run(); err != nil {
		return nil, fmt.Errorf("device mount path '%s' not found or not accessible - device may not be mounted", mountPath)
	}

	// Create the sync directory if it doesn't exist
	mkdirCmd := exec.CommandContext(ctx, "mkdir", "-p", fullSyncPath)
	if err := mkdirCmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to create sync directory '%s': %w - check permissions", fullSyncPath, err)
	}

	// Create temporary directory for sanitized copy
	tempDir, tempErr := os.MkdirTemp("", "soulsolid-fat-sync-")
	if tempErr != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", tempErr)
	}
	defer os.RemoveAll(tempDir)

	// Create sanitized copy for FAT compatibility
	slog.Info("Creating FAT-compatible copy with sanitized filenames...")
	if copyErr := e.createSanitizedCopy(ctx, libraryPath, tempDir); copyErr != nil {
		slog.Warn("Failed to create sanitized copy, using original filenames", "error", copyErr)
		tempDir = libraryPath // fallback to original
	}

	// Build rsync command with FAT-safe options
	rsyncArgs := []string{
		"-av", "--progress", "--stats", "--delete",
		"--modify-window=1",                      // FAT filesystem compatibility
		"--iconv=utf-8,utf-8",                    // Character encoding handling
		"--no-perms", "--no-owner", "--no-group", // FAT doesn't support Unix permissions
		"--inplace",       // Avoid temp files with special chars
		"--chmod=ugo=rwX", // FAT permissions
		"--max-size=4G",   // FAT32 file size limit
		tempDir + "/",
		fullSyncPath + "/",
	}
	slog.Debug("Executing rsync command", "command", "rsync", "args", rsyncArgs)

	// Log source and destination paths for debugging
	slog.Debug("Rsync paths", "source", tempDir+"/", "destination", fullSyncPath+"/")

	cmd := exec.CommandContext(ctx, "rsync", rsyncArgs...)

	// Capture stderr to parse progress and stats
	stderrPipe, pipeErr := cmd.StderrPipe()
	if pipeErr != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", pipeErr)
	}

	// Start command
	if startErr := cmd.Start(); startErr != nil {
		return nil, fmt.Errorf("failed to start rsync: %w", startErr)
	}

	// Parse rsync output for progress and stats
	var stats SyncStats
	go e.parseRsyncOutput(job, stderrPipe, progressUpdater, &stats)

	// Wait for command to complete
	waitErr := cmd.Wait()

	if waitErr != nil {
		if ctx.Err() == context.Canceled {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("rsync failed: %w", waitErr)
	}

	finalMessage := fmt.Sprintf("Sync completed successfully. %s", stats.Summary)
	job.Logger.Info(finalMessage)

	return map[string]any{"stats": stats, "msg": finalMessage}, nil
}

// Cleanup does nothing for syncdap.
func (e *SyncDapTask) Cleanup(job *jobs.Job) error {
	return nil
}

// SyncStats represents the statistics from a sync operation
type SyncStats struct {
	FilesTransferred int
	FilesTotal       int
	DataTransferred  string
	DataTotal        string
	Summary          string
}

// parseRsyncOutput parses rsync stderr output for progress and stats
func (e *SyncDapTask) parseRsyncOutput(job *jobs.Job, reader io.Reader, progressUpdater func(int, string), stats *SyncStats) {
	scanner := bufio.NewScanner(reader)
	fileRegex := regexp.MustCompile(`^(\S+)$`)
	progressRegex := regexp.MustCompile(`^\s*(\d+)\s+(\d+)%\s+([\d.]+\w+/s).*$`)

	for scanner.Scan() {
		line := scanner.Text()
		lineTrim := strings.TrimSpace(line)

		// Check if it's a stats line (contains ":")
		if strings.Contains(lineTrim, ":") {
			parts := strings.SplitN(lineTrim, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				switch key {
				case "Number of files":
					if val, err := strconv.Atoi(value); err == nil {
						stats.FilesTotal = val
					}
				case "Number of files transferred", "Number of regular files transferred":
					if val, err := strconv.Atoi(value); err == nil {
						stats.FilesTransferred = val
					}
				case "Total file size":
					stats.DataTotal = value
				case "Total transferred file size":
					stats.DataTransferred = value
				}
			}
			continue
		}

		// Check if it's a file transfer line (filename on its own line)
		if matches := fileRegex.FindStringSubmatch(line); len(matches) > 1 && !strings.Contains(line, "%") {
			progressUpdater(job.Progress, matches[1])
		}

		// Check if it's a progress line (contains percentage)
		if strings.Contains(line, "%") {
			if matches := progressRegex.FindStringSubmatch(line); len(matches) > 3 {
				progress, _ := strconv.Atoi(matches[2])
				transferRate := matches[3]
				message := fmt.Sprintf("Transferring at %s", transferRate)
				if job.Message != "" {
					message = fmt.Sprintf("%s (%s)", job.Message, transferRate)
				}

				progressUpdater(progress, message)
			}
		}
	}

	// Build summary after parsing
	if stats.FilesTransferred > 0 {
		stats.Summary = fmt.Sprintf("Transferred %d files (%s)", stats.FilesTransferred, stats.DataTransferred)
	} else {
		stats.Summary = "No files needed to be transferred"
	}
}
