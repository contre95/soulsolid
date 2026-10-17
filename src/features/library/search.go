package library

import (
	"context"
	"fmt"
	"log/slog"

	library "github.com/contre95/soulsolid/src/music"
)

// SearchPage is one page of a unified library search, in display order:
// albums, then artists, then tracks.
type SearchPage struct {
	Albums  []*library.Album
	Artists []*library.Artist
	Tracks  []*library.Track
	Total   int
}

// UnifiedSearch searches albums, artists and tracks as one paginated list.
// With no text query and no filters it browses all tracks. Otherwise albums
// and artists matching the text query come first (capped, not paginated),
// followed by the paginated tracks matching the filter. A failure in one part
// of a search is logged and that part is left out, so the others still show.
func (s *Service) UnifiedSearch(ctx context.Context, filter library.TrackFilter, limit, offset int) (*SearchPage, error) {
	page := &SearchPage{}

	hasActiveFilters := filter.Genre != "" || filter.HasAcoustID != nil || filter.LyricsFilter != "" || filter.LyricsText != "" || filter.AddedAfter != "" || filter.AddedBefore != ""

	if filter.TextSearch == "" && !hasActiveFilters {
		tracksCount, err := s.GetTracksCount(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to count tracks: %w", err)
		}
		page.Total = tracksCount

		if offset < tracksCount {
			trackLimit := min(offset+limit, tracksCount) - offset
			if trackLimit > 0 {
				tracks, err := s.GetTracksPaginated(ctx, trackLimit, offset)
				if err != nil {
					return nil, fmt.Errorf("failed to load tracks: %w", err)
				}
				page.Tracks = tracks
			}
		}
		return page, nil
	}

	trackCount, err := s.GetTracksFilteredCount(ctx, &filter)
	if err != nil {
		slog.Error("Error counting tracks", "error", err)
	}

	var albums []*library.Album
	var artists []*library.Artist
	if filter.TextSearch != "" {
		albums, err = s.SearchAlbums(ctx, filter.TextSearch, 20, 0)
		if err != nil {
			slog.Error("Error searching albums", "error", err)
			albums = nil
		}
		artists, err = s.GetArtistsFilteredPaginated(ctx, 20, 0, filter.TextSearch)
		if err != nil {
			slog.Error("Error searching artists", "error", err)
			artists = nil
		}
	}
	albumsCount := len(albums)
	artistsCount := len(artists)
	page.Total = albumsCount + artistsCount + trackCount

	start := offset
	end := min(offset+limit, page.Total)

	// Albums: [0, albumsCount)
	if start < albumsCount {
		page.Albums = albums[start:min(end, albumsCount)]
	}

	// Artists: [albumsCount, albumsCount+artistsCount)
	if end > albumsCount {
		artistStart := max(0, start-albumsCount)
		artistEnd := min(end-albumsCount, artistsCount)
		if artistStart < artistEnd {
			page.Artists = artists[artistStart:artistEnd]
		}
	}

	// Tracks: [albumsCount+artistsCount, Total)
	trackOffset := albumsCount + artistsCount
	if end > trackOffset {
		trackStart := max(0, start-trackOffset)
		trackLimit := (end - trackOffset) - trackStart
		if trackLimit > 0 {
			tracks, err := s.GetTracksFilteredPaginated(ctx, trackLimit, trackStart, &filter)
			if err != nil {
				slog.Error("Error searching tracks", "error", err)
			} else {
				page.Tracks = tracks
			}
		}
	}

	return page, nil
}
