package reorganize

// PathSanitizer is the path-rewriting capability the reorganize feature needs.
// It is declared here, consumer-side, and implemented by an adapter in
// src/infra so this package never imports infrastructure.
type PathSanitizer interface {
	// SanitizeFAT32Path strips characters a FAT32 volume cannot represent.
	SanitizeFAT32Path(path string) string
	// ResolvePathConflict returns a non-colliding variant of path when a file
	// already exists at that location.
	ResolvePathConflict(path string) string
}
