package music

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Playlist represents a collection of tracks that can be played in order.
type Playlist struct {
	ID           string
	Name         string
	Description  string
	Tracks       []*Track
	CreatedDate  time.Time
	ModifiedDate time.Time
}

// TotalDuration returns the total duration of all tracks in the playlist in seconds.
func (p *Playlist) TotalDuration() int {
	total := 0
	for _, track := range p.Tracks {
		total += track.Metadata.Duration
	}
	return total
}

// Validate validates the playlist fields.
func (p *Playlist) Validate() error {
	if strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("playlist name cannot be empty")
	}
	if len(p.Name) > 200 {
		return fmt.Errorf("playlist name cannot exceed 200 characters, got %d: name -> %s", len(p.Name), p.Name)
	}
	if len(p.Description) > 1000 {
		return fmt.Errorf("playlist description cannot exceed 1000 characters, got %d", len(p.Description))
	}
	// Validate tracks if present
	for i, track := range p.Tracks {
		if track == nil {
			return fmt.Errorf("playlist track at index %d cannot be nil", i)
		}
		if err := track.Validate(); err != nil {
			return fmt.Errorf("invalid track in playlist: %w", err)
		}
	}
	return nil
}

// Pretty returns a formatted string representation of the playlist for logging/debugging.
func (p *Playlist) Pretty() string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("%-30s : %s\n", "ID", p.ID))
	builder.WriteString(fmt.Sprintf("%-30s : %s\n", "Name", p.Name))
	if p.Description != "" {
		builder.WriteString(fmt.Sprintf("%-30s : %s\n", "Description", p.Description))
	}
	builder.WriteString(fmt.Sprintf("%-30s : %d\n", "Track Count", len(p.Tracks)))
	if len(p.Tracks) > 0 {
		builder.WriteString("Tracks:\n")
		for i, track := range p.Tracks {
			builder.WriteString(fmt.Sprintf("  %d. %s - %s\n", i+1, track.Title, getArtistsString(track.Artists)))
		}
	}
	builder.WriteString(fmt.Sprintf("%-30s : %s\n", "Created Date", p.CreatedDate.Format("2006-01-02 15:04:05-07:00")))
	builder.WriteString(fmt.Sprintf("%-30s : %s\n", "Modified Date", p.ModifiedDate.Format("2006-01-02 15:04:05-07:00")))
	return builder.String()
}

// AddTrack adds a track to the playlist if it's not already present.
func (p *Playlist) AddTrack(track *Track) error {
	if track == nil {
		return fmt.Errorf("cannot add nil track to playlist")
	}
	// Check if track is already in playlist
	for _, existingTrack := range p.Tracks {
		if existingTrack.ID == track.ID {
			return fmt.Errorf("track %s is already in playlist", track.Title)
		}
	}
	p.Tracks = append(p.Tracks, track)
	p.ModifiedDate = time.Now()
	return nil
}

// RemoveTrack removes a track from the playlist by ID.
func (p *Playlist) RemoveTrack(trackID string) error {
	for i, track := range p.Tracks {
		if track.ID == trackID {
			p.Tracks = append(p.Tracks[:i], p.Tracks[i+1:]...)
			p.ModifiedDate = time.Now()
			return nil
		}
	}
	return fmt.Errorf("track with ID %s not found in playlist", trackID)
}

// ContainsTrack checks if a track is in the playlist.
func (p *Playlist) ContainsTrack(trackID string) bool {
	for _, track := range p.Tracks {
		if track.ID == trackID {
			return true
		}
	}
	return false
}

// PlaylistRepository defines the interface for playlist data access operations.
type PlaylistRepository interface {
	Create(ctx context.Context, playlist *Playlist) error
	GetByID(ctx context.Context, id string) (*Playlist, error)
	GetAll(ctx context.Context) ([]*Playlist, error)
	Update(ctx context.Context, playlist *Playlist) error
	Delete(ctx context.Context, id string) error
	AddTrackToPlaylist(ctx context.Context, playlistID, trackID string) error
	RemoveTrackFromPlaylist(ctx context.Context, playlistID, trackID string) error
	GetTracksForPlaylist(ctx context.Context, playlistID string) ([]*Track, error)
}

// GeneratePlaylistID creates a UUID for a playlist.
func GeneratePlaylistID() string {
	return uuid.New().String()
}

// Helper function to get artist string from ArtistRole slice
func getArtistsString(artists []ArtistRole) string {
	var names []string
	for _, ar := range artists {
		if ar.Artist != nil {
			names = append(names, ar.Artist.Name)
		}
	}
	return strings.Join(names, ", ")
}
