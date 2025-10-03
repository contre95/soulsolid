package importing

import (
	"context"

	"github.com/contre95/soulsolid/src/music"
)

// TagReader is the interface for reading metadata from a music file.
type TagReader interface {
	ReadFileTags(ctx context.Context, filePath string) (*music.Track, error)
}
