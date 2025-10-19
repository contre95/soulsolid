package tagging

import (
	"fmt"
	"log/slog"

	"github.com/contre95/soulsolid/src/music"
	"github.com/gofiber/fiber/v2"
)

// Handler handles tag editing requests
type Handler struct {
	service *Service
}

// NewHandler creates a new tag handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RenderTagEditor renders the tag editing page
func (h *Handler) RenderTagEditor(c *fiber.Ctx) error {
	slog.Debug("RenderTagEditor handler called", "trackId", c.Params("trackId"))

	trackID := c.Params("trackId")
	if trackID == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Track ID is required")
	}

	// Get track data for editing
	track, err := h.service.GetTrackForEditing(c.Context(), trackID)
	if err != nil {
		slog.Error("Failed to get track for editing", "error", err, "trackId", trackID)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to load track data")
	}

	// Fetch all artists and albums for dropdowns
	artists, err := h.service.libraryRepo.GetArtists(c.Context())
	if err != nil {
		slog.Error("Failed to get artists for dropdown", "error", err)
		artists = []*music.Artist{} // Continue with empty list
	}

	albums, err := h.service.libraryRepo.GetAlbums(c.Context())
	if err != nil {
		slog.Error("Failed to get albums for dropdown", "error", err)
		albums = []*music.Album{} // Continue with empty list
	}

	// Ensure track's artists are included in the dropdown, even if missing from main query
	artistMap := make(map[string]bool)
	for _, artist := range artists {
		artistMap[artist.ID] = true
	}
	// Add track artists (include those without IDs for MusicBrainz fetched data)
	for _, artistRole := range track.Artists {
		if artistRole.Artist != nil {
			artistID := artistRole.Artist.ID
			if artistID == "" {
				// Generate a temporary ID for artists without database IDs (for dropdown display)
				artistID = "temp_" + artistRole.Artist.Name
				artistRole.Artist.ID = artistID
			}
			if !artistMap[artistID] {
				artists = append(artists, artistRole.Artist)
				artistMap[artistID] = true
			}
		}
	}
	// Add album artists (include those without IDs for MusicBrainz fetched data)
	if track.Album != nil {
		for _, artistRole := range track.Album.Artists {
			if artistRole.Artist != nil {
				artistID := artistRole.Artist.ID
				if artistID == "" {
					// Generate a temporary ID for artists without database IDs (for dropdown display)
					artistID = "temp_" + artistRole.Artist.Name
					artistRole.Artist.ID = artistID
				}
				if !artistMap[artistID] {
					artists = append(artists, artistRole.Artist)
					artistMap[artistID] = true
				}
			}
		}
	}

	// Check if this is a fetch metadata request
	if c.Query("fetch") == "true" {
		fetchedTrack, err := h.service.FetchMetadataForTrack(c.Context(), trackID)
		if err != nil {
			slog.Warn("Failed to fetch metadata, using existing data", "error", err)
			// Continue with existing track data
		} else {
			// Merge fetched data with existing track (preserve file-specific data)
			track = h.mergeFetchedData(track, fetchedTrack)
			slog.Info("Metadata fetched successfully", "trackId", trackID)
		}
	}

	// Ensure track has valid ID for template
	if track.ID == "" {
		track.ID = trackID
	}

	// Determine selected album artist ID for template
	selectedAlbumArtistID := ""
	if track.Album != nil && len(track.Album.Artists) > 0 {
		selectedAlbumArtistID = track.Album.Artists[0].Artist.ID
	}

	// Create map of selected artist IDs for template
	selectedArtistIDs := make(map[string]bool)
	for _, artistRole := range track.Artists {
		if artistRole.Artist != nil && artistRole.Artist.ID != "" {
			selectedArtistIDs[artistRole.Artist.ID] = true
		}
	}

	// Check if request is HTMX or full page
	if c.Get("HX-Request") == "true" {
		// Return just the section content for HTMX requests
		return c.Render("sections/tag", fiber.Map{
			"Track":                 track,
			"Artists":               artists,
			"Albums":                albums,
			"SelectedAlbumArtistID": selectedAlbumArtistID,
			"SelectedArtistIDs":     selectedArtistIDs,
			"EnabledProviders":      h.service.GetEnabledProviders(),
		})
	}

	// Return full page for direct navigation
	return c.Render("main", fiber.Map{
		"Title":                 "Edit Tags",
		"Track":                 track,
		"IsTagEdit":             true,
		"Artists":               artists,
		"Albums":                albums,
		"SelectedAlbumArtistID": selectedAlbumArtistID,
		"SelectedArtistIDs":     selectedArtistIDs,
		"EnabledProviders":      h.service.GetEnabledProviders(),
	})
}

