package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/contre95/soulsolid/src/features/config"
	"github.com/contre95/soulsolid/src/features/metadata"
	"github.com/contre95/soulsolid/src/infra/fingerprint"
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

// Service implements tagging.ChromaprintAcoustID for AcoustID lookups and chromaprint generation
type AcoustIDAPI struct {
	config *config.Manager
}

// NewAcoustIDService creates a new AcoustID service
func NewAcoustIDService(cfg *config.Manager) metadata.ChromaprintAcoustID {
	return &AcoustIDAPI{
		config: cfg,
	}
}

// LookupAcoustID looks up AcoustID using chromaprint
func (s *AcoustIDAPI) LookupAcoustID(ctx context.Context, chromaprint string, duration int) (string, error) {
	cfg := s.config.Get()

	// Check if AcoustID is enabled
	acoustidProvider, exists := cfg.Metadata.Providers["acoustid"]
	if !exists || !acoustidProvider.Enabled {
		return "", fmt.Errorf("AcoustID lookup is disabled in configuration")
	}

	if acoustidProvider.Secret == nil || *acoustidProvider.Secret == "" {
		return "", fmt.Errorf("AcoustID secret not configured")
	}

	// Prepare API request
	baseURL := "https://api.acoustid.org/v2/lookup"
	params := url.Values{}
	params.Add("client", *acoustidProvider.Secret)
	params.Add("meta", "recordings+sources")
	params.Add("duration", fmt.Sprintf("%d", duration))
	params.Add("fingerprint", chromaprint)

	requestURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Make HTTP request
	resp, err := httpClient.Do(req)
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

// GenerateChromaprint generates a chromaprint fingerprint for an audio file and returns the duration
func (s *AcoustIDAPI) GenerateChromaprint(ctx context.Context, filePath string) (string, int, error) {
	// Use fpcalc to generate fingerprint and get duration
	fingerprint, duration, err := s.generateFingerprintWithFpcalc(ctx, filePath)
	if err != nil {
		return "", 0, err
	}
	return fingerprint, duration, nil
}

// generateFingerprintWithFpcalc generates fingerprint using fpcalc command
func (s *AcoustIDAPI) generateFingerprintWithFpcalc(ctx context.Context, filePath string) (string, int, error) {
	fp, duration, err := fingerprint.Fpcalc(ctx, filePath)
	return fp, int(duration), err
}

// CompareChromaprints compares two chromaprints and returns a similarity score (0.0 to 1.0)
func (s *AcoustIDAPI) CompareChromaprints(cp1, cp2 string) (float64, error) {
	// For now, use simple string comparison
	// In a full implementation, this would use proper audio fingerprint comparison
	if cp1 == cp2 {
		return 1.0, nil
	}
	return 0.0, nil
}
