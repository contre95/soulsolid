package tag

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bogem/id3v2/v2"
	"soulsolid/src/features/downloading"
	"soulsolid/src/music"
	"github.com/go-flac/flacvorbis"
	goflac "github.com/go-flac/go-flac"
)

// TagWriter implements writing tags into files for MP3 and FLAC formats.
type TagWriter struct{}

// NewTagWriter creates a new TagWriter.
func NewTagWriter() downloading.TagWriter {
	return &TagWriter{}
}

// WriteFileTags writes metadata to the file.
func (t *TagWriter) WriteFileTags(ctx context.Context, filePath string, track *music.Track) error {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".mp3":
		return t.tagMP3(ctx, filePath, track)
	case ".flac":
		return t.tagFLAC(ctx, filePath, track)
	default:
		return fmt.Errorf("unsupported format: %s", ext)
	}
}

// tagMP3 handles MP3 tagging using id3v2.
func (t *TagWriter) tagMP3(ctx context.Context, filePath string, track *music.Track) error {
	tag, err := id3v2.Open(filePath, id3v2.Options{Parse: true})
	if err != nil {
		tag = id3v2.NewEmptyTag()
	}
	defer tag.Close()

	// Set basic metadata
	tag.SetTitle(track.Title)
	if len(track.Artists) > 0 {
		tag.SetArtist(track.Artists[0].Artist.Name)
		tag.AddTextFrame(tag.CommonID("Album Artist"), tag.DefaultEncoding(), track.Artists[0].Artist.Name)

		// Add additional artists if present
		if len(track.Artists) > 1 {
			var additionalArtists []string
			for i := 1; i < len(track.Artists); i++ {
				additionalArtists = append(additionalArtists, track.Artists[i].Artist.Name)
			}
			if len(additionalArtists) > 0 {
				tag.AddTextFrame(tag.CommonID("REMIXER"), tag.DefaultEncoding(), strings.Join(additionalArtists, "; "))
			}
		}
	}
	if track.Album != nil {
		tag.SetAlbum(track.Album.Title)
	}
	tag.SetYear(fmt.Sprintf("%d", track.Metadata.Year))
	tag.SetGenre(track.Metadata.Genre)

	// Set additional metadata
	if track.TitleVersion != "" {
		tag.AddTextFrame(tag.CommonID("Subtitle"), tag.DefaultEncoding(), track.TitleVersion)
	}
	if track.Metadata.BPM > 0 {
		tag.AddTextFrame(tag.CommonID("BPM"), tag.DefaultEncoding(), fmt.Sprintf("%.0f", track.Metadata.BPM))
	}
	if track.Metadata.Gain != 0 {
		tag.AddTextFrame(tag.CommonID("REPLAYGAIN_TRACK_GAIN"), tag.DefaultEncoding(), fmt.Sprintf("%.2f dB", track.Metadata.Gain))
	}
	if track.Album != nil {
		if track.Album.Label != "" {
			tag.AddTextFrame(tag.CommonID("PUBLISHER"), tag.DefaultEncoding(), track.Album.Label)
		}
		if track.Album.Barcode != "" {
			tag.AddTextFrame(tag.CommonID("BARCODE"), tag.DefaultEncoding(), track.Album.Barcode)
		}
	}

	// Additional metadata
	if track.ISRC != "" {
		tag.AddTextFrame(tag.CommonID("ISRC"), tag.DefaultEncoding(), track.ISRC)
	}
	if track.Metadata.TrackNumber > 0 {
		tag.AddTextFrame(tag.CommonID("Track number/Position in set"), tag.DefaultEncoding(), fmt.Sprintf("%d", track.Metadata.TrackNumber))
	}
	if track.Metadata.DiscNumber > 0 {
		tag.AddTextFrame(tag.CommonID("Part of a set"), tag.DefaultEncoding(), fmt.Sprintf("%d", track.Metadata.DiscNumber))
	}
	if track.Metadata.Composer != "" {
		tag.AddTextFrame(tag.CommonID("Composer"), tag.DefaultEncoding(), track.Metadata.Composer)
	}
	if track.Metadata.Lyrics != "" {
		fmt.Printf("DEBUG: Writing lyrics to MP3 file %s: %s\n", filePath, track.Metadata.Lyrics)
		tag.AddTextFrame(tag.CommonID("Lyrics"), tag.DefaultEncoding(), track.Metadata.Lyrics)
	} else {
		fmt.Printf("DEBUG: No lyrics to write to MP3 file %s\n", filePath)
	}

	// Embedded picture - prefer album cover, fallback to artist image for compilations
	var imageURL string
	var imageDescription string

	if track.Album != nil && track.Album.ImageLarge != "" {
		imageURL = track.Album.ImageLarge
		imageDescription = "Cover"
	} else if len(track.Artists) > 0 && track.Artists[0].Artist != nil && track.Artists[0].Artist.ImageLarge != "" {
		// Use artist image for compilations or when no album cover is available
		imageURL = track.Artists[0].Artist.ImageLarge
		imageDescription = "Artist"
	}

	if imageURL != "" {
		imgData, err := t.DownloadImage(ctx, imageURL)
		if err != nil {
			slog.Warn("Failed to download artwork for MP3", "filePath", filePath, "imageURL", imageURL, "error", err)
		} else if len(imgData) > 0 {
			// Detect MIME type from data
			mimeType := "image/jpeg" // default
			if len(imgData) >= 4 {
				if string(imgData[:4]) == "\x89PNG" {
					mimeType = "image/png"
				} else if string(imgData[:2]) == "\xFF\xD8" {
					mimeType = "image/jpeg"
				}
			}

			pic := id3v2.PictureFrame{
				Encoding:    tag.DefaultEncoding(),
				MimeType:    mimeType,
				PictureType: id3v2.PTFrontCover,
				Description: imageDescription,
				Picture:     imgData,
			}
			tag.AddAttachedPicture(pic)
			slog.Debug("Embedded artwork in MP3", "filePath", filePath, "size", len(imgData), "type", mimeType)
		}
	}

	return tag.Save()
}

