package metadata

import (
	"context"

	"github.com/contre95/soulsolid/src/features/tagging"
	"github.com/contre95/soulsolid/src/music"
)

// DeezerProvider implements MetadataProvider for Deezer
type DeezerProvider struct {
	enabled bool
}

// NewDeezerProvider creates a new Deezer provider
func NewDeezerProvider(enabled bool) *DeezerProvider {
	return &DeezerProvider{enabled: enabled}
}

func (p *DeezerProvider) SearchTracks(ctx context.Context, params tagging.SearchParams) ([]*music.Track, error) {
	// Mock implementation - simulate search results from Deezer API
	// In a real implementation, this would query the Deezer API with the search parameters

	// If we have search parameters, return relevant results
	if params.Title != "" || params.AlbumArtist != "" {
		tracks := []*music.Track{
			{
				Title: "Billie Jean",
				Artists: []music.ArtistRole{
					{Artist: &music.Artist{Name: "Michael Jackson"}, Role: "main"},
				},
				Album: &music.Album{
					Title: "Thriller",
					Artists: []music.ArtistRole{
						{Artist: &music.Artist{Name: "Michael Jackson"}, Role: "main"},
					},
				},
				Metadata: music.Metadata{
					Year:        1982,
					Genre:       "Pop",
					TrackNumber: 6,
					Composer:    "Michael Jackson",
				},
				ISRC: "USSM18200341",
			},
			{
				Title: "Beat It",
				Artists: []music.ArtistRole{
					{Artist: &music.Artist{Name: "Michael Jackson"}, Role: "main"},
				},
				Album: &music.Album{
					Title: "Thriller",
					Artists: []music.ArtistRole{
						{Artist: &music.Artist{Name: "Michael Jackson"}, Role: "main"},
					},
				},
				Metadata: music.Metadata{
					Year:        1982,
					Genre:       "Pop",
					TrackNumber: 5,
					Composer:    "Michael Jackson",
				},
				ISRC: "USSM18200340",
			},
			{
				Title: "Uptown Funk",
				Artists: []music.ArtistRole{
					{Artist: &music.Artist{Name: "Mark Ronson"}, Role: "main"},
					{Artist: &music.Artist{Name: "Bruno Mars"}, Role: "featured"},
				},
				Album: &music.Album{
					Title: "Uptown Special",
					Artists: []music.ArtistRole{
						{Artist: &music.Artist{Name: "Mark Ronson"}, Role: "main"},
					},
				},
				Metadata: music.Metadata{
					Year:        2014,
					Genre:       "Funk",
					TrackNumber: 1,
					Composer:    "Mark Ronson, Bruno Mars, Jeff Bhasker",
				},
				ISRC: "GBARL1400786",
			},
		}
		return tracks, nil
	}

	// Default fallback results
	tracks := []*music.Track{
		{
			Title: "Billie Jean",
			Artists: []music.ArtistRole{
				{Artist: &music.Artist{Name: "Michael Jackson"}, Role: "main"},
			},
			Album: &music.Album{
				Title: "Thriller",
				Artists: []music.ArtistRole{
					{Artist: &music.Artist{Name: "Michael Jackson"}, Role: "main"},
				},
			},
			Metadata: music.Metadata{
				Year:        1982,
				Genre:       "Pop",
				TrackNumber: 6,
				Composer:    "Michael Jackson",
			},
			ISRC: "USSM18200341",
		},
	}

	return tracks, nil
}

func (p *DeezerProvider) FetchMetadata(ctx context.Context, fingerprint string) (*music.Track, error) {
	// TODO: Implement full Deezer API integration
	// For now, return realistic placeholder data that demonstrates the functionality
	tracks, err := p.SearchTracks(ctx, tagging.SearchParams{})
	if err != nil || len(tracks) == 0 {
		return nil, err
	}
	return tracks[0], nil
}

func (p *DeezerProvider) Name() string    { return "deezer" }
func (p *DeezerProvider) IsEnabled() bool { return p.enabled }
