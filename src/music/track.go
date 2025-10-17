package music

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Track represents a single audio file.
type Track struct {
	ID                     string
	Path                   string
	Title                  string
	TitleVersion           string // Version info (remix, live, etc.)
	Artists                []ArtistRole
	Album                  *Album
	Metadata               Metadata
	ISRC                   string
	ChromaprintFingerprint string
	Bitrate                int
	Format                 string
	SampleRate             int
	BitDepth               int
	Channels               int
	ExplicitContent        bool
	Attributes             map[string]string
	PreviewURL             string // URL for 30-second preview
	AddedDate              time.Time
	ModifiedDate           time.Time
}

// Validate validates the track fields.
func (t *Track) Validate() error {
	if strings.TrimSpace(t.Title) == "" {
		err := fmt.Errorf("track title cannot be empty")
		return err
	}
	if len(t.Title) > 500 {
		err := fmt.Errorf("title cannot exceed 500 characters, got %d: title -> %s", len(t.Title), t.Title)
		return err
	}
	// Trim leading quotes from title
	t.Title = strings.Trim(t.Title, "'\"")
	if strings.HasPrefix(t.Title, "'") || strings.HasPrefix(t.Title, "\"") {
		err := fmt.Errorf("title cannot start with a quote: title -> %s", t.Title)
		return err
	}
	if strings.TrimSpace(t.Path) == "" {
		err := fmt.Errorf("track path cannot be empty")
		return err
	}
	if len(t.Path) > 1000 {
		err := fmt.Errorf("track path cannot exceed 1000 characters, got %d: path -> %s", len(t.Path), t.Path)
		return err
	}
	if len(t.Artists) == 0 {
		err := fmt.Errorf("track must have at least one artist: title -> %s", t.Title)
		return err
	}
	for i, artistRole := range t.Artists {
		if artistRole.Artist == nil {
			err := fmt.Errorf("track artist at index %d cannot be nil", i)
			return err
		}
		if err := artistRole.Artist.Validate(); err != nil {
			err = fmt.Errorf("invalid artist in track: %w", err)
			return err
		}
	}
	if t.Album == nil {
		err := fmt.Errorf("track album cannot be nil")
		return err
	}
	if err := t.Album.Validate(); err != nil {
		err = fmt.Errorf("invalid album in track: %w", err)
		return err
	}

	if t.ISRC != "" && len(t.ISRC) > 12 {
		err := fmt.Errorf("ISRC cannot exceed 12 characters, got %d: isrc -> %s", len(t.ISRC), t.ISRC)
		return err
	}
	if t.Format != "" && len(t.Format) > 50 {
		err := fmt.Errorf("format cannot exceed 50 characters, got %d: format -> %s", len(t.Format), t.Format)
		return err
	}
	if t.PreviewURL != "" && len(t.PreviewURL) > 500 {
		err := fmt.Errorf("preview URL cannot exceed 500 characters, got %d: preview_url -> %s", len(t.PreviewURL), t.PreviewURL)
		return err
	}
	// Validate metadata
	if t.Metadata.Duration < 0 {
		err := fmt.Errorf("duration cannot be negative, got %d", t.Metadata.Duration)
		return err
	}
	if t.Metadata.TrackNumber < 0 {
		err := fmt.Errorf("track number cannot be negative, got %d", t.Metadata.TrackNumber)
		return err
	}
	if t.Metadata.DiscNumber < 0 {
		err := fmt.Errorf("disc number cannot be negative, got %d", t.Metadata.DiscNumber)
		return err
	}
	if t.Metadata.Year < 0 {
		err := fmt.Errorf("year cannot be negative, got %d", t.Metadata.Year)
		return err
	}
	if t.Metadata.OriginalYear < 0 {
		err := fmt.Errorf("original year cannot be negative, got %d", t.Metadata.OriginalYear)
		return err
	}
	if t.Metadata.BPM < 0 {
		err := fmt.Errorf("BPM cannot be negative, got %f", t.Metadata.BPM)
		return err
	}
	if t.Metadata.Genre != "" && len(t.Metadata.Genre) > 100 {
		t.Metadata.Genre = t.Metadata.Genre[:100]
	}
	if t.Metadata.Composer != "" && len(t.Metadata.Composer) > 500 {
		err := fmt.Errorf("composer cannot exceed 500 characters, got %d: composer -> %s", len(t.Metadata.Composer), t.Metadata.Composer)
		return err
	}
	if t.Metadata.Lyrics != "" && len(t.Metadata.Lyrics) > 15000 {
		err := fmt.Errorf("lyrics cannot exceed 15000 characters, got %d", len(t.Metadata.Lyrics))
		return err
	}
	return nil
}

