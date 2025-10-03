package importing

import (
	"context"

	"soulsolid/src/music"
)

// TagReader is the interface for reading metadata from a music file.
type TagReader interface {
	ReadFileTags(ctx context.Context, filePath string) (*music.Track, error)
}
