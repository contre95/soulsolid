package reorganize

import (
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// fat32Forbidden contains all characters disallowed in FAT32 filenames.
const fat32Forbidden = `:*?"<>|\`

// maxFAT32Bytes is the maximum byte length of a single FAT32 filename component.
const maxFAT32Bytes = 255

// sanitizeFAT32Path makes every segment of a file path UTF-8 valid and free of
// FAT32-forbidden characters (: * ? " < > | \). Path separators are preserved;
// only individual segments are processed. The final segment (the filename) has
// its extension preserved when truncating to the 255-byte FAT32 limit.
func sanitizeFAT32Path(path string) string {
	segments := strings.Split(path, string(filepath.Separator))
	last := len(segments) - 1
	for i, seg := range segments {
		segments[i] = sanitizeFAT32Segment(seg, i == last)
	}
	return strings.Join(segments, string(filepath.Separator))
}

// sanitizeFAT32Segment cleans a single path component.
// isFilename should be true for the final segment so the file extension is
// preserved when the name needs to be truncated.
func sanitizeFAT32Segment(seg string, isFilename bool) string {
	if seg == "" {
		return seg
	}

	// Strip invalid UTF-8 sequences.
	seg = strings.ToValidUTF8(seg, "")

	// Replace each FAT32-forbidden character with a hyphen.
	var b strings.Builder
	b.Grow(len(seg))
	for _, r := range seg {
		if strings.ContainsRune(fat32Forbidden, r) {
			b.WriteRune('-')
		} else {
			b.WriteRune(r)
		}
	}
	result := b.String()

	// FAT32 names must not end with a dot or a space (applies to both files
	// and directories; for files the extension already prevents a trailing dot
	// under normal circumstances, but we trim here to be safe).
	result = strings.TrimRight(result, ". ")

	// Enforce the 255-byte FAT32 filename limit.
	// For the filename segment, preserve the extension so the file remains
	// openable even after truncation.
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

// truncateBytesUTF8 shortens s to at most maxBytes bytes, cutting only at valid
// UTF-8 rune boundaries so the result is always a well-formed string.
func truncateBytesUTF8(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	truncated := s[:maxBytes]
	// Walk back until the slice ends on a valid rune boundary.
	for !utf8.ValidString(truncated) {
		truncated = truncated[:len(truncated)-1]
	}
	return truncated
}
