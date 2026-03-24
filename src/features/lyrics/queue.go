package lyrics

import "github.com/contre95/soulsolid/src/music"

// QueueItemType represents the type of item in the lyrics queue

const (
	ExistingLyrics music.QueueItemType = "existing_lyrics"
	Lyric404       music.QueueItemType = "lyric_404"
	FailedLyrics   music.QueueItemType = "failed_lyrics"
)
