package playlists

import (
	"fmt"
	"log/slog"

	"github.com/gofiber/fiber/v2"
)

// Handler handles HTTP requests for playlists
type Handler struct {
	service *Service
}

// NewHandler creates a new playlists handler
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// RenderPlaylistsSection renders the playlists page
func (h *Handler) RenderPlaylistsSection(c *fiber.Ctx) error {
	slog.Debug("RenderPlaylistsSection handler called")

	data := fiber.Map{
		"Title": "Playlists",
	}

	if c.Get("HX-Request") != "true" {
		data["Section"] = "playlists"
		return c.Render("main", data)
	}
	return c.Render("sections/playlists", data)
}

// RenderCreatePlaylistForm renders the create playlist form
func (h *Handler) RenderCreatePlaylistForm(c *fiber.Ctx) error {
	data := fiber.Map{
		"Title": "Create Playlist",
	}

	if c.Get("HX-Request") != "true" {
		data["Section"] = "playlists"
		return c.Render("main", data)
	}
	return c.Render("playlists/create", data)
}

// RenderAddToPlaylistModal renders the modal for adding a track to playlists
func (h *Handler) RenderAddToPlaylistModal(c *fiber.Ctx) error {
	trackID := c.Query("track_id")
	if trackID == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Track ID is required")
	}

	playlists, err := h.service.playlistService.GetPlaylists(c.Context())
	if err != nil {
		slog.Error("Failed to get playlists for modal", "error", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to load playlists")
	}

	return c.Render("playlists/add_modal", fiber.Map{
		"TrackID":   trackID,
		"Playlists": playlists,
	})
}

// GetPlaylists returns a list of playlists
func (h *Handler) GetPlaylists(c *fiber.Ctx) error {
	playlists, err := h.service.playlistService.GetPlaylists(c.Context())
	if err != nil {
		slog.Error("Failed to get playlists", "error", err)
		if c.Get("HX-Request") == "true" {
			return c.Render("toast/toastErr", fiber.Map{
				"Msg": "Failed to load playlists",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load playlists",
		})
	}

	if c.Get("HX-Request") == "true" {
		return c.Render("playlists/list", fiber.Map{
			"Playlists": playlists,
		})
	}

	return c.JSON(fiber.Map{
		"playlists": playlists,
	})
}

// CreatePlaylist creates a new playlist
func (h *Handler) CreatePlaylist(c *fiber.Ctx) error {
	name := c.FormValue("name")
	if name == "" {
		return h.renderError(c, "Playlist name is required")
	}

	description := c.FormValue("description")

	playlist, err := h.service.playlistService.CreatePlaylist(c.Context(), name, description)
	if err != nil {
		slog.Error("Failed to create playlist", "error", err, "name", name)
		return h.renderError(c, "Failed to create playlist")
	}

	slog.Info("Playlist created", "id", playlist.ID, "name", playlist.Name)

	if c.Get("HX-Request") == "true" {
		return c.Render("toast/toastOk", fiber.Map{
			"Msg": fmt.Sprintf("Playlist '%s' created successfully!", playlist.Name),
		})
	}

	return c.JSON(fiber.Map{
		"playlist": playlist,
	})
}

// GetPlaylist returns a specific playlist with its tracks
func (h *Handler) GetPlaylist(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return h.renderError(c, "Playlist ID is required")
	}

	playlist, err := h.service.playlistService.GetPlaylist(c.Context(), id)
	if err != nil {
		slog.Error("Failed to get playlist", "error", err, "id", id)
		return h.renderError(c, "Failed to load playlist")
	}

	if playlist == nil {
		return h.renderError(c, "Playlist not found")
	}

	if c.Get("HX-Request") == "true" {
		return c.Render("playlists/detail", fiber.Map{
			"Playlist": playlist,
		})
	}

	return c.JSON(fiber.Map{
		"playlist": playlist,
	})
}

// UpdatePlaylist updates a playlist
func (h *Handler) UpdatePlaylist(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return h.renderError(c, "Playlist ID is required")
	}

	name := c.FormValue("name")
	if name == "" {
		return h.renderError(c, "Playlist name is required")
	}

	description := c.FormValue("description")

	playlist, err := h.service.playlistService.GetPlaylist(c.Context(), id)
	if err != nil {
		slog.Error("Failed to get playlist for update", "error", err, "id", id)
		return h.renderError(c, "Failed to load playlist")
	}

	if playlist == nil {
		return h.renderError(c, "Playlist not found")
	}

	playlist.Name = name
	playlist.Description = description

	err = h.service.playlistService.UpdatePlaylist(c.Context(), playlist)
	if err != nil {
		slog.Error("Failed to update playlist", "error", err, "id", id)
		return h.renderError(c, "Failed to update playlist")
	}

	if c.Get("HX-Request") == "true" {
		return c.Render("toast/toastOk", fiber.Map{
			"Msg": "Playlist updated successfully!",
		})
	}

	return c.JSON(fiber.Map{
		"playlist": playlist,
	})
}

