package music

import (
	"fmt"
	"strings"
	"time"
)

// Playlist represents a collection of tracks.
type Playlist struct {
	ID           string
	Name         string
	Description  string
	Tracks       []*Track
	CreatedDate  time.Time
	ModifiedDate time.Time
	Attributes   map[string]string
}

// Validate validates the playlist fields.
func (p *Playlist) Validate() error {
	if strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("playlist name cannot be empty")
	}
	if len(p.Name) > 500 {
		return fmt.Errorf("playlist name cannot exceed 500 characters")
	}
	if len(p.Description) > 1000 {
		return fmt.Errorf("playlist description cannot exceed 1000 characters")
	}
	return nil
}

// Pretty returns a formatted string representation of the playlist.
func (p *Playlist) Pretty() string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("%-30s : %s\n", "ID", p.ID))
	builder.WriteString(fmt.Sprintf("%-30s : %s\n", "Name", p.Name))
	builder.WriteString(fmt.Sprintf("%-30s : %s\n", "Description", p.Description))
	builder.WriteString(fmt.Sprintf("%-30s : %d\n", "Track Count", len(p.Tracks)))
	builder.WriteString(fmt.Sprintf("%-30s : %s\n", "Created Date", p.CreatedDate.Format("2006:01:02 15:04:05-07:00")))
	builder.WriteString(fmt.Sprintf("%-30s : %s\n", "Modified Date", p.ModifiedDate.Format("2006:01:02 15:04:05-07:00")))
	if p.Attributes != nil {
		for k, v := range p.Attributes {
			builder.WriteString(fmt.Sprintf("%-30s : %s\n", k, v))
		}
	}
	return builder.String()
}
