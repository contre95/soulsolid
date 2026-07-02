// Package mcp exposes the SoulSolid domain services to AI agents through a
// Model Context Protocol (Streamable HTTP) endpoint mounted on the main app.
package mcp

import (
	"github.com/contre95/soulsolid/src/features/config"
	"github.com/contre95/soulsolid/src/features/downloading"
	"github.com/contre95/soulsolid/src/features/importing"
	"github.com/contre95/soulsolid/src/features/jobs"
	"github.com/contre95/soulsolid/src/features/library"
	"github.com/contre95/soulsolid/src/features/lyrics"
	"github.com/contre95/soulsolid/src/features/metadata"
	"github.com/contre95/soulsolid/src/features/playlists"
	"github.com/contre95/soulsolid/src/features/reorganize"
)

// Service holds the domain services the MCP tools delegate to. It is the same
// set of services the UI handlers use — MCP is just another transport.
type Service struct {
	library     *library.Service
	metadata    *metadata.Service
	lyrics      *lyrics.Service
	jobs        *jobs.Service
	downloading *downloading.Service
	playlists   *playlists.Service
	importing   *importing.Service
	reorganize  *reorganize.Service
	config      *config.Manager
}

// NewService creates a new MCP service.
func NewService(
	libraryService *library.Service,
	metadataService *metadata.Service,
	lyricsService *lyrics.Service,
	jobService *jobs.Service,
	downloadingService *downloading.Service,
	playlistsService *playlists.Service,
	importingService *importing.Service,
	reorganizeService *reorganize.Service,
	configManager *config.Manager,
) *Service {
	return &Service{
		library:     libraryService,
		metadata:    metadataService,
		lyrics:      lyricsService,
		jobs:        jobService,
		downloading: downloadingService,
		playlists:   playlistsService,
		importing:   importingService,
		reorganize:  reorganizeService,
		config:      configManager,
	}
}