// DeletePlaylist deletes a playlist
func (h *Handler) DeletePlaylist(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return h.renderError(c, "Playlist ID is required")
	}

	err := h.service.playlistService.DeletePlaylist(c.Context(), id)
	if err != nil {
		slog.Error("Failed to delete playlist", "error", err, "id", id)
		return h.renderError(c, "Failed to delete playlist")
	}

	slog.Info("Playlist deleted", "id", id)

	if c.Get("HX-Request") == "true" {
		return c.Render("toast/toastOk", fiber.Map{
			"Msg": "Playlist deleted successfully!",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
	})
}

// AddTrackToPlaylist adds a track to a playlist
func (h *Handler) AddTrackToPlaylist(c *fiber.Ctx) error {
	playlistID := c.Params("id")
	if playlistID == "" {
		return h.renderError(c, "Playlist ID is required")
	}

	trackID := c.FormValue("track_id")
	if trackID == "" {
		return h.renderError(c, "Track ID is required")
	}

	err := h.service.playlistService.AddTrackToPlaylist(c.Context(), playlistID, trackID)
	if err != nil {
		slog.Error("Failed to add track to playlist", "error", err, "playlistID", playlistID, "trackID", trackID)
		return h.renderError(c, "Failed to add track to playlist")
	}

	if c.Get("HX-Request") == "true" {
		return c.Render("toast/toastOk", fiber.Map{
			"Msg": "Track added to playlist!",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
	})
}

// RemoveTrackFromPlaylist removes a track from a playlist
func (h *Handler) RemoveTrackFromPlaylist(c *fiber.Ctx) error {
	playlistID := c.Params("playlistId")
	if playlistID == "" {
		return h.renderError(c, "Playlist ID is required")
	}

	trackID := c.Params("trackId")
	if trackID == "" {
		return h.renderError(c, "Track ID is required")
	}

	err := h.service.playlistService.RemoveTrackFromPlaylist(c.Context(), playlistID, trackID)
	if err != nil {
		slog.Error("Failed to remove track from playlist", "error", err, "playlistID", playlistID, "trackID", trackID)
		return h.renderError(c, "Failed to remove track from playlist")
	}

	if c.Get("HX-Request") == "true" {
		return c.Render("toast/toastOk", fiber.Map{
			"Msg": "Track removed from playlist!",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
	})
}

// ImportM3U imports an M3U playlist
func (h *Handler) ImportM3U(c *fiber.Ctx) error {
	name := c.FormValue("name")
	if name == "" {
		return h.renderError(c, "Playlist name is required")
	}

	description := c.FormValue("description")
	content := c.FormValue("content")
	if content == "" {
		return h.renderError(c, "M3U content is required")
	}

	playlist, err := h.service.m3uParser.ImportM3U(c.Context(), name, description, content)
	if err != nil {
		slog.Error("Failed to import M3U", "error", err, "name", name)
		return h.renderError(c, "Failed to import M3U playlist")
	}

	if c.Get("HX-Request") == "true" {
		return c.Render("toast/toastOk", fiber.Map{
			"Msg": fmt.Sprintf("M3U playlist '%s' imported with %d tracks!", playlist.Name, len(playlist.Tracks)),
		})
	}

	return c.JSON(fiber.Map{
		"playlist": playlist,
	})
}

// ExportM3U exports a playlist as M3U
func (h *Handler) ExportM3U(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return h.renderError(c, "Playlist ID is required")
	}

	tracks, err := h.service.playlistService.GetPlaylistTracks(c.Context(), id)
	if err != nil {
		slog.Error("Failed to get playlist tracks for export", "error", err, "id", id)
		return h.renderError(c, "Failed to export playlist")
	}

	m3uContent, err := h.service.m3uParser.GenerateM3U(tracks)
	if err != nil {
		slog.Error("Failed to generate M3U", "error", err, "id", id)
		return h.renderError(c, "Failed to generate M3U content")
	}

	c.Set("Content-Type", "audio/x-mpegurl")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"playlist_%s.m3u\"", id))

	return c.SendString(m3uContent)
}

// renderError renders an error response
func (h *Handler) renderError(c *fiber.Ctx, message string) error {
	if c.Get("HX-Request") == "true" {
		return c.Render("toast/toastErr", fiber.Map{
			"Msg": message,
		})
	}
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"error": message,
	})
}