// tagFLAC handles FLAC tagging using Vorbis comments.
func (t *TagWriter) tagFLAC(ctx context.Context, filePath string, track *music.Track) error {
	// Parse the FLAC file
	f, err := goflac.ParseFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to parse FLAC file: %w", err)
	}

	// Find existing Vorbis comment block
	var vorbisComment *flacvorbis.MetaDataBlockVorbisComment
	var commentIndex = -1

	for idx, meta := range f.Meta {
		if meta.Type == goflac.VorbisComment {
			vorbisComment, err = flacvorbis.ParseFromMetaDataBlock(*meta)
			if err != nil {
				return fmt.Errorf("failed to parse Vorbis comment: %w", err)
			}
			commentIndex = idx
			break
		}
	}

	// Create new Vorbis comment block if none exists
	if vorbisComment == nil {
		vorbisComment = flacvorbis.New()
	}

	// Set basic metadata
	vorbisComment.Add(flacvorbis.FIELD_TITLE, track.Title)

	if len(track.Artists) > 0 {
		vorbisComment.Add(flacvorbis.FIELD_ARTIST, track.Artists[0].Artist.Name)

		// Add additional artists if present
		if len(track.Artists) > 1 {
			for i := 1; i < len(track.Artists); i++ {
				vorbisComment.Add("REMIXER", track.Artists[i].Artist.Name)
			}
		}
	}

	if track.Album != nil {
		vorbisComment.Add(flacvorbis.FIELD_ALBUM, track.Album.Title)
		if len(track.Album.Artists) > 0 {
			vorbisComment.Add("ALBUMARTIST", track.Album.Artists[0].Artist.Name)

			// Add additional album artists if present
			if len(track.Album.Artists) > 1 {
				for i := 1; i < len(track.Album.Artists); i++ {
					vorbisComment.Add("ALBUMARTIST", track.Album.Artists[i].Artist.Name)
				}
			}
		}
	}

	if track.Metadata.Year > 0 {
		vorbisComment.Add(flacvorbis.FIELD_DATE, strconv.Itoa(track.Metadata.Year))
	}

	if track.Metadata.Genre != "" {
		vorbisComment.Add(flacvorbis.FIELD_GENRE, track.Metadata.Genre)
	}

	// Additional metadata
	if track.ISRC != "" {
		vorbisComment.Add(flacvorbis.FIELD_ISRC, track.ISRC)
	}

	if track.Metadata.TrackNumber > 0 {
		vorbisComment.Add(flacvorbis.FIELD_TRACKNUMBER, strconv.Itoa(track.Metadata.TrackNumber))
	}

	if track.Metadata.DiscNumber > 0 {
		vorbisComment.Add("DISCNUMBER", strconv.Itoa(track.Metadata.DiscNumber))
	}

	if track.Metadata.Composer != "" {
		vorbisComment.Add("COMPOSER", track.Metadata.Composer)
	}

	if track.Metadata.Lyrics != "" {
		fmt.Printf("DEBUG: Writing lyrics to FLAC file %s: %s\n", filePath, track.Metadata.Lyrics)
		vorbisComment.Add("LYRICS", track.Metadata.Lyrics)
	} else {
		fmt.Printf("DEBUG: No lyrics to write to FLAC file %s\n", filePath)
	}

	// Set additional metadata
	if track.TitleVersion != "" {
		vorbisComment.Add("VERSION", track.TitleVersion)
	}
	if track.Metadata.BPM > 0 {
		vorbisComment.Add("BPM", fmt.Sprintf("%.0f", track.Metadata.BPM))
	}
	if track.Metadata.Gain != 0 {
		vorbisComment.Add("REPLAYGAIN_TRACK_GAIN", fmt.Sprintf("%.2f dB", track.Metadata.Gain))
	}
	if track.Album != nil {
		if track.Album.Label != "" {
			vorbisComment.Add("LABEL", track.Album.Label)
		}
		if track.Album.Barcode != "" {
			vorbisComment.Add("BARCODE", track.Album.Barcode)
		}
	}

	// Marshal back to metadata block
	commentMeta := vorbisComment.Marshal()

	// Update or add the metadata block
	if commentIndex >= 0 {
		f.Meta[commentIndex] = &commentMeta
	} else {
		f.Meta = append(f.Meta, &commentMeta)
	}

	// Add artwork as PICTURE metadata block - prefer album cover, fallback to artist image
	var imageURL string
	var imageDescription string

	if track.Album != nil && track.Album.ImageLarge != "" {
		imageURL = track.Album.ImageLarge
		imageDescription = "Cover"
	} else if len(track.Artists) > 0 && track.Artists[0].Artist != nil && track.Artists[0].Artist.ImageLarge != "" {
		// Use artist image for compilations or when no album cover is available
		imageURL = track.Artists[0].Artist.ImageLarge
		imageDescription = "Artist"
	}

	if imageURL != "" {
		imgData, err := t.DownloadImage(ctx, imageURL)
		if err != nil {
			slog.Warn("Failed to download artwork for FLAC", "filePath", filePath, "imageURL", imageURL, "error", err)
		} else if len(imgData) > 0 {
			// Detect MIME type from data
			mimeType := "image/jpeg" // default
			if len(imgData) >= 4 {
				if string(imgData[:4]) == "\x89PNG" {
					mimeType = "image/png"
				} else if string(imgData[:2]) == "\xFF\xD8" {
					mimeType = "image/jpeg"
				}
			}

			// Create PICTURE metadata block manually
			// PICTURE block type is 6 according to FLAC specification
			pictureData := t.createFLACPictureBlock(mimeType, 3, imageDescription, imgData)
			pictureBlock := &goflac.MetaDataBlock{
				Type: 6, // PICTURE block type
				Data: pictureData,
			}
			f.Meta = append(f.Meta, pictureBlock)
			slog.Debug("Embedded artwork in FLAC", "filePath", filePath, "size", len(imgData), "type", mimeType)
		}
	}

	// Save the file
	if err := f.Save(filePath); err != nil {
		return fmt.Errorf("failed to save FLAC file: %w", err)
	}

	return nil
}