// getProviderColors returns color classes for a given provider
func (h *Handler) getProviderColors(providerName string) map[string]string {
	switch providerName {
	case "musicbrainz":
		return map[string]string{
			"label":     "text-orange-600 dark:text-orange-300",
			"border":    "border-orange-400 dark:border-orange-300",
			"focusRing": "focus:ring-orange-500 focus:border-orange-500",
			"text":      "text-violet-700 dark:text-violet-300",
		}
	case "discogs":
		return map[string]string{
			"label":     "text-black dark:text-white",
			"border":    "border-black dark:border-purple-600",
			"focusRing": "focus:ring-black focus:border-black",
			"text":      "text-black dark:text-white",
		}
	case "deezer":
		return map[string]string{
			"label":     "text-purple-600 dark:text-purple-300",
			"border":    "border-purple-400 dark:border-purple-300",
			"focusRing": "focus:ring-purple-500 focus:border-purple-500",
			"text":      "text-purple-700 dark:text-purple-300",
		}
	default:
		// Default to orange for unknown providers
		return map[string]string{
			"label":     "text-orange-600 dark:text-orange-300",
			"border":    "border-orange-400 dark:border-orange-300",
			"focusRing": "focus:ring-orange-500 focus:border-orange-500",
			"text":      "text-violet-700 dark:text-violet-300",
		}
	}
}

