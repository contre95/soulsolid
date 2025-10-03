package tag

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/contre95/soulsolid/src/features/importing"
	"github.com/contre95/soulsolid/src/music"
	"github.com/dhowden/tag"
)

// TagReader is an implementation of the MetadataReader interface that uses the dhowden/tag library.
type TagReader struct{}

// NewTagReader creates a new TagReader
func NewTagReader() importing.TagReader {
	return &TagReader{}
}

// parseArtists parses a string containing multiple artists separated by common delimiters
func parseArtists(artistString string) []*music.Artist {
	if strings.TrimSpace(artistString) == "" {
		return nil
	}

	// Common delimiters: semicolon, slash, comma, "feat.", "ft.", "&"
	delimiters := []string{";", "/", ",", " feat. ", " ft. ", " & "}

	// Try each delimiter
	for _, delim := range delimiters {
		if strings.Contains(artistString, delim) {
			names := strings.Split(artistString, delim)
			artists := make([]*music.Artist, 0, len(names))
			for _, name := range names {
				name = strings.TrimSpace(name)
				if name != "" {
					artists = append(artists, &music.Artist{Name: name})
				}
			}
			if len(artists) > 0 {
				return artists
			}
		}
	}

	// If no delimiters found, treat as single artist
	return []*music.Artist{{Name: strings.TrimSpace(artistString)}}
}

// Read reads metadata from a music file.
func (r *TagReader) ReadFileTags(ctx context.Context, filePath string) (*music.Track, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	tags, err := tag.ReadFrom(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read tags: %w", err)
	}

	trackNumber, _ := tags.Track()
	discNumber, _ := tags.Disc()

	// Get album artist, fall back to track artist if empty
	albumArtist := tags.AlbumArtist()
	if albumArtist == "" {
		albumArtist = tags.Artist()
	}

	// Parse multiple artists for track and album
	trackArtists := parseArtists(tags.Artist())
	albumArtists := parseArtists(albumArtist)

	track := &music.Track{
		Path:  filePath,
		Title: tags.Title(),
		Album: &music.Album{
			Title:   tags.Album(),
			Artists: make([]music.ArtistRole, 0, len(albumArtists)),
		},
		Artists: make([]music.ArtistRole, 0, len(trackArtists)),
		Metadata: music.Metadata{
			Year:        tags.Year(),
			Genre:       tags.Genre(),
			TrackNumber: trackNumber,
			DiscNumber:  discNumber,
			Composer:    tags.Composer(),
			Lyrics:      tags.Lyrics(),
		},
	}

	// Add track artists with "main" role
	for _, artist := range trackArtists {
		track.Artists = append(track.Artists, music.ArtistRole{
			Artist: artist,
			Role:   "main",
		})
	}

	// Add album artists with "main" role
	for _, artist := range albumArtists {
		track.Album.Artists = append(track.Album.Artists, music.ArtistRole{
			Artist: artist,
			Role:   "main",
		})
	}

	// Set format from file extension
	ext := strings.ToLower(filepath.Ext(filePath))
	track.Format = strings.TrimPrefix(ext, ".")

	// Try to read additional metadata from raw tags
	r.readAdditionalMetadata(tags, track)

	return track, nil
}

// readAdditionalMetadata attempts to read additional metadata fields from tags
func (r *TagReader) readAdditionalMetadata(tags tag.Metadata, track *music.Track) {
	// Try to read ISRC from various tag fields
	if isrc := r.findISRC(tags); isrc != "" {
		track.ISRC = isrc
	}

	// Try to read lyrics from tags
	if lyrics := r.readLyrics(tags); lyrics != "" {
		fmt.Printf("DEBUG: Found lyrics in file %s: %s\n", track.Path, lyrics)
		track.Metadata.Lyrics = lyrics
	} else {
		fmt.Printf("DEBUG: No lyrics found in file %s\n", track.Path)
	}

	// Try to extract basic audio properties
	r.extractAudioProperties(track)
}

