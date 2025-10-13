package music

import (
	"fmt"
	"log/slog"
	"strings"
	"time"
)

type AlbumType string

const (
	AlbumTypeCompilation AlbumType = "compilation"
	AlbumTypeSoundtrack  AlbumType = "soundtrack"
	AlbumTypeEP          AlbumType = "ep"
	AlbumTypeSingle      AlbumType = "single"
	AlbumTypeDefault     AlbumType = "default"
)

// ArtistRole represents the role of an artist on a track or album
type ArtistRole struct {
	Artist *Artist
	Role   string // main, featured, remixer, etc.
}

// Album represents a collection of tracks.
type Album struct {
	ID             string
	Title          string
	Type           AlbumType
	Artists        []ArtistRole
	Tracks         []*Track
	ReleaseDate    time.Time
	ReleaseGroupID string
	Label          string
	CatalogNumber  string
	Country        string
	Status         string
	Barcode        string
	Attributes     map[string]string
	AddedDate      time.Time
	ModifiedDate   time.Time
	// Image URLs from external sources
	ImageSmall  string
	ImageMedium string
	ImageLarge  string
	ImageXL     string
	// Artwork data for embedding
	ArtworkData []byte
	Genre       string
}

// Validate validates the album fields.
func (a *Album) Validate() error {
	if strings.TrimSpace(a.Title) == "" {
		return fmt.Errorf("album title cannot be empty")
	}
	if len(a.Title) > 500 {
		return fmt.Errorf("album title cannot exceed 500 characters")
	}
	if len(a.Artists) == 0 {
		return fmt.Errorf("album must have at least one artist")
	}
	for _, artistRole := range a.Artists {
		if artistRole.Artist == nil {
			return fmt.Errorf("album artist cannot be nil")
		}
		if err := artistRole.Artist.Validate(); err != nil {
			return fmt.Errorf("invalid artist in album: %w", err)
		}
	}
	if a.ReleaseGroupID != "" && len(a.ReleaseGroupID) > 100 {
		return fmt.Errorf("release group ID cannot exceed 100 characters")
	}
	if a.Label != "" && len(a.Label) > 200 {
		return fmt.Errorf("label cannot exceed 200 characters")
	}
	if a.CatalogNumber != "" && len(a.CatalogNumber) > 100 {
		return fmt.Errorf("catalog number cannot exceed 100 characters")
	}
	if a.Country != "" && len(a.Country) > 2 {
		return fmt.Errorf("country code cannot exceed 2 characters")
	}
	if a.Status != "" && len(a.Status) > 50 {
		return fmt.Errorf("status cannot exceed 50 characters")
	}
	if a.Barcode != "" && len(a.Barcode) > 50 {
		return fmt.Errorf("barcode cannot exceed 50 characters")
	}
	if a.Genre != "" && len(a.Genre) > 100 {
		slog.Error("Album validation failed: genre too long", "genre", a.Genre, "length", len(a.Genre))
		return fmt.Errorf("genre cannot exceed 100 characters")
	}
	return nil
}
