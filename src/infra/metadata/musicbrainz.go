package metadata

import (
	"context"

	"github.com/contre95/soulsolid/src/features/tagging"
	"github.com/contre95/soulsolid/src/music"
)

// MusicBrainzProvider implements MetadataProvider for MusicBrainz
type MusicBrainzProvider struct {
	enabled bool
}

// NewMusicBrainzProvider creates a new MusicBrainz provider
func NewMusicBrainzProvider(enabled bool) *MusicBrainzProvider {
	return &MusicBrainzProvider{enabled: enabled}
}

func (p *MusicBrainzProvider) SearchTracks(ctx context.Context, params tagging.SearchParams) ([]*music.Track, error) {
	// Mock implementation - simulate search results from MusicBrainz API
	// In a real implementation, this would query the MusicBrainz API with the search parameters

	// If we have search parameters, return relevant results
	if params.Title != "" || params.AlbumArtist != "" {
		tracks := []*music.Track{
			{
				Title: "Bohemian Rhapsody",
				Artists: []music.ArtistRole{
					{Artist: &music.Artist{Name: "Queen"}, Role: "main"},
				},
				Album: &music.Album{
					Title: "A Night at the Opera",
					Artists: []music.ArtistRole{
						{Artist: &music.Artist{Name: "Queen"}, Role: "main"},
					},
				},
				Metadata: music.Metadata{
					Year:        1975,
					Genre:       "Rock",
					TrackNumber: 11,
					Composer:    "Freddie Mercury",
				},
				ISRC: "GBCEE7500710",
				SourceData: music.SourceData{
					Source: "musicbrainz",
					URL:    "https://musicbrainz.org/recording/9a4b5b5b-7b5b-4b5b-8b5b-9a4b5b5b7b5b",
				},
			},
			{
				Title: "Stairway to Heaven",
				Artists: []music.ArtistRole{
					{Artist: &music.Artist{Name: "Led Zeppelin"}, Role: "main"},
				},
				Album: &music.Album{
					Title: "Led Zeppelin IV",
					Artists: []music.ArtistRole{
						{Artist: &music.Artist{Name: "Led Zeppelin"}, Role: "main"},
					},
				},
				Metadata: music.Metadata{
					Year:        1971,
					Genre:       "Rock",
					TrackNumber: 4,
					Composer:    "Jimmy Page, Robert Plant",
				},
				ISRC: "USAT21300959",
				SourceData: music.SourceData{
					Source: "musicbrainz",
					URL:    "https://musicbrainz.org/recording/8c4a4a4a-6a6a-4a4a-9a4a-8c4a4a4a6a6a",
				},
			},
			{
				Title: "Hotel California",
				Artists: []music.ArtistRole{
					{Artist: &music.Artist{Name: "Eagles"}, Role: "main"},
				},
				Album: &music.Album{
					Title: "Hotel California",
					Artists: []music.ArtistRole{
						{Artist: &music.Artist{Name: "Eagles"}, Role: "main"},
					},
				},
				Metadata: music.Metadata{
					Year:        1976,
					Genre:       "Rock",
					TrackNumber: 1,
					Composer:    "Don Felder, Don Henley, Glenn Frey",
				},
				ISRC: "USWB17600001",
				SourceData: music.SourceData{
					Source: "musicbrainz",
					URL:    "https://musicbrainz.org/recording/7b3b3b3b-5b5b-3b3b-8b3b-7b3b3b3b5b5b",
				},
			},
		}
		return tracks, nil
	}

	// Default fallback results
	tracks := []*music.Track{
		{
			Title: "Bohemian Rhapsody",
			Artists: []music.ArtistRole{
				{Artist: &music.Artist{Name: "Queen"}, Role: "main"},
			},
			Album: &music.Album{
				Title: "A Night at the Opera",
				Artists: []music.ArtistRole{
					{Artist: &music.Artist{Name: "Queen"}, Role: "main"},
				},
			},
			Metadata: music.Metadata{
				Year:        1975,
				Genre:       "Rock",
				TrackNumber: 11,
				Composer:    "Freddie Mercury",
			},
			ISRC: "GBCEE7500710",
			SourceData: music.SourceData{
				Source: "musicbrainz",
				URL:    "https://musicbrainz.org/recording/9a4b5b5b-7b5b-4b5b-8b5b-9a4b5b5b7b5b",
			},
		},
	}

	return tracks, nil
}

// Legacy method for backward compatibility
func (p *MusicBrainzProvider) FetchMetadata(ctx context.Context, fingerprint string) (*music.Track, error) {
	tracks, err := p.SearchTracks(ctx, tagging.SearchParams{})
	if err != nil || len(tracks) == 0 {
		return nil, err
	}
	return tracks[0], nil
}

func (p *MusicBrainzProvider) Name() string    { return "musicbrainz" }
func (p *MusicBrainzProvider) IsEnabled() bool { return p.enabled }
