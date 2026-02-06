package playback

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"os"

	"github.com/contre95/soulsolid/src/music"
)

// Service provides playback functionality for audio previews
type Service struct {
	library music.Library
}

// NewService creates a new playback service
func NewService(library music.Library) *Service {
	return &Service{
		library: library,
	}
}

// PreviewTrackReader wraps a file reader to limit reading to 30 seconds
type PreviewTrackReader struct {
	file      *os.File
	remaining int64
}

func (r *PreviewTrackReader) Read(p []byte) (n int, err error) {
	if r.remaining <= 0 {
		slog.Debug("Preview reader EOF reached", "remaining", r.remaining)
		// Auto-close when done
		r.file.Close()
		return 0, io.EOF
	}
	if int64(len(p)) > r.remaining {
		p = p[:r.remaining]
	}
	n, err = r.file.Read(p)
	r.remaining -= int64(n)

	// Auto-close on EOF
	if err == io.EOF {
		r.file.Close()
	}

	// Log first few reads for debugging
	if r.remaining == int64(r.remaining) && n > 0 { // This is approximate for first read check
		slog.Debug("Preview reader reading", "bytesRequested", len(p), "bytesRead", n, "remainingAfter", r.remaining)
	}

	if err != nil && err != io.EOF {
		slog.Error("Error reading from preview reader", "error", err, "remaining", r.remaining)
		r.file.Close() // Close on error too
	}

	return n, err
}

func (r *PreviewTrackReader) Close() error {
	return r.file.Close()
}

// GetTrackPreview returns a reader for a 30-second random preview of the track
func (s *Service) GetTrackPreview(ctx context.Context, trackID string) (io.ReadCloser, string, error) {
	slog.Debug("GetTrackPreview service called", "trackID", trackID)

	// Get track info
	track, err := s.library.GetTrack(ctx, trackID)
	if err != nil {
		slog.Error("Failed to get track", "trackID", trackID, "error", err)
		return nil, "", fmt.Errorf("failed to get track: %w", err)
	}
	if track == nil {
		slog.Error("Track not found", "trackID", trackID)
		return nil, "", fmt.Errorf("track not found")
	}

	// Open the audio file
	slog.Debug("Attempting to open audio file", "path", track.Path)
	file, err := os.Open(track.Path)
	if err != nil {
		slog.Error("Failed to open track file", "path", track.Path, "error", err)
		return nil, "", fmt.Errorf("failed to open track file: %w", err)
	}
	slog.Debug("Successfully opened audio file", "path", track.Path)

	// Get file info to determine file size
	fileInfo, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, "", fmt.Errorf("failed to get file info: %w", err)
	}
	fileSize := fileInfo.Size()

	// If we have duration information, calculate a random starting point
	// For now, we'll use a simple approach: start at a random position up to 1/3 of the file
	// and read for approximately 30 seconds worth of data
	var seekOffset int64 = 0
	var previewSize int64 = fileSize // fallback to full file

	if track.Metadata.Duration > 0 {
		// Calculate bytes per second
		bytesPerSecond := fileSize / int64(track.Metadata.Duration)

		// Random start time: up to max(30 seconds, or 1/3 of track duration)
		maxStartSeconds := track.Metadata.Duration / 3
		if maxStartSeconds < 30 {
			maxStartSeconds = track.Metadata.Duration
		}
		if maxStartSeconds > 0 {
			maxStartSeconds-- // ensure we don't start at the very end
		}

		startSeconds := rand.Intn(int(maxStartSeconds) + 1)
		seekOffset = int64(startSeconds) * bytesPerSecond

		// Calculate 30 seconds worth of data
		previewSize = 30 * bytesPerSecond

		// Make sure we don't go beyond file size
		if seekOffset+previewSize > fileSize {
			if fileSize > seekOffset {
				previewSize = fileSize - seekOffset
			} else {
				seekOffset = 0 // start from beginning if we can't seek to our desired position
				previewSize = fileSize
			}
		}

		slog.Debug("Track preview calculated", "duration", track.Metadata.Duration, "startSeconds", startSeconds, "previewSize", previewSize, "fileSize", fileSize)
	}

	// Seek to the calculated offset
	if seekOffset > 0 {
		_, err = file.Seek(seekOffset, io.SeekStart)
		if err != nil {
			slog.Warn("Failed to seek to random position, starting from beginning", "offset", seekOffset, "error", err)
			file.Seek(0, io.SeekStart) // reset to beginning
			previewSize = fileSize
		}
	}

	// Return the preview reader
	return &PreviewTrackReader{
		file:      file,
		remaining: previewSize,
	}, track.Format, nil
}