func (t *Track) Pretty() {
	fmt.Printf("%-30s : %s\n", "ID", t.ID)
	fmt.Printf("%-30s : %s\n", "Path", t.Path)
	fmt.Printf("%-30s : %s\n", "Title", t.Title)
	if t.TitleVersion != "" {
		fmt.Printf("%-30s : %s\n", "Title Version", t.TitleVersion)
	}
	var artistNames []string
	for _, ar := range t.Artists {
		artistNames = append(artistNames, ar.Artist.Name)
	}
	fmt.Printf("%-30s : %d\n", "Artwork", len(t.Album.ArtworkData))
	fmt.Printf("%-30s : %s\n", "Artist", strings.Join(artistNames, ", "))
	if t.Album != nil {
		fmt.Printf("%-30s : %s\n", "Album", t.Album.Title)
	}
	if t.Metadata.Composer != "" {
		fmt.Printf("%-30s : %s\n", "Composer", t.Metadata.Composer)
	}
	fmt.Printf("%-30s : %s\n", "Genre", t.Metadata.Genre)
	fmt.Printf("%-30s : %d\n", "Year", t.Metadata.Year)
	fmt.Printf("%-30s : %d\n", "Duration", t.Metadata.Duration)
	if t.Metadata.OriginalYear != 0 {
		fmt.Printf("%-30s : %d\n", "Original Year", t.Metadata.OriginalYear)
	}
	fmt.Printf("%-30s : %d\n", "Disc Number", t.Metadata.DiscNumber)
	fmt.Printf("%-30s : %d\n", "Track Number", t.Metadata.TrackNumber)
	if t.Metadata.Lyrics != "" {
		lyrics := t.Metadata.Lyrics
		if len(lyrics) > 100 {
			lyrics = lyrics[:100] + "..."
		}
		fmt.Printf("%-30s : %s\n", "Lyrics", lyrics)
	}
	fmt.Printf("%-30s : %t\n", "Explicit Lyrics", t.Metadata.ExplicitLyrics)
	fmt.Printf("%-30s : %.1f\n", "BPM", t.Metadata.BPM)
	fmt.Printf("%-30s : %.1f\n", "Gain", t.Metadata.Gain)
	fmt.Printf("%-30s : %s\n", "ISRC", t.ISRC)
	fmt.Printf("%-30s : %s\n", "Chromaprint Fingerprint", t.ChromaprintFingerprint)
	fmt.Printf("%-30s : %d\n", "Bitrate", t.Bitrate)
	fmt.Printf("%-30s : %s\n", "Format", t.Format)
	fmt.Printf("%-30s : %d\n", "Sample Rate", t.SampleRate)
	fmt.Printf("%-30s : %d\n", "Bit Depth", t.BitDepth)
	fmt.Printf("%-30s : %d\n", "Channels", t.Channels)
	fmt.Printf("%-30s : %t\n", "Explicit Content", t.ExplicitContent)
	for k, v := range t.Attributes {
		fmt.Printf("%-30s : %s\n", k, v)
	}
	if t.PreviewURL != "" {
		fmt.Printf("%-30s : %s\n", "Preview URL", t.PreviewURL)
	}
	fmt.Printf("%-30s : %s\n", "Added Date", t.AddedDate.Format("2006:01:02 15:04:05-07:00"))
	fmt.Printf("%-30s : %s\n", "Modified Date", t.ModifiedDate.Format("2006:01:02 15:04:05-07:00"))
}

// EnsureMetadataDefaults adds fallback values for missing metadata fields
func (t *Track) EnsureMetadataDefaults() {
	// Fallback for missing artist
	if len(t.Artists) == 0 || t.Artists[0].Artist.Name == "" {
		t.Artists = []ArtistRole{{
			Artist: &Artist{Name: "Unknown Artist"},
			Role:   "main",
		}}
	}
	// Fallback for missing album
	if t.Album == nil || t.Album.Title == "" {
		t.Album = &Album{Title: "Unknown Album"}
	}
	// Fallback for missing year
	if t.Metadata.Year == 0 {
		t.Metadata.Year = 0000
	}
	// Fallback for missing genre
	if t.Metadata.Genre == "" {
		t.Metadata.Genre = "Unknown"
	}
}

// ValidateRequiredMetadata checks for required metadata fields and returns an error if any are missing
func (t *Track) ValidateRequiredMetadata() error {
	var missingFields []string
	if len(t.Artists) == 0 || t.Artists[0].Artist.Name == "" {
		missingFields = append(missingFields, "Artist")
	}
	if t.Album == nil || t.Album.Title == "" {
		missingFields = append(missingFields, "Album")
	}
	if t.Metadata.Year == 0 {
		missingFields = append(missingFields, "Year")
	}
	if len(missingFields) > 0 {
		return fmt.Errorf("missing required metadata fields: %s", strings.Join(missingFields, ", "))
	}
	return nil
}

// GenerateTrackID creates a deterministic UUID for a track from its fingerprint
func GenerateTrackID(fingerprint string) string {
	inputBytes := []byte(fingerprint)
	return uuid.NewSHA1(uuid.NameSpaceDNS, inputBytes).String()
}

type Metadata struct {
	Composer       string
	Genre          string
	Year           int
	Duration       int
	OriginalYear   int
	DiscNumber     int
	TrackNumber    int
	Lyrics         string
	ExplicitLyrics bool
	BPM            float64
	Gain           float64
}