// FetchFromProvider handles fetching metadata from any provider and rendering the form
func (h *Handler) FetchFromProvider(c *fiber.Ctx) error {
	trackID := c.Params("trackId")
	providerName := c.Params("provider")

	if trackID == "" || trackID == "0" {
		slog.Error("Invalid track ID in FetchFromProvider", "trackId", trackID)
		return c.Status(fiber.StatusBadRequest).SendString("Invalid track ID")
	}

	if providerName == "" {
		slog.Error("Invalid provider name in FetchFromProvider", "provider", providerName)
		return c.Status(fiber.StatusBadRequest).SendString("Invalid provider name")
	}

	// Get current track data
	track, err := h.service.GetTrackForEditing(c.Context(), trackID)
	if err != nil {
		slog.Error("Failed to get track for editing", "error", err, "trackId", trackID)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to load track data")
	}

	// Fetch all artists and albums for dropdowns
	artists, err := h.service.libraryRepo.GetArtists(c.Context())
	if err != nil {
		slog.Error("Failed to get artists for dropdown", "error", err)
		artists = []*music.Artist{} // Continue with empty list
	}

	albums, err := h.service.libraryRepo.GetAlbums(c.Context())
	if err != nil {
		slog.Error("Failed to get albums for dropdown", "error", err)
		albums = []*music.Album{} // Continue with empty list
	}

	// Ensure track's artists are included in the dropdown, even if missing from main query
	artistMap := make(map[string]bool)
	for _, artist := range artists {
		artistMap[artist.ID] = true
	}
	// Add track artists (include those without IDs for fetched data)
	for _, artistRole := range track.Artists {
		if artistRole.Artist != nil {
			artistID := artistRole.Artist.ID
			if artistID == "" {
				// Generate a temporary ID for artists without database IDs (for dropdown display)
				artistID = "temp_" + artistRole.Artist.Name
				artistRole.Artist.ID = artistID
			}
			if !artistMap[artistID] {
				artists = append(artists, artistRole.Artist)
				artistMap[artistID] = true
			}
		}
	}
	// Add album artists (include those without IDs for fetched data)
	if track.Album != nil {
		for _, artistRole := range track.Album.Artists {
			if artistRole.Artist != nil {
				artistID := artistRole.Artist.ID
				if artistID == "" {
					// Generate a temporary ID for artists without database IDs (for dropdown display)
					artistID = "temp_" + artistRole.Artist.Name
					artistRole.Artist.ID = artistID
				}
				if !artistMap[artistID] {
					artists = append(artists, artistRole.Artist)
					artistMap[artistID] = true
				}
			}
		}
	}

	// Create map of selected artist IDs for template
	selectedArtistIDs := make(map[string]bool)
	for _, artistRole := range track.Artists {
		if artistRole.Artist != nil && artistRole.Artist.ID != "" {
			selectedArtistIDs[artistRole.Artist.ID] = true
		}
	}

	// Debug logging
	slog.Debug("FetchFromProvider called", "trackId", trackID, "provider", providerName, "trackTitle", track.Title, "trackID", track.ID)

	// Fetch metadata
	fetchedTrack, err := h.service.FetchMetadataForTrack(c.Context(), trackID)
	if err != nil {
		slog.Warn("Failed to fetch metadata, using existing data", "error", err, "trackId", trackID, "provider", providerName)
		// Determine selected album artist ID for template
		selectedAlbumArtistID := ""
		if track.Album != nil && len(track.Album.Artists) > 0 && track.Album.Artists[0].Artist != nil {
			selectedAlbumArtistID = track.Album.Artists[0].Artist.ID
			// If the artist doesn't have an ID, it should have been assigned a temporary one above
		}

		// Create map of selected artist IDs for template
		selectedArtistIDs := make(map[string]bool)
		for _, artistRole := range track.Artists {
			if artistRole.Artist != nil && artistRole.Artist.ID != "" {
				selectedArtistIDs[artistRole.Artist.ID] = true
			}
		}

		// Get provider colors
		providerColors := h.getProviderColors(providerName)

		// Check if request is HTMX or full page
		if c.Get("HX-Request") == "true" {
			return c.Render("sections/tag", fiber.Map{
				"Track":                 track,
				"Artists":               artists,
				"Albums":                albums,
				"FetchError":            "err",
				"ProviderColors":        providerColors,
				"SelectedAlbumArtistID": selectedAlbumArtistID,
				"SelectedArtistIDs":     selectedArtistIDs,
				"EnabledProviders":      h.service.GetEnabledProviders(),
			})
		} else {
			return c.Render("main", fiber.Map{
				"Title":                 "Edit Tags",
				"Track":                 track,
				"IsTagEdit":             true,
				"Artists":               artists,
				"Albums":                albums,
				"FetchError":            "err",
				"ProviderColors":        providerColors,
				"SelectedAlbumArtistID": selectedAlbumArtistID,
				"SelectedArtistIDs":     selectedArtistIDs,
				"EnabledProviders":      h.service.GetEnabledProviders(),
			})
		}
	}

	// Match fetched album with existing albums
	if fetchedTrack.Album != nil {
		albumFound := false
		for _, album := range albums {
			if album.Title == fetchedTrack.Album.Title {
				// Keep the fetched album but use the database album's ID
				fetchedAlbum := fetchedTrack.Album
				fetchedAlbum.ID = album.ID
				// Preserve any additional data from database album if needed
				fetchedTrack.Album = fetchedAlbum
				albumFound = true
				break
			}
		}
		if !albumFound {
			// If album doesn't exist, add it to the list for selection
			// Generate a temporary ID for the new album
			fetchedTrack.Album.ID = "temp_" + fetchedTrack.Album.Title
			albums = append(albums, fetchedTrack.Album)
		}
	}

	// Merge and render
	track = h.mergeFetchedData(track, fetchedTrack)
	slog.Info("Metadata fetched successfully", "trackId", trackID, "provider", providerName, "fetchedTitle", fetchedTrack.Title)

	// Ensure fetched track's artists are included in the dropdown
	// Add track artists (include those without IDs for fetched data)
	for _, artistRole := range track.Artists {
		if artistRole.Artist != nil {
			artistID := artistRole.Artist.ID
			if artistID == "" {
				// Generate a temporary ID for artists without database IDs (for dropdown display)
				artistID = "temp_" + artistRole.Artist.Name
				artistRole.Artist.ID = artistID
			}
			if !artistMap[artistID] {
				artists = append(artists, artistRole.Artist)
				artistMap[artistID] = true
			}
		}
	}
	// Add album artists (include those without IDs for fetched data)
	if track.Album != nil {
		for _, artistRole := range track.Album.Artists {
			if artistRole.Artist != nil {
				artistID := artistRole.Artist.ID
				if artistID == "" {
					// Generate a temporary ID for artists without database IDs (for dropdown display)
					artistID = "temp_" + artistRole.Artist.Name
					artistRole.Artist.ID = artistID
				}
				if !artistMap[artistID] {
					artists = append(artists, artistRole.Artist)
					artistMap[artistID] = true
				}
			}
		}
	}

	// Update selected artist IDs with merged track
	selectedArtistIDs = make(map[string]bool)
	for _, artistRole := range track.Artists {
		if artistRole.Artist != nil && artistRole.Artist.ID != "" {
			selectedArtistIDs[artistRole.Artist.ID] = true
		}
	}

	// Ensure track has valid ID for template
	if track.ID == "" {
		track.ID = trackID
	}

	// Determine selected album artist ID for template
	selectedAlbumArtistID := ""
	if track.Album != nil && len(track.Album.Artists) > 0 && track.Album.Artists[0].Artist != nil {
		selectedAlbumArtistID = track.Album.Artists[0].Artist.ID
		// If the artist doesn't have an ID, it should have been assigned a temporary one above
	}

	// Get provider colors
	providerColors := h.getProviderColors(providerName)

	// Check if request is HTMX or full page
	if c.Get("HX-Request") == "true" {
		return c.Render("sections/tag", fiber.Map{
			"Track":                 track,
			"Artists":               artists,
			"Albums":                albums,
			"FromProvider":          providerName,
			"ProviderColors":        providerColors,
			"SelectedAlbumArtistID": selectedAlbumArtistID,
			"SelectedArtistIDs":     selectedArtistIDs,
			"EnabledProviders":      h.service.GetEnabledProviders(),
		})
	} else {
		return c.Render("main", fiber.Map{
			"Title":                 "Edit Tags",
			"Track":                 track,
			"IsTagEdit":             true,
			"Artists":               artists,
			"Albums":                albums,
			"FromProvider":          providerName,
			"ProviderColors":        providerColors,
			"SelectedAlbumArtistID": selectedAlbumArtistID,
			"SelectedArtistIDs":     selectedArtistIDs,
			"EnabledProviders":      h.service.GetEnabledProviders(),
		})
	}
}

