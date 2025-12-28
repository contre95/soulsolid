package playlists

import (
	"github.com/contre95/soulsolid/src/music"
)

// Service handles playlist operations
type Service struct {
	playlistService music.PlaylistService
	m3uParser       M3UParser
}

// NewService creates a new playlists service
func NewService(playlistService music.PlaylistService, m3uParser M3UParser) *Service {
	return &Service{
		playlistService: playlistService,
		m3uParser:       m3uParser,
	}
}
