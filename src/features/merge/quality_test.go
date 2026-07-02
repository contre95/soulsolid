package merge

import (
	"testing"

	"github.com/contre95/soulsolid/src/music"
)

func TestBestQualityTrack(t *testing.T) {
	flac := &music.Track{ID: "flac", Format: "FLAC", Bitrate: 1024, BitDepth: 16, SampleRate: 44100}
	mp3High := &music.Track{ID: "mp3-320", Format: "MP3", Bitrate: 320, SampleRate: 44100}
	mp3Low := &music.Track{ID: "mp3-128", Format: "MP3", Bitrate: 128, SampleRate: 44100}
	hiRes := &music.Track{ID: "flac-hires", Format: "flac", Bitrate: 2304, BitDepth: 24, SampleRate: 96000}

	cases := []struct {
		name   string
		tracks []*music.Track
		want   string
	}{
		{"lossless beats lossy despite order", []*music.Track{mp3High, flac}, "flac"},
		{"lossless beats higher-bitrate lossy", []*music.Track{mp3High, mp3Low, flac}, "flac"},
		{"higher bitrate wins within lossy", []*music.Track{mp3Low, mp3High}, "mp3-320"},
		{"higher bitrate wins within lossless (case-insensitive format)", []*music.Track{flac, hiRes}, "flac-hires"},
		{"single track", []*music.Track{mp3Low}, "mp3-128"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := bestQualityTrack(c.tracks); got.ID != c.want {
				t.Errorf("bestQualityTrack() = %s, want %s", got.ID, c.want)
			}
		})
	}
}

func TestQualityBadge(t *testing.T) {
	cases := []struct {
		name  string
		track *music.Track
		want  string
	}{
		{"flac full info", &music.Track{Format: "flac", Bitrate: 1024, BitDepth: 16, SampleRate: 44100}, "FLAC · 1024 kbps · 16-bit/44.1 kHz"},
		{"mp3 without depth", &music.Track{Format: "MP3", Bitrate: 320}, "MP3 · 320 kbps"},
		{"no info", &music.Track{}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := qualityBadge(c.track); got != c.want {
				t.Errorf("qualityBadge() = %q, want %q", got, c.want)
			}
		})
	}
}
