package reorganize

import (
	"path/filepath"
	"strings"
)

// fat32Forbidden contains all characters disallowed in FAT32 filenames.
const fat32Forbidden = `:*?"<>|\`

// sanitizeFAT32Path makes every segment of a file path UTF-8 valid and free of
// FAT32-forbidden characters (: * ? " < > | \). Path separators are preserved;
// only individual segments are processed.
func sanitizeFAT32Path(path string) string {
	segments := strings.Split(path, string(filepath.Separator))
	for i, seg := range segments {
		segments[i] = sanitizeFAT32Segment(seg)
	}
	return strings.Join(segments, string(filepath.Separator))
}

// sanitizeFAT32Segment cleans a single path component.
func sanitizeFAT32Segment(seg string) string {
	if seg == "" {
		return seg
	}
	// Strip invalid UTF-8 sequences by replacing them with nothing.
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
	// FAT32 names must not end with a dot or a space.
	result = strings.TrimRight(result, ". ")
	return result
}