// readLyrics attempts to read lyrics from various tag fields
func (r *TagReader) readLyrics(tags tag.Metadata) string {
	// Try to read from raw tags for lyric fields
	if rawTags := tags.Raw(); rawTags != nil {
		fmt.Printf("DEBUG: Available raw tags: %v\n", getTagKeys(rawTags))
		// Check for common lyric field names in different formats
		lyricFields := []string{"LYRICS", "UNSYNCEDLYRICS", "USLT", "USLT0", "USLT1", "Lyrics", "UnsyncedLyrics"}
		for _, field := range lyricFields {
			if value := rawTags[field]; value != nil {
				fmt.Printf("DEBUG: Found lyric field %s\n", field)
				if str, ok := value.(string); ok && str != "" {
					return str
				}
				// Handle byte slices
				if bytes, ok := value.([]byte); ok && len(bytes) > 0 {
					return string(bytes)
				}
			}
		}
	}

	return ""
}

// getTagKeys returns a slice of all tag field names for debugging
func getTagKeys(rawTags map[string]interface{}) []string {
	keys := make([]string, 0, len(rawTags))
	for k := range rawTags {
		keys = append(keys, k)
	}
	return keys
}

// extractAudioProperties attempts to extract audio properties from the file
func (r *TagReader) extractAudioProperties(track *music.Track) {
	// For FLAC files, we can make some reasonable assumptions and estimates
	if track.Format == "flac" {
		// FLAC files are typically 44.1kHz, 16-bit, stereo
		track.SampleRate = 44100
		track.BitDepth = 16
		track.Channels = 2

		// Calculate bitrate and duration based on file size
		if fileInfo, err := os.Stat(track.Path); err == nil {
			fileSizeBytes := fileInfo.Size()
			fileSizeBits := fileSizeBytes * 8

			// For FLAC, typical bitrates are 700-1200 kbps
			// Let's estimate based on common FLAC compression ratios
			// CD quality uncompressed: 44.1kHz * 16-bit * 2 channels = 1,411,200 bps = 1411 kbps
			// FLAC compression ratio is typically 0.6-0.8, so ~850-1130 kbps
			estimatedBitrate := 1000 // kbps - reasonable average for FLAC

			track.Bitrate = estimatedBitrate

			// Calculate duration: (file_size_bits) / (bitrate * 1000) = seconds
			calculatedDuration := int(fileSizeBits / int64(estimatedBitrate*1000))
			track.Metadata.Duration = calculatedDuration
		}
	}
}

// findISRC attempts to find ISRC in various tag fields
func (r *TagReader) findISRC(tags tag.Metadata) string {
	rawTags := tags.Raw()
	if rawTags != nil {
		// Debug: print all available tag fields that might contain ISRC
		for key, value := range rawTags {
			if strings.Contains(strings.ToUpper(key), "ISRC") || strings.Contains(strings.ToUpper(key), "TSRC") {
				if strValue, ok := value.(string); ok && strValue != "" {
					fmt.Printf("DEBUG: Found potential ISRC field %s: %s (length: %d)\n", key, strValue, len(strValue))
				}
			}
		}
	}

	// Try common ISRC field names (both uppercase and lowercase)
	isrcFields := []string{"ISRC", "isrc", "TSRC", "tsrc", "ISRC1", "isrc1", "ISRC2", "isrc2"}

	for _, field := range isrcFields {
		if value, ok := rawTags[field]; ok {
			if strValue, ok := value.(string); ok && strValue != "" {
				fmt.Printf("DEBUG: Processing ISRC field %s: %s\n", field, strValue)
				// Handle multiple ISRCs separated by "/" - take only the first one
				if strings.Contains(strValue, "/") {
					parts := strings.Split(strValue, "/")
					if len(parts) > 0 {
						firstISRC := strings.TrimSpace(parts[0])
						if firstISRC != "" {
							fmt.Printf("DEBUG: Returning first ISRC from slash-separated: %s\n", firstISRC)
							return firstISRC
						}
					}
				}
				// Handle concatenated ISRCs without separators (take first 12 chars if multiple of 12)
				strValue = strings.TrimSpace(strValue)
				if len(strValue) > 12 && len(strValue)%12 == 0 {
					result := strValue[:12]
					fmt.Printf("DEBUG: Returning first 12 chars of concatenated ISRC: %s\n", result)
					return result
				}
				// Handle any ISRC longer than 12 characters by taking the first 12
				if len(strValue) > 12 {
					result := strValue[:12]
					fmt.Printf("DEBUG: Returning first 12 chars of long ISRC: %s\n", result)
					return result
				}
				fmt.Printf("DEBUG: Returning ISRC as-is: %s\n", strValue)
				return strValue
			}
		}
	}

	fmt.Printf("DEBUG: No ISRC found in standard fields\n")
	return ""
}
