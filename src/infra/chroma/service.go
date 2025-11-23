package chroma

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"strings"

	"github.com/contre95/soulsolid/src/features/config"
	"github.com/contre95/soulsolid/src/features/importing"
	"github.com/contre95/soulsolid/src/music"
)

// AcoustIDResponse represents response from AcoustID API
type AcoustIDResponse struct {
	Status string `json:"status"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
	Results []AcoustIDResult `json:"results"`
}

// AcoustIDResult represents a single result from AcoustID
type AcoustIDResult struct {
	ID         string              `json:"id"`
	Score      float64             `json:"score"`
	Recordings []AcoustIDRecording `json:"recordings"`
}

// AcoustIDRecording represents recording information from AcoustID
type AcoustIDRecording struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Artists []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"artists"`
	Duration int `json:"duration"`
}

// Service implements FingerprintReader for audio fingerprinting
type Service struct {
	config *config.Manager
}

// NewFingerprintService creates a new fingerprint service
func NewFingerprintService(cfg *config.Manager) importing.FingerprintProvider {
	return &Service{
		config: cfg,
	}
}

// GenerateFingerprint generates a fingerprint for an audio file
func (s *Service) GenerateFingerprint(ctx context.Context, filePath string) (string, error) {
	// Try to use fpcalc for now since it handles audio decoding
	// In future, we can implement proper audio decoding for gochroma
	fingerprint, err := s.generateFingerprintWithFpcalc(ctx, filePath)
	if err != nil {
		// If fpcalc is not available, provide a helpful error message
		if strings.Contains(err.Error(), "fpcalc not found") {
			return "", fmt.Errorf("fpcalc not found. Please install chromaprint tools:\n" +
				"  Nix: nix-shell -p chromaprint\n" +
				"  Ubuntu/Debian: sudo apt-get install chromaprint-tools\n" +
				"  macOS: brew install chromaprint\n" +
				"  Or download from: https://acoustid.org/chromaprint")
		}
		return "", err
	}
	return fingerprint, nil
}

// generateFingerprintWithFpcalc generates fingerprint using fpcalc command
// TODO: In future, replace this with gochroma library for pure Go implementation
// The gochroma library requires raw audio data, so we'd need to add audio decoding
func (s *Service) generateFingerprintWithFpcalc(ctx context.Context, filePath string) (string, error) {
	// Check if fpcalc is available
	if _, err := exec.LookPath("fpcalc"); err != nil {
		return "", fmt.Errorf("fpcalc not found. Please install chromaprint tools: %w", err)
	}

	// Run fpcalc to generate fingerprint
	cmd := exec.CommandContext(ctx, "fpcalc", "-json", filePath)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to generate fingerprint with fpcalc: %w", err)
	}

	// Parse JSON output
	var result struct {
		Fingerprint string  `json:"fingerprint"`
		Duration    float64 `json:"duration"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return "", fmt.Errorf("failed to parse fpcalc output: %w", err)
	}

	return result.Fingerprint, nil
}

// LookupAcoustID looks up AcoustID using fingerprint
func (s *Service) LookupAcoustID(ctx context.Context, fingerprint string, duration int) (string, error) {
	cfg := s.config.Get()

	// Check if AcoustID is enabled
	if !cfg.AcoustID.Enabled {
		return "", fmt.Errorf("AcoustID lookup is disabled in configuration")
	}

	if cfg.AcoustID.ClientKey == "" || cfg.AcoustID.ClientKey == "<acoustid_client_key>" {
		return "", fmt.Errorf("AcoustID client key not configured")
	}

	// Prepare API request
	baseURL := "https://api.acoustid.org/v2/lookup"
	params := url.Values{}
	params.Add("client", cfg.AcoustID.ClientKey)
	params.Add("meta", "recordings+sources")
	params.Add("duration", fmt.Sprintf("%d", duration))
	params.Add("fingerprint", fingerprint)

	requestURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Make HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to query AcoustID API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("AcoustID API returned status: %d", resp.StatusCode)
	}

	// Parse response
	var response AcoustIDResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to parse AcoustID response: %w", err)
	}

	if response.Status != "ok" {
		if response.Error != nil {
			return "", fmt.Errorf("AcoustID API error: %s", response.Error.Message)
		}
		return "", fmt.Errorf("AcoustID API returned status: %s", response.Status)
	}

	if len(response.Results) == 0 {
		return "", nil // No results found
	}

	// Return best match (highest score)
	bestResult := response.Results[0]
	for _, result := range response.Results {
		if result.Score > bestResult.Score {
			bestResult = result
		}
	}

	return bestResult.ID, nil
}

// GenerateFingerprintWithAcoustID generates fingerprint and looks up AcoustID
func (s *Service) GenerateFingerprintWithAcoustID(ctx context.Context, filePath string) (string, string, error) {
	// Generate fingerprint
	fingerprint, err := s.GenerateFingerprint(ctx, filePath)
	if err != nil {
		return "", "", err
	}

	// Get duration for AcoustID lookup
	duration, err := s.getAudioDuration(filePath)
	if err != nil {
		return fingerprint, "", fmt.Errorf("failed to get audio duration: %w", err)
	}

	// Lookup AcoustID
	acoustID, err := s.LookupAcoustID(ctx, fingerprint, duration)
	if err != nil {
		return fingerprint, "", fmt.Errorf("AcoustID lookup failed: %w", err)
	}

	return fingerprint, acoustID, nil
}

// getAudioDuration gets audio duration using fpcalc
func (s *Service) getAudioDuration(filePath string) (int, error) {
	cmd := exec.Command("fpcalc", "-json", filePath)
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to get audio duration: %w", err)
	}

	var result struct {
		Duration float64 `json:"duration"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return 0, fmt.Errorf("failed to parse fpcalc output: %w", err)
	}

	return int(result.Duration), nil
}

// CompareFingerprints compares two fingerprints and returns a similarity score (0.0 to 1.0)
func (s *Service) CompareFingerprints(fp1, fp2 string) (float64, error) {
	// For now, use simple string comparison
	// In a full implementation, this would use proper audio fingerprint comparison
	if fp1 == fp2 {
		return 1.0, nil
	}
	return 0.0, nil
}

// UpdateTrackWithAcoustID updates a track with AcoustID information
func (s *Service) UpdateTrackWithAcoustID(ctx context.Context, track *music.Track) error {
	if track.Path == "" {
		return fmt.Errorf("track path is empty")
	}

	// Generate fingerprint and lookup AcoustID
	fingerprint, acoustID, err := s.GenerateFingerprintWithAcoustID(ctx, track.Path)
	if err != nil {
		return fmt.Errorf("failed to generate fingerprint and lookup AcoustID: %w", err)
	}

	// Update track
	track.ChromaprintFingerprint = fingerprint
	if acoustID != "" {
		track.AcoustID = acoustID
	}

	return nil
}
