package files

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestMoveTrackFileNeverOverwrites(t *testing.T) {
	dir := t.TempDir()
	src, dst := filepath.Join(dir, "src.mp3"), filepath.Join(dir, "dst.mp3")
	for path, content := range map[string]string{src: "moved", dst: "kept"} {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	organizer := NewFileOrganizer(func() string { return dir }, func() string { return dir }, nil, func() bool { return false })

	if _, err := organizer.MoveTrackFile(context.Background(), src, dst); !errors.Is(err, os.ErrExist) {
		t.Fatalf("expected the move refused because %s exists, got %v", dst, err)
	}
	for path, content := range map[string]string{src: "moved", dst: "kept"} {
		if got, err := os.ReadFile(path); err != nil || string(got) != content {
			t.Errorf("expected %s to still hold %q, got %q (%v)", path, content, got, err)
		}
	}
}
