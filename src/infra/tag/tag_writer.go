package tag

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"log/slog"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	_ "image/gif"

	"github.com/bogem/id3v2/v2"
	"github.com/contre95/soulsolid/src/features/config"
	"github.com/contre95/soulsolid/src/features/downloading"
	"github.com/contre95/soulsolid/src/music"
	"github.com/go-flac/flacpicture"
	"github.com/go-flac/flacvorbis"
	goflac "github.com/go-flac/go-flac"
	"github.com/nfnt/resize"
	_ "golang.org/x/image/webp"
)

// TagWriter implements writing tags into files for MP3 and FLAC formats.
type TagWriter struct {
	artworkConfig config.EmbeddedArtwork
	mu            sync.Mutex
}

// SetCover embeds image data as cover art
func (t *TagWriter) SetCover(tag *id3v2.Tag, imgData []byte, mimeType string) error {
	pic := id3v2.PictureFrame{
		Encoding:    id3v2.EncodingUTF8,
		MimeType:    mimeType,
		PictureType: id3v2.PTFrontCover,
		Description: "Front cover",
		Picture:     imgData,
	}
	tag.AddAttachedPicture(pic)
	return nil
}

// NewTagWriter creates a new TagWriter.
func NewTagWriter(artworkConfig config.EmbeddedArtwork) downloading.TagWriter {
	return &TagWriter{artworkConfig: artworkConfig}
}

// WriteFileTags writes metadata to the file.
func (t *TagWriter) WriteFileTags(ctx context.Context, filePath string, track *music.Track) error {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".mp3":
		return t.tagMP3(filePath, track)
	case ".flac":
		return t.tagFLAC(filePath, track)
	default:
		return fmt.Errorf("unsupported format: %s", ext)
	}
}

// tagMP3 handles MP3 tagging using id3v2 - minimal approach like working example.
func (t *TagWriter) tagMP3(filePath string, track *music.Track) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	tag, err := id3v2.Open(filePath, id3v2.Options{Parse: true})
	if err != nil {
		return fmt.Errorf("failed to open MP3 file for tagging: %w", err)
	}
	defer tag.Close()

	// Set default encoding to UTF-8 to match working example app
	tag.SetDefaultEncoding(id3v2.EncodingUTF8)

	// Set minimal tags only (like working example app)
	track.Pretty()

	// Title
	if title := tag.Title(); title == "" && track.Title != "" {
		tag.SetTitle(track.Title)
	}

	// Artist
	if artist := tag.Artist(); artist == "" && len(track.Artists) > 0 {
		tag.SetArtist(track.Artists[0].Artist.Name)
	}

	// Album
	if album := tag.Album(); album == "" && track.Album != nil && track.Album.Title != "" {
		tag.SetAlbum(track.Album.Title)
	}

	// Genre
	if genre := tag.Genre(); genre == "" && track.Metadata.Genre != "" {
		tag.SetGenre(track.Metadata.Genre)
	}

	// Year
	if year := tag.Year(); year == "" && track.Metadata.Year > 0 {
		tag.SetYear(strconv.Itoa(track.Metadata.Year))
	}

	// ISRC
	if track.ISRC != "" {
		tag.AddTextFrame("TSRC", id3v2.EncodingUTF8, track.ISRC)
	}

	// Track number
	if track.Metadata.TrackNumber > 0 {
		tag.AddTextFrame("TRCK", id3v2.EncodingUTF8, strconv.Itoa(track.Metadata.TrackNumber))
	}

	// Cover artwork - embedded image only (URL references cause compatibility issues)
	if track.Album != nil && len(track.Album.ArtworkData) > 0 {
		mimeType := t.detectMimeType(track.Album.ArtworkData)

		// Resize if configured
		imgData := track.Album.ArtworkData
		if t.artworkConfig.Enabled && t.artworkConfig.Size > 0 {
			if resized, err := t.resizeImage(track.Album.ArtworkData, t.artworkConfig.Size); err == nil {
				// Validate that resized image is still valid
				if _, _, err := image.Decode(bytes.NewReader(resized)); err == nil {
					imgData = resized
					slog.Debug("Artwork resized successfully", "filePath", filePath, "originalSize", len(track.Album.ArtworkData), "resizedSize", len(resized))
				} else {
					slog.Warn("Resized artwork is invalid, using original", "filePath", filePath, "error", err)
				}
			} else {
				slog.Warn("Failed to resize artwork during tagging", "filePath", filePath, "error", err)
			}
		}

		// Final validation and embedding
		if _, _, err := image.Decode(bytes.NewReader(imgData)); err != nil {
			slog.Warn("Artwork data is invalid, skipping embedding", "filePath", filePath, "error", err)
		} else {
			if err := t.SetCover(tag, imgData, mimeType); err != nil {
				slog.Warn("Failed to set cover image", "filePath", filePath, "error", err)
			} else {
				slog.Debug("Embedded artwork image", "filePath", filePath, "bytes", len(imgData), "mimeType", mimeType)
			}
		}
	}

	if err := tag.Save(); err != nil {
		return fmt.Errorf("failed to save MP3 tags: %w", err)
	}

	slog.Info("Tagged MP3 successfully",
		"filePath", filePath,
		"title", track.Title,
		"artist", tag.Artist(),
		"album", tag.Album(),
	)

	return nil
}