// mergeFetchedData merges fetched metadata with existing track data
func (h *Handler) mergeFetchedData(existing, fetched *music.Track) *music.Track {
	// Preserve database-specific data
	result := *fetched
	result.ID = existing.ID // Preserve the database track ID

	// Preserve file-specific data
	result.Path = existing.Path
	result.Format = existing.Format
	result.SampleRate = existing.SampleRate
	result.BitDepth = existing.BitDepth
	result.Channels = existing.Channels
	result.Bitrate = existing.Bitrate

	// Use fetched album artists (since we're fetching new metadata)
	// Only preserve existing album artists if fetched album has no artists
	if result.Album != nil && len(result.Album.Artists) == 0 && existing.Album != nil && len(existing.Album.Artists) > 0 {
		result.Album.Artists = existing.Album.Artists
	}

	// Preserve existing lyrics if available (since they're stored in file, not DB)
	if existing.Metadata.Lyrics != "" && result.Metadata.Lyrics == "" {
		result.Metadata.Lyrics = existing.Metadata.Lyrics
	}

	return &result
}

// ModalData holds data for the search results modal
type ModalData struct {
	Tracks         []*music.Track
	ProviderName   string
	ProviderColors map[string]string
	TrackID        string
}

// SearchTracksFromProvider handles searching for tracks from a specific provider
func (h *Handler) SearchTracksFromProvider(c *fiber.Ctx) error {
	trackID := c.Params("trackId")
	providerName := c.Params("provider")

	if trackID == "" || providerName == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Track ID and provider name are required")
	}

	// Search for tracks
	tracks, err := h.service.SearchTracksForTrack(c.Context(), trackID, providerName)
	if err != nil {
		slog.Error("Failed to search tracks", "error", err, "trackId", trackID, "provider", providerName)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to search tracks")
	}

	// Get provider colors for styling
	providerColors := h.getProviderColors(providerName)

	// Render modal with search results
	return c.Render("tag/search_results_modal", ModalData{
		Tracks:         tracks,
		ProviderName:   providerName,
		ProviderColors: providerColors,
		TrackID:        trackID,
	})
}

