package tagging

import (
	"context"

	"github.com/contre95/soulsolid/src/music"
)

// TagReader is the interface for reading metadata from a music file.
// NOTE: Similar and atm using the same implementation of https://github.com/contre95/soulsolid/blob/f3b8b31c9e5fea2d53dfae36d435152272608f6f/src/features/importing/tagger.go?plain=1#L9-L12
type TagReader interface {
	ReadFileTags(ctx context.Context, filePath string) (*music.Track, error)
}

// TagWriter defines the interface for writing metadata tags to music files.
// NOTE: Similar and atm using the same implementation of https://github.com/contre95/soulsolid/blob/f3b8b31c9e5fea2d53dfae36d435152272608f6f/src/features/downloading/tagger.go?plain=1#L9-L12
type TagWriter interface {
	WriteFileTags(ctx context.Context, filePath string, track *music.Track) error
}
