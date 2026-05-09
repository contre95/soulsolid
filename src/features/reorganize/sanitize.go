package reorganize

import "github.com/contre95/soulsolid/src/infra/files"

func sanitizeFAT32Path(path string) string {
	return files.SanitizeFAT32Path(path)
}

func resolvePathConflict(path string) string {
	return files.ResolvePathConflict(path)
}
