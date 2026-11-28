package fingerprint

import (
	"context"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// newMP3 encodes seconds of a tone over noise, which fingerprints like music
// where a bare tone does not. It skips the test when ffmpeg or fpcalc is not
// installed.
func newMP3(t *testing.T, seconds string) string {
	t.Helper()
	for _, tool := range []string{"ffmpeg", "ffprobe", "fpcalc"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s not found in PATH", tool)
		}
	}
	path := filepath.Join(t.TempDir(), "track.mp3")
	cmd := exec.Command("ffmpeg", "-nostdin", "-loglevel", "error",
		"-f", "lavfi", "-i", "sine=frequency=440:duration="+seconds,
		"-f", "lavfi", "-i", "anoisesrc=amplitude=0.1:seed=1:duration="+seconds,
		"-filter_complex", "amix", "-c:a", "libmp3lame", "-b:a", "128k", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("ffmpeg failed to encode the fixture: %v\n%s", err, out)
	}
	return path
}

// damage overwrites a frame's worth of the file at offset with garbage that
// begins like a frame header, which fpcalc's decoder rejects.
func damage(t *testing.T, path string, offset int) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for i := range 400 {
		data[offset+i] = byte(i*37 + 11)
	}
	copy(data[offset:], []byte{0xFF, 0xFB, 0xF3})
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// fpcalcExitCode is fpcalc's own exit status on a file.
func fpcalcExitCode(t *testing.T, path string) int {
	t.Helper()
	err := exec.Command("fpcalc", "-json", path).Run()
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode()
	}
	if err != nil {
		t.Fatal(err)
	}
	return 0
}

func TestFpcalc_FingerprintsAFileWithDamagedFrames(t *testing.T) {
	for _, tc := range []struct {
		name     string
		offset   int
		exitCode int
	}{
		{"damage before fpcalc has a fingerprint", 8000, 2},
		{"damage after fpcalc has a fingerprint", 240000, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := newMP3(t, "30")
			damage(t, path, tc.offset)
			if code := fpcalcExitCode(t, path); code != tc.exitCode {
				t.Fatalf("expected the fixture to make fpcalc exit %d, got %d", tc.exitCode, code)
			}

			fingerprint, duration, err := Fpcalc(context.Background(), path)
			if err != nil {
				t.Fatalf("expected a fingerprint, got %v", err)
			}
			if fingerprint == "" {
				t.Error("expected a fingerprint, got none")
			}
			if math.Abs(duration-30) > 1 {
				t.Errorf("expected the file's duration of 30 s, got %v", duration)
			}
		})
	}
}

func TestFpcalc_GivesFpcalcsReasonForAFileTooShort(t *testing.T) {
	path := newMP3(t, "1")

	_, _, err := Fpcalc(context.Background(), path)
	if err == nil || !strings.Contains(err.Error(), "Empty fingerprint") {
		t.Fatalf("expected fpcalc's Empty fingerprint error, got %v", err)
	}
}

func TestFpcalc_KeepsAnUndamagedFilesFingerprint(t *testing.T) {
	path := newMP3(t, "30")
	want, err := exec.Command("fpcalc", "-plain", path).Output()
	if err != nil {
		t.Fatal(err)
	}

	fingerprint, duration, err := Fpcalc(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if fingerprint != strings.TrimSpace(string(want)) {
		t.Error("expected fpcalc's fingerprint of the file itself")
	}
	if math.Abs(duration-30) > 1 {
		t.Errorf("expected the file's duration of 30 s, got %v", duration)
	}
}
