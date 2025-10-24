package tagging

import (
	"math"
	"sort"
	"strings"

	"github.com/contre95/soulsolid/src/music"
)

// Scorer handles scoring and ranking of metadata search results
type Scorer struct {
	config *ScoringConfig
}

// ScoringConfig contains configuration for the scoring algorithm
type ScoringConfig struct {
	TitleWeight        int `yaml:"title_match"`
	ArtistWeight       int `yaml:"artist_match"`
	AlbumWeight        int `yaml:"album_match"`
	YearWeight         int `yaml:"year_match"`
	ISRCWeight         int `yaml:"isrc_match"`
	AutoApplyThreshold int `yaml:"auto_apply_threshold"`
	QueueThreshold     int `yaml:"queue_threshold"`
}

// MatchScore represents the scoring result for a metadata match
type MatchScore struct {
	TotalScore int      // Actual score achieved
	MaxScore   int      // Maximum possible score
	Confidence float64  // Score as percentage (0.0 - 1.0)
	Reasons    []string // List of matching criteria that contributed to score
}

// ScoredResult represents a search result with its score
type ScoredResult struct {
	Track    *music.Track
	Score    MatchScore
	Provider string
	Action   music.AutoTagAction
}

// NewScorer creates a new scorer with the given configuration
func NewScorer(config *ScoringConfig) *Scorer {
	return &Scorer{config: config}
}

// ScoreMatch calculates a confidence score for how well a candidate track matches the original
func (s *Scorer) ScoreMatch(original *music.Track, candidate *music.Track) MatchScore {
	score := 0
	maxScore := s.config.TitleWeight + s.config.ArtistWeight +
		s.config.AlbumWeight + s.config.YearWeight + s.config.ISRCWeight

	reasons := []string{}

	// Title matching (fuzzy string similarity)
	if titleSim := s.calculateStringSimilarity(original.Title, candidate.Title); titleSim > 0.8 {
		score += int(float64(s.config.TitleWeight) * titleSim)
		reasons = append(reasons, "title_match")
	}

	// Artist matching (exact name match for primary artist)
	if s.matchPrimaryArtist(original, candidate) {
		score += s.config.ArtistWeight
		reasons = append(reasons, "artist_exact")
	}

	// Album matching (exact album title match)
	if s.matchAlbum(original, candidate) {
		score += s.config.AlbumWeight
		reasons = append(reasons, "album_exact")
	}

	// Year matching (within 2 years)
	if s.matchYear(original, candidate) {
		score += s.config.YearWeight
		reasons = append(reasons, "year_close")
	}

	// ISRC matching (exact match - highest confidence indicator)
	if s.matchISRC(original, candidate) {
		score += s.config.ISRCWeight
		reasons = append(reasons, "isrc_exact")
	}

	confidence := float64(score) / float64(maxScore)

	return MatchScore{
		TotalScore: score,
		MaxScore:   maxScore,
		Confidence: confidence,
		Reasons:    reasons,
	}
}

// EvaluateMatches scores all candidates and returns them sorted by confidence
func (s *Scorer) EvaluateMatches(original *music.Track, candidates []*music.Track, providerName string) []ScoredResult {
	results := make([]ScoredResult, 0, len(candidates))

	for _, candidate := range candidates {
		score := s.ScoreMatch(original, candidate)
		results = append(results, ScoredResult{
			Track:    candidate,
			Score:    score,
			Provider: providerName,
			Action:   s.determineAction(score),
		})
	}

	// Sort by confidence descending (highest scores first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score.Confidence > results[j].Score.Confidence
	})

	return results
}

// determineAction decides what action to take based on the confidence score
func (s *Scorer) determineAction(score MatchScore) music.AutoTagAction {
	confidencePercent := score.Confidence * 100

	if confidencePercent >= float64(s.config.AutoApplyThreshold) {
		return music.AutoApply
	}
	if confidencePercent >= float64(s.config.QueueThreshold) {
		return music.QueueReview
	}
	return music.Reject
}

// calculateStringSimilarity performs a simple fuzzy string matching
// Returns a value between 0.0 (no similarity) and 1.0 (exact match)
func (s *Scorer) calculateStringSimilarity(a, b string) float64 {
	if a == b {
		return 1.0
	}

	// Normalize strings for comparison
	a = strings.ToLower(strings.TrimSpace(a))
	b = strings.ToLower(strings.TrimSpace(b))

	if a == b {
		return 1.0
	}

	// Simple Levenshtein distance ratio
	distance := levenshteinDistance(a, b)
	maxLen := math.Max(float64(len(a)), float64(len(b)))

	if maxLen == 0 {
		return 1.0
	}

	similarity := 1.0 - float64(distance)/maxLen
	return math.Max(0.0, similarity) // Ensure non-negative
}

// matchPrimaryArtist checks if the primary artists match exactly
func (s *Scorer) matchPrimaryArtist(original, candidate *music.Track) bool {
	if len(original.Artists) == 0 || len(candidate.Artists) == 0 {
		return false
	}

	originalArtist := strings.ToLower(strings.TrimSpace(original.Artists[0].Artist.Name))
	candidateArtist := strings.ToLower(strings.TrimSpace(candidate.Artists[0].Artist.Name))

	return originalArtist == candidateArtist
}

// matchAlbum checks if album titles match exactly
func (s *Scorer) matchAlbum(original, candidate *music.Track) bool {
	if original.Album == nil || candidate.Album == nil {
		return false
	}

	originalAlbum := strings.ToLower(strings.TrimSpace(original.Album.Title))
	candidateAlbum := strings.ToLower(strings.TrimSpace(candidate.Album.Title))

	return originalAlbum == candidateAlbum
}

// matchYear checks if years are within 2 years of each other
func (s *Scorer) matchYear(original, candidate *music.Track) bool {
	yearDiff := original.Metadata.Year - candidate.Metadata.Year
	return yearDiff >= -2 && yearDiff <= 2
}

// matchISRC checks for exact ISRC match
func (s *Scorer) matchISRC(original, candidate *music.Track) bool {
	return original.ISRC != "" && original.ISRC == candidate.ISRC
}

// levenshteinDistance calculates the edit distance between two strings
func levenshteinDistance(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	matrix := make([][]int, len(a)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(b)+1)
		matrix[i][0] = i
	}
	for j := range matrix[0] {
		matrix[0][j] = j
	}

	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}

			matrix[i][j] = int(math.Min(
				float64(matrix[i-1][j]+1), // deletion
				math.Min(
					float64(matrix[i][j-1]+1),      // insertion
					float64(matrix[i-1][j-1]+cost), // substitution
				),
			))
		}
	}

	return matrix[len(a)][len(b)]
}
