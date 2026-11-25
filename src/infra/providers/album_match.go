package providers

import (
	"strings"
	"unicode"
)

// normalizeMatch lowercases s and drops everything but letters and digits, so titles that only
// differ in case, punctuation or spacing compare equal.
func normalizeMatch(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}

// stripQuotes removes characters that would break out of a quoted search term.
func stripQuotes(s string) string {
	return strings.NewReplacer(`"`, "", `\`, "").Replace(strings.TrimSpace(s))
}
