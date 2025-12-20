package library

import (
	"context"
	"errors"
	"testing"

	"github.com/contre95/soulsolid/src/music"
	"github.com/google/uuid"
)

// MockLibrary is a mock implementation of music.Library
type MockLibrary struct {
	music.Library // Embed interface to avoid implementing all methods for now, will panic if unused methods called
	artists       map[string]*music.Artist
	artistsByName map[string]*music.Artist
}

func NewMockLibrary() *MockLibrary {
	return &MockLibrary{
		artists:       make(map[string]*music.Artist),
		artistsByName: make(map[string]*music.Artist),
	}
}

func (m *MockLibrary) GetArtistByName(ctx context.Context, name string) (*music.Artist, error) {
	if artist, ok := m.artistsByName[name]; ok {
		return artist, nil
	}
	return nil, nil // Return nil, nil when not found, simulating database behavior
}

func (m *MockLibrary) AddArtist(ctx context.Context, artist *music.Artist) error {
	if _, ok := m.artists[artist.ID]; ok {
		return errors.New("artist already exists")
	}
	m.artists[artist.ID] = artist
	m.artistsByName[artist.Name] = artist
	return nil
}

func TestFindOrCreateArtist_CreatesNewArtist(t *testing.T) {
	mockLib := NewMockLibrary()
	// We don't need config manager for this specific test as it doesn't use it
	service := NewService(mockLib, nil)
	ctx := context.Background()
	artistName := "New Artist"

	// This should fail with current implementation because it returns error when not found
	artist, err := service.FindOrCreateArtist(ctx, artistName)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if artist == nil {
		t.Fatal("expected artist to be returned")
	}

	if artist.Name != artistName {
		t.Errorf("expected artist name %s, got %s", artistName, artist.Name)
	}

	// Verify it was added to library
	if _, ok := mockLib.artistsByName[artistName]; !ok {
		t.Error("artist was not added to library")
	}
}

func TestFindOrCreateArtist_ReturnsExistingArtist(t *testing.T) {
	mockLib := NewMockLibrary()
	existingArtist := &music.Artist{
		ID:   uuid.New().String(),
		Name: "Existing Artist",
	}
	mockLib.AddArtist(context.Background(), existingArtist)

	service := NewService(mockLib, nil)
	ctx := context.Background()

	artist, err := service.FindOrCreateArtist(ctx, existingArtist.Name)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if artist.ID != existingArtist.ID {
		t.Errorf("expected artist ID %s, got %s", existingArtist.ID, artist.ID)
	}
}
