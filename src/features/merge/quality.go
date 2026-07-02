package merge

import (
	"fmt"
	"strings"

	"github.com/contre95/soulsolid/src/music"
)

// losslessFormats are the audio formats treated as lossless when ranking duplicate files.
var losslessFormats = map[string]bool{
	"flac": true, "alac": true, "wav": true, "wave": true, "aiff": true, "aif": true, "ape": true, "wv": true,
}

// isLossless reports whether the track's format is a lossless one.
func isLossless(t *music.Track) bool {
	return losslessFormats[strings.ToLower(t.Format)]
}

// betterQuality reports whether a is higher audio quality than b: lossless beats lossy, then
// higher bitrate, bit depth and sample rate break ties in that order.
func betterQuality(a, b *music.Track) bool {
	if la, lb := isLossless(a), isLossless(b); la != lb {
		return la
	}
	if a.Bitrate != b.Bitrate {
		return a.Bitrate > b.Bitrate
	}
	if a.BitDepth != b.BitDepth {
		return a.BitDepth > b.BitDepth
	}
	return a.SampleRate > b.SampleRate
}

// bestQualityTrack returns the highest-quality track of the group (the suggested one to keep).
func bestQualityTrack(tracks []*music.Track) *music.Track {
	best := tracks[0]
	for _, t := range tracks[1:] {
		if betterQuality(t, best) {
			best = t
		}
	}
	return best
}

// qualityBadge builds a short human-readable quality label, e.g. "FLAC · 16-bit/44.1 kHz" or
// "MP3 · 320 kbps".
func qualityBadge(t *music.Track) string {
	parts := make([]string, 0, 3)
	if t.Format != "" {
		parts = append(parts, strings.ToUpper(t.Format))
	}
	if t.Bitrate > 0 {
		parts = append(parts, fmt.Sprintf("%d kbps", t.Bitrate))
	}
	if t.BitDepth > 0 && t.SampleRate > 0 {
		parts = append(parts, fmt.Sprintf("%d-bit/%.4g kHz", t.BitDepth, float64(t.SampleRate)/1000))
	}
	return strings.Join(parts, " · ")
}
