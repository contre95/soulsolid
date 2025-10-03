package files

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"soulsolid/src/features/importing"
	"soulsolid/src/music"
)

// FileOrganizer is the infrastructure implementation of the importing.FileOrganizer interface.
type FileOrganizer struct {
	libraryPath string
	pathParser  importing.PathParser
}

// NewFileOrganizer creates a new file organizer implementation.
func NewFileOrganizer(libraryPath string, pathParser importing.PathParser) *FileOrganizer {
	return &FileOrganizer{libraryPath: libraryPath, pathParser: pathParser}
}

// GetLibraryPath generates the library path for a track without moving it.
func (o *FileOrganizer) GetLibraryPath(ctx context.Context, track *music.Track) (string, error) {
	renderedPath, err := o.pathParser.RenderPath(track)
	if err != nil {
		return "", fmt.Errorf("failed to render path: %w", err)
	}

	newPath := filepath.Join(o.libraryPath, renderedPath+filepath.Ext(track.Path))
	return newPath, nil
}

// MoveTrack moves a track to a new location based on its metadata.
func (o *FileOrganizer) MoveTrack(ctx context.Context, track *music.Track) (string, error) {
	renderedPath, err := o.pathParser.RenderPath(track)
	if err != nil {
		return "", fmt.Errorf("failed to render path: %w", err)
	}

	newPath := filepath.Join(o.libraryPath, renderedPath+filepath.Ext(track.Path))
	if err := os.MkdirAll(filepath.Dir(newPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Try to rename first (works within the same filesystem)
	if err := os.Rename(track.Path, newPath); err != nil {
		// If rename fails due to cross-device link, fall back to copy+delete
		if isCrossDeviceError(err) {
			if err := copyFile(track.Path, newPath); err != nil {
				return "", fmt.Errorf("failed to copy file: %w", err)
			}
			// Remove the original file after successful copy
			if err := os.Remove(track.Path); err != nil {
				return "", fmt.Errorf("failed to remove original file after copy: %w", err)
			}
		} else {
			return "", fmt.Errorf("failed to move file: %w", err)
		}
	}
	return newPath, nil
}

// isCrossDeviceError checks if an error is due to cross-device link (moving across filesystems)
func isCrossDeviceError(err error) bool {
	return err != nil && (err.Error() == "invalid cross-device link" || err.Error() == "cross-device link")
}

// CopyTrack copies a track to a new location based on its metadata.
func (o *FileOrganizer) CopyTrack(ctx context.Context, track *music.Track) (string, error) {
	renderedPath, err := o.pathParser.RenderPath(track)
	if err != nil {
		return "", fmt.Errorf("failed to render path: %w", err)
	}

	newPath := filepath.Join(o.libraryPath, renderedPath+filepath.Ext(track.Path))
	if err := os.MkdirAll(filepath.Dir(newPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	if err := copyFile(track.Path, newPath); err != nil {
		return "", fmt.Errorf("failed to copy file: %w", err)
	}
	return newPath, nil
}

// DeleteTrack removes a track file from the library
func (o *FileOrganizer) DeleteTrack(ctx context.Context, trackPath string) error {
	if err := os.Remove(trackPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete track file: %w", err)
	}
	return nil
}

func copyFile(src, dst string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()
	_, err = io.Copy(destination, source)
	return err
}