// SelectTrackFromResults handles selecting a track from search results and applying its metadata
func (h *Handler) SelectTrackFromResults(c *fiber.Ctx) error {
	trackID := c.Params("trackId")
	selectedTrackIndex := c.QueryInt("index", -1)
	providerName := c.Params("provider")

	if trackID == "" || selectedTrackIndex == -1 || providerName == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Track ID, provider name, and track index are required")
	}

	// Get search results again (could be optimized with caching)
	tracks, err := h.service.SearchTracksForTrack(c.Context(), trackID, providerName)
	if err != nil {
		slog.Error("Failed to get search results", "error", err, "trackId", trackID, "provider", providerName)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to get search results")
	}

	if selectedTrackIndex < 0 || selectedTrackIndex >= len(tracks) {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid track index")
	}

	selectedTrack := tracks[selectedTrackIndex]

	// Create/find artists and album for the selected track only
	// Create/find track artists
	for j, artistRole := range selectedTrack.Artists {
		dbArtist, err := h.service.libraryRepo.FindOrCreateArtist(c.Context(), artistRole.Artist.Name)
		if err != nil {
			slog.Warn("Failed to find/create selected track artist", "artistName", artistRole.Artist.Name, "error", err)
			continue
		}
		selectedTrack.Artists[j].Artist = dbArtist
	}

	// Handle album if present
	if selectedTrack.Album != nil {
		// Create/find album artists
		for j, artistRole := range selectedTrack.Album.Artists {
			dbArtist, err := h.service.libraryRepo.FindOrCreateArtist(c.Context(), artistRole.Artist.Name)
			if err != nil {
				slog.Warn("Failed to find/create selected album artist", "artistName", artistRole.Artist.Name, "error", err)
				continue
			}
			selectedTrack.Album.Artists[j].Artist = dbArtist
		}

		// Find or create album using first album artist
		var albumArtist *music.Artist
		if len(selectedTrack.Album.Artists) > 0 {
			albumArtist = selectedTrack.Album.Artists[0].Artist
		}

		dbAlbum, err := h.service.libraryRepo.FindOrCreateAlbum(c.Context(), albumArtist, selectedTrack.Album.Title, selectedTrack.Metadata.Year)
		if err != nil {
			slog.Warn("Failed to find/create selected album", "albumTitle", selectedTrack.Album.Title, "error", err)
		} else {
			selectedTrack.Album = dbAlbum
		}
	}

	// Get current track data
	currentTrack, err := h.service.GetTrackForEditing(c.Context(), trackID)
	if err != nil {
		slog.Error("Failed to get current track", "error", err, "trackId", trackID)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to get current track")
	}

	// Merge selected track data with current track (preserve file-specific data)
	mergedTrack := h.mergeFetchedData(currentTrack, selectedTrack)

	// Get all artists and albums for dropdowns
	artists, err := h.service.libraryRepo.GetArtists(c.Context())
	if err != nil {
		artists = []*music.Artist{} // Continue with empty list
	}

	albums, err := h.service.libraryRepo.GetAlbums(c.Context())
	if err != nil {
		albums = []*music.Album{} // Continue with empty list
	}

	// Ensure track's artists are included in the dropdown
	artistMap := make(map[string]bool)
	for _, artist := range artists {
		artistMap[artist.ID] = true
	}
	for _, artistRole := range mergedTrack.Artists {
		if artistRole.Artist != nil {
			artistID := artistRole.Artist.ID
			if artistID == "" {
				artistID = "temp_" + artistRole.Artist.Name
				artistRole.Artist.ID = artistID
			}
			if !artistMap[artistID] {
				artists = append(artists, artistRole.Artist)
				artistMap[artistID] = true
			}
		}
	}
	if mergedTrack.Album != nil {
		for _, artistRole := range mergedTrack.Album.Artists {
			if artistRole.Artist != nil {
				artistID := artistRole.Artist.ID
				if artistID == "" {
					artistID = "temp_" + artistRole.Artist.Name
					artistRole.Artist.ID = artistID
				}
				if !artistMap[artistID] {
					artists = append(artists, artistRole.Artist)
					artistMap[artistID] = true
				}
			}
		}
	}

	// Create selected artist IDs map
	selectedArtistIDs := make(map[string]bool)
	for _, artistRole := range mergedTrack.Artists {
		if artistRole.Artist != nil && artistRole.Artist.ID != "" {
			selectedArtistIDs[artistRole.Artist.ID] = true
		}
	}

	// Determine selected album artist ID
	selectedAlbumArtistID := ""
	if mergedTrack.Album != nil && len(mergedTrack.Album.Artists) > 0 {
		selectedAlbumArtistID = mergedTrack.Album.Artists[0].Artist.ID
	}

	// Get provider colors
	providerColors := h.getProviderColors(providerName)

	// Render the updated form
	return c.Render("sections/tag", fiber.Map{
		"Track":                 mergedTrack,
		"Artists":               artists,
		"Albums":                albums,
		"SelectedAlbumArtistID": selectedAlbumArtistID,
		"SelectedArtistIDs":     selectedArtistIDs,
		"FromProvider":          providerName,
		"ProviderColors":        providerColors,
		"EnabledProviders":      h.service.GetEnabledProviders(),
	})
}

