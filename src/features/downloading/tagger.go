package downloading

import (
	"context"

	"soulsolid/src/music"
)

// TagWriter defines the interface for writing metadata tags to music files.
type TagWriter interface {
	WriteFileTags(ctx context.Context, filePath string, track *music.Track) error
}
