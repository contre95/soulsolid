package database

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// LibraryRootReport summarises how stored track paths relate to a library root.
type LibraryRootReport struct {
	Root    string   // cleaned root the comparison was made against
	Total   int      // tracks in the database
	Outside int      // tracks whose stored path is not under Root
	Samples []string // a few offending paths, for diagnostics
}

// Mismatch reports whether any stored track path falls outside the root.
func (r LibraryRootReport) Mismatch() bool { return r.Outside > 0 }

// InspectLibraryRoot compares every stored track path against root. tracks.path
// bakes the root in, so tracks outside it are unreachable until they are moved.
func (d *SqliteLibrary) InspectLibraryRoot(ctx context.Context, root string) (LibraryRootReport, error) {
	report := LibraryRootReport{Root: filepath.Clean(root)}

	if err := d.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tracks`).Scan(&report.Total); err != nil {
		return report, fmt.Errorf("failed to count tracks: %w", err)
	}
	if report.Total == 0 {
		return report, nil
	}

	// Compare on a directory boundary so "/music" does not match "/musicvideos".
	sep := string(filepath.Separator)
	prefix := report.Root
	if !strings.HasSuffix(prefix, sep) {
		prefix += sep
	}
	n := utf8.RuneCountInString(prefix) // substr() counts runes, not bytes

	const outside = `SELECT %s FROM tracks WHERE substr(path, 1, ?) <> ?`

	if err := d.db.QueryRowContext(ctx, fmt.Sprintf(outside, "COUNT(*)"), n, prefix).Scan(&report.Outside); err != nil {
		return report, fmt.Errorf("failed to count tracks outside %q: %w", report.Root, err)
	}
	if report.Outside == 0 {
		return report, nil
	}

	rows, err := d.db.QueryContext(ctx, fmt.Sprintf(outside, "path")+" LIMIT 3", n, prefix)
	if err != nil {
		return report, fmt.Errorf("failed to sample tracks outside %q: %w", report.Root, err)
	}
	defer rows.Close()

	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return report, fmt.Errorf("failed to scan sampled track path: %w", err)
		}
		report.Samples = append(report.Samples, path)
	}

	return report, rows.Err()
}
