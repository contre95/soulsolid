package playlists

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/contre95/soulsolid/src/music"
	"github.com/gofiber/fiber/v2"
)

// Handler is the handler for the playlists feature.
type Handler struct {
	service *Service
}

// NewHandler creates a new handler for the playlists feature.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RenderPlaylistsSection renders the playlists page.
func (h *Handler) RenderPlaylistsSection(c *fiber.Ctx) error {
	slog.Debug("RenderPlaylistsSection handler called")

	playlists, err := h.service.GetAllPlaylists(c.Context())
	if err != nil {
		slog.Error("Error loading playlists", "error", err)
		playlists = []*music.Playlist{} // Continue with empty list
	}

	data := fiber.Map{
		"Title":     "Playlists",
		"Playlists": playlists,
	}
	if c.Get("HX-Request") != "true" {
		data["Section"] = "playlists"
		return c.Render("main", data)
	}
	return c.Render("sections/playlists", data)
}

// GetPlaylist renders a single playlist page.
func (h *Handler) GetPlaylist(c *fiber.Ctx) error {
	slog.Debug("GetPlaylist handler called", "id", c.Params("id"))

	playlist, err := h.service.GetPlaylist(c.Context(), c.Params("id"))
	if err != nil {
		slog.Error("Error loading playlist", "error", err, "id", c.Params("id"))
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to load playlist")
	}
	if playlist == nil {
		return c.Status(fiber.StatusNotFound).SendString("Playlist not found")
	}

	data := fiber.Map{
		"Title":    fmt.Sprintf("Playlist: %s", playlist.Name),
		"Playlist": playlist,
	}

	if c.Get("HX-Request") != "true" {
		data["Section"] = "playlists"
		return c.Render("main", data)
	}
	return c.Render("playlists/playlist", data)
}

// CreatePlaylist handles creating a new playlist.
func (h *Handler) CreatePlaylist(c *fiber.Ctx) error {
	slog.Debug("CreatePlaylist handler called")

	name := c.FormValue("name")
	description := c.FormValue("description")

	if name == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Playlist name is required")
	}

	_, err := h.service.CreatePlaylist(c.Context(), name, description)
	if err != nil {
		slog.Error("Error creating playlist", "error", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to create playlist")
	}

	// Return success toast
	return c.Render("toast/toastOk", fiber.Map{"Msg": "Playlist created successfully"})
}

// UpdatePlaylist handles updating a playlist.
func (h *Handler) UpdatePlaylist(c *fiber.Ctx) error {
	slog.Debug("UpdatePlaylist handler called", "id", c.Params("id"))

	playlistID := c.Params("id")
	name := c.FormValue("name")
	description := c.FormValue("description")

	if name == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Playlist name is required")
	}

	playlist, err := h.service.GetPlaylist(c.Context(), playlistID)
	if err != nil {
		slog.Error("Error loading playlist for update", "error", err, "id", playlistID)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to load playlist")
	}
	if playlist == nil {
		return c.Status(fiber.StatusNotFound).SendString("Playlist not found")
	}

	playlist.Name = name
	playlist.Description = description

	err = h.service.UpdatePlaylist(c.Context(), playlist)
	if err != nil {
		slog.Error("Error updating playlist", "error", err, "id", playlistID)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to update playlist")
	}

	// Return success toast
	return c.Render("toast/toastOk", fiber.Map{"Msg": "Playlist updated successfully"})
}

// DeletePlaylist handles deleting a playlist.
func (h *Handler) DeletePlaylist(c *fiber.Ctx) error {
	slog.Debug("DeletePlaylist handler called", "id", c.Params("id"))

	err := h.service.DeletePlaylist(c.Context(), c.Params("id"))
	if err != nil {
		slog.Error("Error deleting playlist", "error", err, "id", c.Params("id"))
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to delete playlist")
	}

	// Return success toast
	return c.Render("toast/toastOk", fiber.Map{"Msg": "Playlist deleted successfully"})
}

