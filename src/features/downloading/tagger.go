package downloading

import (
	"context"

	"github.com/contre95/soulsolid/src/music"
)

// TagWriter defines the interface for writing metadata tags to music files.
type TagWriter interface {
	WriteFileTags(ctx context.Context, filePath string, track *music.Track) error
	TagAudioData(ctx context.Context, audioData []byte, track *music.Track) ([]byte, error)
}

// ArtworkService defines the interface for artwork operations.
type ArtworkService interface {
	SaveLocalArtwork(ctx context.Context, track *music.Track, dirPath string) error
	EmbedArtwork(audioPath, artworkPath string) error
	GetArtworkForTrack(ctx context.Context, track *music.Track) (string, func(), error)
}
