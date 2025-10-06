package tag

import (
	"context"

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

func (p *DeezerProvider) FetchMetadata(ctx context.Context, fingerprint string) (*music.Track, error) {
	// TODO: Implement full Deezer API integration
	// For now, return realistic placeholder data that demonstrates the functionality
	return &music.Track{
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
	}, nil
}

func (p *DeezerProvider) Name() string    { return "deezer" }
func (p *DeezerProvider) IsEnabled() bool { return p.enabled }
