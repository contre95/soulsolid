package tag

import (
	"context"

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

func (p *MusicBrainzProvider) FetchMetadata(ctx context.Context, fingerprint string) (*music.Track, error) {
	return &music.Track{
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
	}, nil
}

func (p *MusicBrainzProvider) Name() string    { return "musicbrainz" }
func (p *MusicBrainzProvider) IsEnabled() bool { return p.enabled }
