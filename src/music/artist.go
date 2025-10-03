package music

import (
	"fmt"
	"strings"
)

// VariousArtistsName is the standard name for compilation albums
const VariousArtistsName = "Various Artists"

// Artist represents a music artist.
type Artist struct {
	ID         string
	Name       string
	SortName   string
	Attributes map[string]string
	// Image URLs from external sources
	ImageSmall  string
	ImageMedium string
	ImageLarge  string
	ImageXL     string
}

// Validate validates the artist fields.
func (a *Artist) Validate() error {
	if strings.TrimSpace(a.Name) == "" {
		return fmt.Errorf("artist name cannot be empty")
	}
	if len(a.Name) > 500 {
		return fmt.Errorf("artist name cannot exceed 500 characters")
	}
	if a.SortName != "" && len(a.SortName) > 500 {
		return fmt.Errorf("artist sort name cannot exceed 500 characters")
	}
	return nil
}