// createFLACPictureBlock creates a FLAC PICTURE metadata block
func (t *TagWriter) createFLACPictureBlock(mimeType string, pictureType uint32, description string, imageData []byte) []byte {
	// FLAC PICTURE block format:
	// 4 bytes: picture type (big endian)
	// 4 bytes: MIME type length (big endian)
	// n bytes: MIME type string
	// 4 bytes: description length (big endian)
	// n bytes: description string
	// 4 bytes: width (big endian) - set to 0
	// 4 bytes: height (big endian) - set to 0
	// 4 bytes: color depth (big endian) - set to 0
	// 4 bytes: color count (big endian) - set to 0
	// 4 bytes: image data length (big endian)
	// n bytes: image data

	mimeTypeBytes := []byte(mimeType)
	descriptionBytes := []byte(description)

	// Calculate total size
	totalSize := 4 + 4 + len(mimeTypeBytes) + 4 + len(descriptionBytes) + 4 + 4 + 4 + 4 + len(imageData)

	result := make([]byte, totalSize)

	offset := 0

	// Picture type (3 = Cover (front))
	result[offset] = byte(pictureType >> 24)
	result[offset+1] = byte(pictureType >> 16)
	result[offset+2] = byte(pictureType >> 8)
	result[offset+3] = byte(pictureType)
	offset += 4

	// MIME type length
	mimeLen := uint32(len(mimeTypeBytes))
	result[offset] = byte(mimeLen >> 24)
	result[offset+1] = byte(mimeLen >> 16)
	result[offset+2] = byte(mimeLen >> 8)
	result[offset+3] = byte(mimeLen)
	offset += 4

	// MIME type
	copy(result[offset:], mimeTypeBytes)
	offset += len(mimeTypeBytes)

	// Description length
	descLen := uint32(len(descriptionBytes))
	result[offset] = byte(descLen >> 24)
	result[offset+1] = byte(descLen >> 16)
	result[offset+2] = byte(descLen >> 8)
	result[offset+3] = byte(descLen)
	offset += 4

	// Description
	copy(result[offset:], descriptionBytes)
	offset += len(descriptionBytes)

	// Width (0 = unspecified)
	result[offset] = 0
	result[offset+1] = 0
	result[offset+2] = 0
	result[offset+3] = 0
	offset += 4

	// Height (0 = unspecified)
	result[offset] = 0
	result[offset+1] = 0
	result[offset+2] = 0
	result[offset+3] = 0
	offset += 4

	// Color depth (0 = unspecified)
	result[offset] = 0
	result[offset+1] = 0
	result[offset+2] = 0
	result[offset+3] = 0
	offset += 4

	// Color count (0 = unspecified)
	result[offset] = 0
	result[offset+1] = 0
	result[offset+2] = 0
	result[offset+3] = 0
	offset += 4

	// Image data length
	dataLen := uint32(len(imageData))
	result[offset] = byte(dataLen >> 24)
	result[offset+1] = byte(dataLen >> 16)
	result[offset+2] = byte(dataLen >> 8)
	result[offset+3] = byte(dataLen)
	offset += 4

	// Image data
	copy(result[offset:], imageData)

	return result
}

// DownloadImage fetches image data from URL with improved error handling.
func (t *TagWriter) DownloadImage(ctx context.Context, url string) ([]byte, error) {
	if url == "" {
		return nil, fmt.Errorf("empty image URL")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set user agent to avoid blocking
	req.Header.Set("User-Agent", "Soulsolid/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("image download returned status %d", resp.StatusCode)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "" && !strings.HasPrefix(contentType, "image/") {
		return nil, fmt.Errorf("invalid content type: %s", contentType)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("empty image data received")
	}

	// Basic validation - check if it looks like image data
	if len(data) < 4 {
		return nil, fmt.Errorf("image data too small")
	}

	return data, nil
}
