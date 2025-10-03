package importing

import "soulsolid/src/music"

// PathParser is the interface for rendering a track's destination path based on its metadata.
type PathParser interface {
	RenderPath(track *music.Track) (string, error)
}
