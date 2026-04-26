package files

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const maxFAT32Bytes = 255

var fat32Replacer = strings.NewReplacer(
	":", "-", "*", "-", "?", "-", `"`, "-",
	"<", "-", ">", "-", "|", "-", `\`, "-",
)

// SanitizeFAT32Path lowercases every path segment, strips FAT32-forbidden
// characters, trims trailing dots/spaces, and truncates to 255 bytes.
func SanitizeFAT32Path(path string) string {
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
	seg = strings.ToLower(seg)
	result := fat32Replacer.Replace(seg)
	result = strings.TrimRight(result, ". ")
	if isFilename {
		ext := filepath.Ext(result)
		stem := result[:len(result)-len(ext)]
		maxStem := max(maxFAT32Bytes-len(ext), 1)
		result = truncateBytesUTF8(stem, maxStem) + ext
	} else {
		result = truncateBytesUTF8(result, maxFAT32Bytes)
	}
	return result
}

// ResolvePathConflict returns path unchanged if no file exists there, otherwise
// appends _1, _2, … before the extension until a free slot is found.
func ResolvePathConflict(path string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}
	ext := filepath.Ext(path)
	stem := path[:len(path)-len(ext)]
	for i := 1; ; i++ {
		candidate := fmt.Sprintf("%s_%d%s", stem, i, ext)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}

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
