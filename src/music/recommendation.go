package music

import "time"

// Recommendation represents a song recommendation from an external service.
type Recommendation struct {
	Title          string    `json:"title"`
	Artist         string    `json:"artist"`
	Album          string    `json:"album,omitempty"`
	Score          float64   `json:"score"`
	Source         string    `json:"source"` // "listenbrainz" or "lastfm"
	MBID           string    `json:"mbid,omitempty"`
	ImageURL       string    `json:"image_url,omitempty"`        // Album artwork
	ArtistImageURL string    `json:"artist_image_url,omitempty"` // Artist image
	Duration       int       `json:"duration,omitempty"`
	ListenedAt     time.Time `json:"listened_at,omitempty"`
	URL            string    `json:"url,omitempty"`
}