// UpdateTags handles the form submission to update track tags
func (h *Handler) UpdateTags(c *fiber.Ctx) error {
	slog.Info("UpdateTags handler called", "trackId", c.Params("trackId"))

	trackID := c.Params("trackId")
	if trackID == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Track ID is required")
	}

	// Parse form data
	formData := make(map[string]string)
	c.BodyParser(&formData)

	// Get form values
	if title := c.FormValue("title"); title != "" {
		formData["title"] = title
	}

	// Handle dropdown fields - strict selection only
	if artistIDs := c.FormValue("artist_ids"); artistIDs != "" {
		formData["artist_ids"] = artistIDs
	}
	if albumID := c.FormValue("album_id"); albumID != "" {
		formData["album_id"] = albumID
	}
	if albumArtistID := c.FormValue("album_artist_id"); albumArtistID != "" {
		formData["album_artist_id"] = albumArtistID
	}
	if year := c.FormValue("year"); year != "" {
		formData["year"] = year
	}
	if genre := c.FormValue("genre"); genre != "" {
		formData["genre"] = genre
	}
	if trackNumber := c.FormValue("track_number"); trackNumber != "" {
		formData["track_number"] = trackNumber
	}
	if discNumber := c.FormValue("disc_number"); discNumber != "" {
		formData["disc_number"] = discNumber
	}
	if isrc := c.FormValue("isrc"); isrc != "" {
		formData["isrc"] = isrc
	}
	if composer := c.FormValue("composer"); composer != "" {
		formData["composer"] = composer
	}
	if lyrics := c.FormValue("lyrics"); lyrics != "" {
		formData["lyrics"] = lyrics
	}
	if bpm := c.FormValue("bpm"); bpm != "" {
		formData["bpm"] = bpm
	}
	if gain := c.FormValue("gain"); gain != "" {
		formData["gain"] = gain
	}
	if titleVersion := c.FormValue("title_version"); titleVersion != "" {
		formData["title_version"] = titleVersion
	}

	slog.Debug("Parsed form data", "formData", formData)

	// Update the tags
	err := h.service.UpdateTrackTags(c.Context(), trackID, formData)
	if err != nil {
		slog.Error("Failed to update track tags", "error", err, "trackId", trackID)
		return c.Render("toast/toastErr", fiber.Map{
			"Msg": fmt.Sprintf("Failed to update tags: %v", err),
		})
	}

	// Return success toast
	return c.Render("toast/toastOk", fiber.Map{
		"Msg": "Tags updated successfully!",
	})
}
