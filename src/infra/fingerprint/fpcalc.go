package fingerprint

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const fpcalcBin = "fpcalc"

// Fpcalc returns the Chromaprint fingerprint of an audio file and its
// duration in seconds.
//
// fpcalc gives up at the first frame its decoder rejects and exits non-zero.
// With exit status 3 it still prints the fingerprint of the audio before that
// frame; with 2, the frame came too early for any. ffmpeg skips such frames,
// so the file is then fingerprinted again from ffmpeg's decoded audio, which
// covers the whole of fpcalc's window. fpcalc's partial fingerprint is the
// fallback when that fails too.
func Fpcalc(ctx context.Context, filePath string) (string, float64, error) {
	if _, err := exec.LookPath(fpcalcBin); err != nil {
		return "", 0, fmt.Errorf("%s not found, please install chromaprint tools: %w", fpcalcBin, err)
	}
	fingerprint, duration, err := runFpcalc(exec.CommandContext(ctx, fpcalcBin, "-json", filePath))
	if err == nil {
		return fingerprint, duration, nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return "", 0, ctxErr
	}
	decodedFingerprint, decodedDuration, decodeErr := fpcalcDecoded(ctx, filePath)
	if decodeErr == nil {
		return decodedFingerprint, decodedDuration, nil
	}
	if fingerprint != "" {
		return fingerprint, duration, nil
	}
	return "", 0, fmt.Errorf("%w (fingerprinting ffmpeg's decoding instead: %v)", err, decodeErr)
}

// runFpcalc runs an `fpcalc -json` command and parses its fingerprint. The
// fingerprint is returned along with the error when fpcalc printed one but
// exited non-zero. Only stdout is parsed: fpcalc writes its errors to stderr,
// ahead of the JSON.
func runFpcalc(cmd *exec.Cmd) (string, float64, error) {
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	var result struct {
		Fingerprint string  `json:"fingerprint"`
		Duration    float64 `json:"duration"`
	}
	if parseErr := json.Unmarshal(output, &result); parseErr != nil && err == nil {
		return "", 0, fmt.Errorf("failed to parse %s output: %w", fpcalcBin, parseErr)
	}
	if err == nil && result.Fingerprint == "" {
		err = errors.New("no fingerprint in its output")
	}
	if err != nil {
		reason := strings.ReplaceAll(strings.TrimSpace(stderr.String()), "\n", "; ")
		return result.Fingerprint, result.Duration, fmt.Errorf("failed to generate fingerprint with %s: %w: %s", fpcalcBin, err, reason)
	}
	return result.Fingerprint, result.Duration, nil
}

// fpcalcDecoded fingerprints the audio ffmpeg decodes from a file. fpcalc
// cannot tell the duration of raw audio, so ffprobe reads it from the file.
func fpcalcDecoded(ctx context.Context, filePath string) (string, float64, error) {
	pcmReader, pcmWriter, err := os.Pipe()
	if err != nil {
		return "", 0, err
	}
	ffmpeg := exec.CommandContext(ctx, "ffmpeg", "-nostdin", "-v", "quiet", "-i", filePath, "-f", "s16le", "-ac", "2", "-ar", "44100", "-")
	ffmpeg.Stdout = pcmWriter
	err = ffmpeg.Start()
	pcmWriter.Close()
	if err != nil {
		pcmReader.Close()
		return "", 0, fmt.Errorf("ffmpeg: %w", err)
	}
	fpcalc := exec.CommandContext(ctx, fpcalcBin, "-json", "-format", "s16le", "-rate", "44100", "-channels", "2", "-")
	fpcalc.Stdin = pcmReader
	fingerprint, _, err := runFpcalc(fpcalc)
	// fpcalc stops reading after its window; closing our end of the pipe
	// lets ffmpeg exit on the broken pipe rather than block. That exit is
	// expected on the success path, but when fingerprinting failed too it
	// can carry the actual reason, such as an unreadable or corrupt file.
	pcmReader.Close()
	if waitErr := ffmpeg.Wait(); waitErr != nil && err != nil {
		err = errors.Join(err, fmt.Errorf("ffmpeg: %w", waitErr))
	}
	if err != nil {
		return "", 0, err
	}
	output, err := exec.CommandContext(ctx, "ffprobe", "-v", "quiet", "-show_entries", "format=duration", "-of", "csv=p=0", filePath).Output()
	if err != nil {
		return "", 0, fmt.Errorf("ffprobe: %w", err)
	}
	duration, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	if err != nil {
		return "", 0, fmt.Errorf("failed to parse ffprobe duration: %w", err)
	}
	return fingerprint, duration, nil
}