// Helper: build title + version (e.g., "Song (Live)")

// tagFLAC handles FLAC tagging using Vorbis comments.
func (t *TagWriter) tagFLAC(filePath string, track *music.Track) error {
	t.mu.Lock()
	defer t.mu.Unlock()

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

	// Embed artwork if available
	if track.Album != nil && len(track.Album.ArtworkData) > 0 {
		imgData := track.Album.ArtworkData

		// Resize image if configured
		if t.artworkConfig.Enabled && t.artworkConfig.Size > 0 {
			maxSize := t.artworkConfig.Size
			if maxSize > 0 {
				slog.Debug("Resizing artwork for FLAC", "filePath", filePath, "maxSize", maxSize)
				resizedData, err := t.resizeImage(imgData, maxSize)
				if err != nil {
					slog.Warn("Failed to resize artwork for FLAC", "filePath", filePath, "error", err)
				} else {
					imgData = resizedData
					slog.Debug("Resized artwork for FLAC", "filePath", filePath, "newSize", len(imgData))
				}
			}
		}

		// Detect MIME type
		mimeType := "image/jpeg" // default
		if len(imgData) >= 4 {
			if string(imgData[:4]) == "\x89PNG" {
				mimeType = "image/png"
			} else if string(imgData[:2]) == "\xFF\xD8" {
				mimeType = "image/jpeg"
			}
		}

		// Create PICTURE metadata block
		pic, _ := flacpicture.NewFromImageData(flacpicture.PictureTypeFrontCover, "Cover", imgData, mimeType)
		marshaled := pic.Marshal()
		pictureBlock := &goflac.MetaDataBlock{
			Type: goflac.Picture,
			Data: marshaled.Data,
		}
		f.Meta = append(f.Meta, pictureBlock)
		slog.Info("Embedded artwork in FLAC", "filePath", filePath, "size", len(imgData), "type", mimeType, "blocks", len(f.Meta))
	}

	// Save the file
	if err := f.Save(filePath); err != nil {
		return fmt.Errorf("failed to save FLAC file: %w", err)
	}

	return nil
}

// detectMimeType detects the MIME type of image data using the image library.
func (t *TagWriter) detectMimeType(imgData []byte) string {
	_, format, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		slog.Warn("Failed to decode image for MIME detection, defaulting to jpeg", "error", err)
		return "image/jpeg"
	}
	slog.Debug("Detected image format", "format", format)
	switch format {
	case "jpeg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "webp":
		return "image/webp"
	default:
		slog.Warn("Unknown image format, defaulting to jpeg", "format", format)
		return "image/jpeg"
	}
}

// resizeImage resizes image data to fit within maxSize pixels, maintaining aspect ratio.
func (t *TagWriter) resizeImage(imgData []byte, maxSize int) ([]byte, error) {
	if maxSize <= 0 {
		return imgData, nil
	}

	// Decode image
	img, format, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		return imgData, fmt.Errorf("failed to decode image: %w", err)
	}

	// Get current bounds
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Check if resizing is needed
	if width <= maxSize && height <= maxSize {
		return imgData, nil
	}

	// Calculate new dimensions
	if width > height {
		height = (height * maxSize) / width
		width = maxSize
	} else {
		width = (width * maxSize) / height
		height = maxSize
	}

	// Resize
	resizedImg := resize.Resize(uint(width), uint(height), img, resize.Lanczos3)

	// Encode back
	var buf bytes.Buffer
	switch strings.ToLower(format) {
	case "jpeg":
		quality := t.artworkConfig.Quality
		if quality <= 0 {
			quality = 85
		}
		err = jpeg.Encode(&buf, resizedImg, &jpeg.Options{Quality: quality})
	case "png":
		err = png.Encode(&buf, resizedImg)
	default:
		// Default to JPEG
		quality := t.artworkConfig.Quality
		if quality <= 0 {
			quality = 85
		}
		err = jpeg.Encode(&buf, resizedImg, &jpeg.Options{Quality: quality})
	}
	if err != nil {
		return imgData, fmt.Errorf("failed to encode resized image: %w", err)
	}

	return buf.Bytes(), nil
}
