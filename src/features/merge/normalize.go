package merge

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// normalizeKey reduces a name to a comparison key by folding case and accents and dropping
// everything that isn't a letter or number. Two names that differ only by case, punctuation,
// accents, or spacing therefore produce the same key — e.g. "Un Verano Sin Tí",
// "un verano sin ti" and "hip-hop" / "Hip Hop" / "HipHop".
//
// It uses norm.NFD.String (stateless and safe for concurrent use) to decompose accented
// characters into a base rune plus combining marks, then drops the combining marks.
func normalizeKey(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(norm.NFD.String(s)) {
		switch {
		case unicode.Is(unicode.Mn, r):
			// combining mark (accent) left over from NFD decomposition — drop it
			continue
		case unicode.IsLetter(r) || unicode.IsNumber(r):
			b.WriteRune(r)
		}
	}
	return b.String()
}

// smartCanonical picks the best-looking display value among a group's variants as a default
// canonical suggestion. It prefers proper mixed-case strings over ALL-CAPS or all-lowercase,
// then multi-word strings, then longer ones; ties break alphabetically for determinism.
func smartCanonical(values []string) string {
	best := ""
	bestScore := -1
	for _, v := range values {
		score := canonicalScore(v)
		if score > bestScore || (score == bestScore && v < best) {
			best = v
			bestScore = score
		}
	}
	return best
}

func canonicalScore(v string) int {
	hasUpper, hasLower := false, false
	for _, r := range v {
		if unicode.IsUpper(r) {
			hasUpper = true
		}
		if unicode.IsLower(r) {
			hasLower = true
		}
	}
	score := 0
	switch {
	case hasUpper && hasLower:
		score += 1000 // "Hip Hop" beats "HIP HOP" and "hip hop"
	case hasUpper:
		score += 100
	}
	if strings.ContainsRune(v, ' ') {
		score += 50
	}
	score += len(v)
	return score
}