// AddTrackToPlaylist handles adding a track to a playlist.
func (h *Handler) AddTrackToPlaylist(c *fiber.Ctx) error {
	playlistID := c.FormValue("playlist_id")
	trackID := c.FormValue("track_id")

	slog.Debug("AddTrackToPlaylist handler called", "playlistID", playlistID, "trackID", trackID)

	if playlistID == "" || trackID == "" {
		slog.Error("AddTrackToPlaylist: missing required parameters", "playlistID", playlistID, "trackID", trackID)
		return c.Status(fiber.StatusBadRequest).SendString("Playlist ID and Track ID are required")
	}

	err := h.service.AddTrackToPlaylist(c.Context(), playlistID, trackID)
	if err != nil {
		slog.Error("Error adding track to playlist", "error", err, "playlistID", playlistID, "trackID", trackID)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to add track to playlist")
	}

	slog.Info("Track successfully added to playlist", "playlistID", playlistID, "trackID", trackID)

	// Trigger playlist refresh and return success toast
	c.Set("HX-Trigger", "playlistUpdated")
	return c.Render("toast/toastOk", fiber.Map{"Msg": "Track added to playlist"})
}

// RemoveTrackFromPlaylist handles removing a track from a playlist.
func (h *Handler) RemoveTrackFromPlaylist(c *fiber.Ctx) error {
	slog.Debug("RemoveTrackFromPlaylist handler called")

	playlistID := c.Params("playlistId")
	trackID := c.Params("trackId")

	if playlistID == "" || trackID == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Playlist ID and Track ID are required")
	}

	err := h.service.RemoveTrackFromPlaylist(c.Context(), playlistID, trackID)
	if err != nil {
		slog.Error("Error removing track from playlist", "error", err, "playlistID", playlistID, "trackID", trackID)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to remove track from playlist")
	}

	// Trigger playlist refresh and return success toast
	c.Set("HX-Trigger", "playlistUpdated")
	return c.Render("toast/toastOk", fiber.Map{"Msg": "Track removed from playlist"})
}

// GetPlaylistsForTrack returns all playlists containing a specific track.
func (h *Handler) GetPlaylistsForTrack(c *fiber.Ctx) error {
	slog.Debug("GetPlaylistsForTrack handler called", "trackID", c.Params("trackId"))

	playlists, err := h.service.GetAllPlaylists(c.Context())
	if err != nil {
		slog.Error("Error loading playlists for track", "error", err, "trackID", c.Params("trackId"))
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to load playlists")
	}

	data := fiber.Map{
		"Playlists": playlists,
		"TrackID":   c.Params("trackId"),
	}

	return c.Render("playlists/add_to_playlist_modal", data)
}

// ImportM3U handles importing a playlist from an M3U file.
func (h *Handler) ImportM3U(c *fiber.Ctx) error {
	slog.Debug("ImportM3U handler called")

	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("No file uploaded")
	}

	playlistName := c.FormValue("name")
	if playlistName == "" {
		playlistName = file.Filename
	}

	// Save uploaded file temporarily
	tempPath := fmt.Sprintf("/tmp/%s", file.Filename)
	err = c.SaveFile(file, tempPath)
	if err != nil {
		slog.Error("Error saving uploaded file", "error", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to save uploaded file")
	}
	defer func() {
		// Clean up temp file
		if removeErr := os.Remove(tempPath); removeErr != nil {
			slog.Warn("Failed to clean up temp file", "path", tempPath, "error", removeErr)
		}
	}()

	playlist, err := h.service.ImportM3U(c.Context(), tempPath, playlistName)
	if err != nil {
		slog.Error("Error importing M3U", "error", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to import playlist")
	}

	// Return success toast
	return c.Render("toast/toastOk", fiber.Map{"Msg": fmt.Sprintf("Playlist '%s' imported successfully", playlist.Name)})
}

// ExportM3U handles exporting a playlist to an M3U file.
func (h *Handler) ExportM3U(c *fiber.Ctx) error {
	slog.Debug("ExportM3U handler called", "id", c.Params("id"))

	playlistID := c.Params("id")

	// Get playlist name for filename
	playlist, err := h.service.GetPlaylist(c.Context(), playlistID)
	if err != nil {
		slog.Error("Error loading playlist for export", "error", err, "id", playlistID)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to load playlist")
	}
	if playlist == nil {
		return c.Status(fiber.StatusNotFound).SendString("Playlist not found")
	}

	filename := fmt.Sprintf("%s.m3u", playlist.Name)
	tempPath := fmt.Sprintf("/tmp/%s", filename)

	err = h.service.ExportM3U(c.Context(), playlistID, tempPath)
	if err != nil {
		slog.Error("Error exporting M3U", "error", err, "id", playlistID)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to export playlist")
	}

	return c.Download(tempPath, filename)
}
