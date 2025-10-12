package tag

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bogem/id3v2/v2"
	"github.com/contre95/soulsolid/src/features/config"
	"github.com/contre95/soulsolid/src/features/downloading"
	"github.com/contre95/soulsolid/src/music"
	"github.com/go-flac/flacpicture"
	"github.com/go-flac/flacvorbis"
	goflac "github.com/go-flac/go-flac"
	"github.com/nfnt/resize"
	_ "golang.org/x/image/webp"
	_ "image/gif"
)

// TagWriter implements writing tags into files for MP3 and FLAC formats.
type TagWriter struct {
	config *config.Manager
}

// NewTagWriter creates a new TagWriter.
func NewTagWriter(cfg *config.Manager) downloading.TagWriter {
	return &TagWriter{config: cfg}
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
		quality := t.config.Get().Downloaders.Artwork.Embedded.Quality
		if quality <= 0 {
			quality = 85
		}
		err = jpeg.Encode(&buf, resizedImg, &jpeg.Options{Quality: quality})
	case "png":
		err = png.Encode(&buf, resizedImg)
	default:
		// Default to JPEG
		quality := t.config.Get().Downloaders.Artwork.Embedded.Quality
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

// TagAudioData tags audio data in memory and returns the tagged data.
func (t *TagWriter) TagAudioData(ctx context.Context, audioData []byte, track *music.Track) ([]byte, error) {
	// Determine format from track
	ext := "." + strings.ToLower(track.Format)
	if ext == "." {
		// Try to detect from data
		if len(audioData) > 0 {
			if audioData[0] == 0xFF && (audioData[1]&0xE0) == 0xE0 {
				ext = ".mp3"
			} else if len(audioData) >= 4 && string(audioData[:4]) == "fLaC" {
				ext = ".flac"
			}
		}
	}

	switch ext {
	case ".mp3":
		return t.tagMP3Data(audioData, track)
	case ".flac":
		return t.tagFLACData(audioData, track)
	default:
		return audioData, fmt.Errorf("unsupported format: %s", ext)
	}
}

// tagMP3 handles MP3 tagging using id3v2.
func (t *TagWriter) tagMP3(ctx context.Context, filePath string, track *music.Track) error {
	// Open MP3 file with id3v2 library
	tag, err := id3v2.Open(filePath, id3v2.Options{Parse: false})
	if err != nil {
		return fmt.Errorf("failed to open MP3 file for tagging: %w", err)
	}
	defer tag.Close()

	// Set basic metadata
	tag.SetTitle(track.Title)
	if len(track.Artists) > 0 {
		tag.SetArtist(track.Artists[0].Artist.Name)
		tag.AddTextFrame(tag.CommonID("Album Artist"), id3v2.EncodingUTF8, track.Artists[0].Artist.Name)

		// Add additional artists if present
		if len(track.Artists) > 1 {
			var additionalArtists []string
			for i := 1; i < len(track.Artists); i++ {
				additionalArtists = append(additionalArtists, track.Artists[i].Artist.Name)
			}
			if len(additionalArtists) > 0 {
				tag.AddTextFrame(tag.CommonID("REMIXER"), id3v2.EncodingUTF8, strings.Join(additionalArtists, "; "))
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
		tag.AddTextFrame(tag.CommonID("Subtitle"), id3v2.EncodingUTF8, track.TitleVersion)
	}
	if track.Metadata.BPM > 0 {
		tag.AddTextFrame(tag.CommonID("BPM"), id3v2.EncodingUTF8, fmt.Sprintf("%.0f", track.Metadata.BPM))
	}
	if track.Metadata.Gain != 0 {
		tag.AddTextFrame(tag.CommonID("REPLAYGAIN_TRACK_GAIN"), id3v2.EncodingUTF8, fmt.Sprintf("%.2f dB", track.Metadata.Gain))
	}
	if track.Album != nil {
		if track.Album.Label != "" {
			tag.AddTextFrame(tag.CommonID("PUBLISHER"), id3v2.EncodingUTF8, track.Album.Label)
		}
		if track.Album.Barcode != "" {
			tag.AddTextFrame(tag.CommonID("BARCODE"), id3v2.EncodingUTF8, track.Album.Barcode)
		}
	}

	// Additional metadata
	if track.ISRC != "" {
		tag.AddTextFrame(tag.CommonID("ISRC"), id3v2.EncodingUTF8, track.ISRC)
	}
	if track.Metadata.TrackNumber > 0 {
		tag.AddTextFrame(tag.CommonID("Track number/Position in set"), id3v2.EncodingUTF8, fmt.Sprintf("%d", track.Metadata.TrackNumber))
	}
	if track.Metadata.DiscNumber > 0 {
		tag.AddTextFrame(tag.CommonID("Part of a set"), id3v2.EncodingUTF8, fmt.Sprintf("%d", track.Metadata.DiscNumber))
	}
	if track.Metadata.Composer != "" {
		tag.AddTextFrame(tag.CommonID("Composer"), id3v2.EncodingUTF8, track.Metadata.Composer)
	}
	if track.Metadata.Lyrics != "" {
		fmt.Printf("DEBUG: Writing lyrics to MP3 file %s: %s\n", filePath, track.Metadata.Lyrics)
		tag.AddTextFrame(tag.CommonID("Lyrics"), id3v2.EncodingUTF8, track.Metadata.Lyrics)
	} else {
		fmt.Printf("DEBUG: No lyrics to write to MP3 file %s\n", filePath)
	}

	// Add artwork if available
	if track.Album != nil && track.Album.ArtworkPath != "" {
		imgData, err := os.ReadFile(track.Album.ArtworkPath)
		if err != nil {
			slog.Warn("Failed to read local artwork file", "filePath", filePath, "artworkPath", track.Album.ArtworkPath, "error", err)
		} else if len(imgData) > 0 {
			// Resize image if configured
			if t.config != nil {
				cfg := t.config.Get()
				if cfg.Downloaders.Artwork.Embedded.Enabled {
					maxSize := cfg.Downloaders.Artwork.Embedded.Size
					if maxSize > 0 {
						resizedData, err := t.resizeImage(imgData, maxSize)
						if err != nil {
							slog.Warn("Failed to resize artwork for MP3", "filePath", filePath, "error", err)
						} else {
							imgData = resizedData
						}
					}
				}
			}

			// Always use JPEG for consistency
			mimeType := "image/jpeg"

			pic := id3v2.PictureFrame{
				Encoding:    id3v2.EncodingUTF8,
				MimeType:    mimeType,
				PictureType: id3v2.PTFrontCover,
				Description: "",
				Picture:     imgData,
			}
			tag.AddAttachedPicture(pic)
			slog.Debug("Embedded artwork in MP3", "filePath", filePath, "size", len(imgData), "type", mimeType)
		}
	}

	// Save the tag (this properly interleaves tags with audio data)
	if err := tag.Save(); err != nil {
		return fmt.Errorf("failed to save MP3 tags: %w", err)
	}

	slog.Info("Tagged MP3 file", "filePath", filePath, "title", track.Title)
	return nil
}

// tagMP3Data tags MP3 data in memory and returns the tagged data.
func (t *TagWriter) tagMP3Data(audioData []byte, track *music.Track) ([]byte, error) {
	// Create new tag
	tag := id3v2.NewEmptyTag()

	// Set basic metadata
	tag.SetTitle(track.Title)
	if len(track.Artists) > 0 {
		tag.SetArtist(track.Artists[0].Artist.Name)
		tag.AddTextFrame(tag.CommonID("Album Artist"), id3v2.EncodingUTF8, track.Artists[0].Artist.Name)

		// Add additional artists if present
		if len(track.Artists) > 1 {
			var additionalArtists []string
			for i := 1; i < len(track.Artists); i++ {
				additionalArtists = append(additionalArtists, track.Artists[i].Artist.Name)
			}
			if len(additionalArtists) > 0 {
				tag.AddTextFrame(tag.CommonID("REMIXER"), id3v2.EncodingUTF8, strings.Join(additionalArtists, "; "))
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
		tag.AddTextFrame(tag.CommonID("Subtitle"), id3v2.EncodingUTF8, track.TitleVersion)
	}
	if track.Metadata.BPM > 0 {
		tag.AddTextFrame(tag.CommonID("BPM"), id3v2.EncodingUTF8, fmt.Sprintf("%.0f", track.Metadata.BPM))
	}
	if track.Metadata.Gain != 0 {
		tag.AddTextFrame(tag.CommonID("REPLAYGAIN_TRACK_GAIN"), id3v2.EncodingUTF8, fmt.Sprintf("%.2f dB", track.Metadata.Gain))
	}
	if track.Album != nil {
		if track.Album.Label != "" {
			tag.AddTextFrame(tag.CommonID("PUBLISHER"), id3v2.EncodingUTF8, track.Album.Label)
		}
		if track.Album.Barcode != "" {
			tag.AddTextFrame(tag.CommonID("BARCODE"), id3v2.EncodingUTF8, track.Album.Barcode)
		}
	}

	// Additional metadata
	if track.ISRC != "" {
		tag.AddTextFrame(tag.CommonID("ISRC"), id3v2.EncodingUTF8, track.ISRC)
	}
	if track.Metadata.TrackNumber > 0 {
		tag.AddTextFrame(tag.CommonID("Track number/Position in set"), id3v2.EncodingUTF8, fmt.Sprintf("%d", track.Metadata.TrackNumber))
	}
	if track.Metadata.DiscNumber > 0 {
		tag.AddTextFrame(tag.CommonID("Part of a set"), id3v2.EncodingUTF8, fmt.Sprintf("%d", track.Metadata.DiscNumber))
	}
	if track.Metadata.Composer != "" {
		tag.AddTextFrame(tag.CommonID("Composer"), id3v2.EncodingUTF8, track.Metadata.Composer)
	}
	if track.Metadata.Lyrics != "" {
		fmt.Printf("DEBUG: Writing lyrics to MP3 data: %s\n", track.Metadata.Lyrics)
		tag.AddTextFrame(tag.CommonID("Lyrics"), id3v2.EncodingUTF8, track.Metadata.Lyrics)
	} else {
		fmt.Printf("DEBUG: No lyrics to write to MP3 data\n")
	}

	// Add artwork if available
	if track.Album != nil && track.Album.ArtworkPath != "" {
		imgData, err := os.ReadFile(track.Album.ArtworkPath)
		if err != nil {
			slog.Warn("Failed to read local artwork file", "artworkPath", track.Album.ArtworkPath, "error", err)
		} else if len(imgData) > 0 {
			// Resize image if configured
			if t.config != nil {
				cfg := t.config.Get()
				if cfg.Downloaders.Artwork.Embedded.Enabled {
					maxSize := cfg.Downloaders.Artwork.Embedded.Size
					if maxSize > 0 {
						resizedData, err := t.resizeImage(imgData, maxSize)
						if err != nil {
							slog.Warn("Failed to resize artwork for MP3", "error", err)
						} else {
							imgData = resizedData
						}
					}
				}
			}

			// Always use JPEG for consistency
			mimeType := "image/jpeg"

			pic := id3v2.PictureFrame{
				Encoding:    id3v2.EncodingUTF8,
				MimeType:    mimeType,
				PictureType: id3v2.PTFrontCover,
				Description: "",
				Picture:     imgData,
			}
			tag.AddAttachedPicture(pic)
			slog.Debug("Embedded artwork in MP3 data", "size", len(imgData), "type", mimeType)
		}
	}

	// Get tag bytes
	var buf bytes.Buffer
	_, err := tag.WriteTo(&buf)
	if err != nil {
		return audioData, fmt.Errorf("failed to write tag to buffer: %w", err)
	}
	tagData := buf.Bytes()

	// Prepend tag data to audio data
	taggedData := append(tagData, audioData...)

	slog.Info("Tagged MP3 data in memory", "title", track.Title, "originalSize", len(audioData), "taggedSize", len(taggedData))
	return taggedData, nil
}

// tagFLACData tags FLAC data in memory and returns the tagged data.
func (t *TagWriter) tagFLACData(audioData []byte, track *music.Track) ([]byte, error) {
	// For FLAC, we need to work with the file structure
	// Create a temporary file, tag it, then read it back
	tempDir := "/tmp"
	tempFile, err := os.CreateTemp(tempDir, "flac_tag_*.flac")
	if err != nil {
		return audioData, fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)
	defer tempFile.Close()

	// Write audio data to temp file
	if _, err := tempFile.Write(audioData); err != nil {
		return audioData, fmt.Errorf("failed to write audio data to temp file: %w", err)
	}
	tempFile.Close()

	// Tag the temp file
	if err := t.tagFLAC(context.Background(), tempPath, track); err != nil {
		return audioData, fmt.Errorf("failed to tag FLAC temp file: %w", err)
	}

	// Read back the tagged data
	taggedData, err := os.ReadFile(tempPath)
	if err != nil {
		return audioData, fmt.Errorf("failed to read tagged FLAC data: %w", err)
	}

	slog.Info("Tagged FLAC data in memory", "title", track.Title, "originalSize", len(audioData), "taggedSize", len(taggedData))
	return taggedData, nil
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

	// Add artwork as PICTURE metadata block - use local artwork path only
	var imgData []byte
	var imageDescription string
	var artworkPath string

	if track.Album != nil && track.Album.ArtworkPath != "" {
		// Use local artwork file
		artworkPath = track.Album.ArtworkPath
		imageDescription = "Cover"
		imgData, err = os.ReadFile(artworkPath)
		if err != nil {
			slog.Warn("Failed to read local artwork file for FLAC", "filePath", filePath, "artworkPath", artworkPath, "error", err)
		}
	}

	if len(imgData) > 0 {
		slog.Info("Embedding artwork in FLAC", "filePath", filePath, "artworkPath", artworkPath, "imgSize", len(imgData))

		// Resize image if configured
		if t.config != nil {
			cfg := t.config.Get()
			if cfg.Downloaders.Artwork.Embedded.Enabled {
				maxSize := cfg.Downloaders.Artwork.Embedded.Size
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
			} else {
				slog.Debug("Artwork embedding disabled in config", "filePath", filePath)
			}
		}

		// Detect MIME type from data
		mimeType := "image/jpeg" // default
		if len(imgData) >= 4 {
			if string(imgData[:4]) == "\x89PNG" {
				mimeType = "image/png"
			} else if string(imgData[:2]) == "\xFF\xD8" {
				mimeType = "image/jpeg"
			}
		}

		// Create PICTURE metadata block using flacpicture library
		pic, _ := flacpicture.NewFromImageData(flacpicture.PictureTypeFrontCover, imageDescription, imgData, mimeType)
		marshaled := pic.Marshal()
		pictureBlock := &goflac.MetaDataBlock{
			Type: goflac.Picture,
			Data: marshaled.Data,
		}
		f.Meta = append(f.Meta, pictureBlock)
		f.Meta = append(f.Meta, pictureBlock)
		slog.Info("Embedded artwork in FLAC", "filePath", filePath, "size", len(imgData), "type", mimeType, "blocks", len(f.Meta))

	} else {
		slog.Debug("No artwork data to embed in FLAC", "filePath", filePath)
	}

	// Save the file
	if err := f.Save(filePath); err != nil {
		return fmt.Errorf("failed to save FLAC file: %w", err)
	}

	return nil
}
