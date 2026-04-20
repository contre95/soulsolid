package reorganize

import (
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const maxFAT32Bytes = 255

var fat32Replacer = strings.NewReplacer(
	":", "-", "*", "-", "?", "-", `"`, "-",
	"<", "-", ">", "-", "|", "-", `\`, "-",
)

func sanitizeFAT32Path(path string) string {
	segments := strings.Split(path, string(filepath.Separator))
	last := len(segments) - 1
	for i, seg := range segments {
		segments[i] = sanitizeFAT32Segment(seg, i == last)
	}
	return strings.Join(segments, string(filepath.Separator))
}

func sanitizeFAT32Segment(seg string, isFilename bool) string {
	if seg == "" {
		return seg
	}
	seg = strings.ToValidUTF8(seg, "")
	result := fat32Replacer.Replace(seg)
	result = strings.TrimRight(result, ". ")
	if isFilename {
		ext := filepath.Ext(result)
		stem := result[:len(result)-len(ext)]
		maxStem := maxFAT32Bytes - len(ext)
		if maxStem < 1 {
			maxStem = 1
		}
		result = truncateBytesUTF8(stem, maxStem) + ext
	} else {
		result = truncateBytesUTF8(result, maxFAT32Bytes)
	}
	return result
}

// truncateBytesUTF8 cuts s to maxBytes, stepping back to a valid UTF-8 rune boundary.
func truncateBytesUTF8(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	truncated := s[:maxBytes]
	for len(truncated) > 0 {
		r, size := utf8.DecodeLastRuneInString(truncated)
		if r != utf8.RuneError || size != 1 {
			break
		}
		truncated = truncated[:len(truncated)-1]
	}
	return truncated
}
