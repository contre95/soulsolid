package playlists

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/contre95/soulsolid/src/features/hosting/respond"
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
		playlists = []*music.Playlist{}
	}

	return respond.Section(c, "playlists", fiber.Map{
		"Title":     "Playlists",
		"Playlists": playlists,
	})
}

// RenderPlaylist renders a single playlist page (HTML, HTMX-aware).
func (h *Handler) RenderPlaylist(c *fiber.Ctx) error {
	slog.Debug("RenderPlaylist handler called", "id", c.Params("id"))

	playlist, err := h.service.GetPlaylist(c.Context(), c.Params("id"))
	if err != nil {
		slog.Error("Error loading playlist", "error", err, "id", c.Params("id"))
		return respond.Err(c, fiber.StatusInternalServerError, "Failed to load playlist")
	}
	if playlist == nil {
		return respond.Err(c, fiber.StatusNotFound, "Playlist not found")
	}

	data := fiber.Map{
		"Title":    fmt.Sprintf("Playlist: %s", playlist.Name),
		"Playlist": playlist,
	}
	if c.Get("HX-Request") != "true" {
		return c.Render("main", data)
	}
	return c.Render("playlists/playlist", data)
}

// GetAllPlaylists returns all playlists as JSON.
func (h *Handler) GetAllPlaylists(c *fiber.Ctx) error {
	slog.Debug("GetAllPlaylists handler called")

	playlists, err := h.service.GetAllPlaylists(c.Context())
	if err != nil {
		slog.Error("Error loading playlists", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"playlists": playlists})
}

// GetPlaylist returns a single playlist as JSON.
func (h *Handler) GetPlaylist(c *fiber.Ctx) error {
	slog.Debug("GetPlaylist handler called", "id", c.Params("id"))

	playlist, err := h.service.GetPlaylist(c.Context(), c.Params("id"))
	if err != nil {
		slog.Error("Error loading playlist", "error", err, "id", c.Params("id"))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if playlist == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "playlist not found"})
	}
	return c.JSON(playlist)
}

// CreatePlaylist handles creating a new playlist.
func (h *Handler) CreatePlaylist(c *fiber.Ctx) error {
	slog.Debug("CreatePlaylist handler called")

	var req struct {
		Name        string `json:"name" form:"name"`
		Description string `json:"description" form:"description"`
	}
	if err := c.BodyParser(&req); err != nil {
		return respond.Err(c, fiber.StatusBadRequest, "Invalid request body")
	}
	if req.Name == "" {
		return respond.Err(c, fiber.StatusBadRequest, "Playlist name is required")
	}

	playlist, err := h.service.CreatePlaylist(c.Context(), req.Name, req.Description)
	if err != nil {
		slog.Error("Error creating playlist", "error", err)
		return respond.Err(c, fiber.StatusInternalServerError, "Failed to create playlist")
	}

	c.Set("HX-Trigger", "refreshPlaylists")
	if c.Get("HX-Request") == "true" {
		return c.Render("toast/toastOk", fiber.Map{"Msg": "Playlist created successfully"})
	}
	return c.Status(fiber.StatusCreated).JSON(playlist)
}

// UpdatePlaylist handles updating a playlist.
func (h *Handler) UpdatePlaylist(c *fiber.Ctx) error {
	slog.Debug("UpdatePlaylist handler called", "id", c.Params("id"))

	var req struct {
		Name        string `json:"name" form:"name"`
		Description string `json:"description" form:"description"`
	}
	if err := c.BodyParser(&req); err != nil {
		return respond.Err(c, fiber.StatusBadRequest, "Invalid request body")
	}
	if req.Name == "" {
		return respond.Err(c, fiber.StatusBadRequest, "Playlist name is required")
	}

	playlist, err := h.service.GetPlaylist(c.Context(), c.Params("id"))
	if err != nil {
		slog.Error("Error loading playlist for update", "error", err, "id", c.Params("id"))
		return respond.Err(c, fiber.StatusInternalServerError, "Failed to load playlist")
	}
	if playlist == nil {
		return respond.Err(c, fiber.StatusNotFound, "Playlist not found")
	}

	playlist.Name = req.Name
	playlist.Description = req.Description

	if err := h.service.UpdatePlaylist(c.Context(), playlist); err != nil {
		slog.Error("Error updating playlist", "error", err, "id", c.Params("id"))
		return respond.Err(c, fiber.StatusInternalServerError, "Failed to update playlist")
	}
	return respond.Ok(c, "Playlist updated successfully")
}

// DeletePlaylist handles deleting a playlist.
func (h *Handler) DeletePlaylist(c *fiber.Ctx) error {
	slog.Debug("DeletePlaylist handler called", "id", c.Params("id"))

	if err := h.service.DeletePlaylist(c.Context(), c.Params("id")); err != nil {
		slog.Error("Error deleting playlist", "error", err, "id", c.Params("id"))
		return respond.Err(c, fiber.StatusInternalServerError, "Failed to delete playlist")
	}

	c.Set("HX-Trigger", "refreshPlaylists")
	return respond.Ok(c, "Playlist deleted successfully")
}

// AddItemToPlaylist handles adding tracks, artists, or albums to a playlist.
func (h *Handler) AddItemToPlaylist(c *fiber.Ctx) error {
	var req struct {
		PlaylistID string `json:"playlist_id" form:"playlist_id"`
		ItemType   string `json:"item_type" form:"item_type"`
		ItemID     string `json:"item_id" form:"item_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return respond.Err(c, fiber.StatusBadRequest, "Invalid request body")
	}

	slog.Debug("AddItemToPlaylist handler called", "playlistID", req.PlaylistID, "itemType", req.ItemType, "itemID", req.ItemID)

	if req.PlaylistID == "" || req.ItemType == "" || req.ItemID == "" {
		slog.Error("AddItemToPlaylist: missing required parameters", "playlistID", req.PlaylistID, "itemType", req.ItemType, "itemID", req.ItemID)
		return respond.Err(c, fiber.StatusBadRequest, "Playlist ID, item type, and item ID are required")
	}

	if err := h.service.AddItemToPlaylist(c.Context(), req.PlaylistID, req.ItemType, req.ItemID); err != nil {
		slog.Error("Error adding item to playlist", "error", err, "playlistID", req.PlaylistID, "itemType", req.ItemType, "itemID", req.ItemID)
		return respond.Err(c, fiber.StatusInternalServerError, "Failed to add item to playlist")
	}

	var itemName string
	switch req.ItemType {
	case "track":
		if track, err := h.service.library.GetTrack(c.Context(), req.ItemID); err == nil && track != nil {
			itemName = track.Title
		}
	case "artist":
		if artist, err := h.service.library.GetArtist(c.Context(), req.ItemID); err == nil && artist != nil {
			itemName = artist.Name
		}
	case "album":
		if album, err := h.service.library.GetAlbum(c.Context(), req.ItemID); err == nil && album != nil {
			itemName = album.Title
		}
	}

	var msg string
	switch req.ItemType {
	case "track":
		msg = fmt.Sprintf("Track '%s' added to playlist", itemName)
	case "artist":
		msg = fmt.Sprintf("All tracks by '%s' added to playlist", itemName)
	case "album":
		msg = fmt.Sprintf("All tracks from '%s' added to playlist", itemName)
	default:
		msg = "Item added to playlist"
	}

	slog.Info("Item successfully added to playlist", "playlistID", req.PlaylistID, "itemType", req.ItemType, "itemID", req.ItemID)

	c.Set("HX-Trigger", "playlistUpdated")
	return respond.Ok(c, msg)
}

// RemoveTrackFromPlaylist handles removing a track from a playlist.
func (h *Handler) RemoveTrackFromPlaylist(c *fiber.Ctx) error {
	slog.Debug("RemoveTrackFromPlaylist handler called")

	playlistID := c.Params("playlistId")
	trackID := c.Params("trackId")

	if playlistID == "" || trackID == "" {
		return respond.Err(c, fiber.StatusBadRequest, "Playlist ID and Track ID are required")
	}

	if err := h.service.RemoveTrackFromPlaylist(c.Context(), playlistID, trackID); err != nil {
		slog.Error("Error removing track from playlist", "error", err, "playlistID", playlistID, "trackID", trackID)
		return respond.Err(c, fiber.StatusInternalServerError, "Failed to remove track from playlist")
	}

	c.Set("HX-Trigger", "playlistUpdated")
	return respond.Ok(c, "Track removed from playlist")
}

// GetPlaylistCreationModal returns the create playlist modal.
func (h *Handler) GetPlaylistCreationModal(c *fiber.Ctx) error {
	slog.Debug("GetCreatePlaylistModal handler called")
	return c.Render("playlists/create_playlist_modal", nil)
}

// GetPlaylistsForItem returns the add-to-playlist modal for a given item.
func (h *Handler) GetPlaylistsForItem(c *fiber.Ctx) error {
	itemType := c.Params("type")
	itemID := c.Params("id")

	slog.Debug("GetPlaylistsForItem handler called", "type", itemType, "id", itemID)

	playlists, err := h.service.GetAllPlaylists(c.Context())
	if err != nil {
		slog.Error("Error loading playlists", "error", err, "type", itemType, "id", itemID)
		return respond.Err(c, fiber.StatusInternalServerError, "Failed to load playlists")
	}

	var itemName string
	switch itemType {
	case "track":
		track, err := h.service.library.GetTrack(c.Context(), itemID)
		if err != nil || track == nil {
			return respond.Err(c, fiber.StatusNotFound, "Track not found")
		}
		itemName = track.Title
	case "artist":
		artist, err := h.service.library.GetArtist(c.Context(), itemID)
		if err != nil || artist == nil {
			return respond.Err(c, fiber.StatusNotFound, "Artist not found")
		}
		itemName = artist.Name
	case "album":
		album, err := h.service.library.GetAlbum(c.Context(), itemID)
		if err != nil || album == nil {
			return respond.Err(c, fiber.StatusNotFound, "Album not found")
		}
		itemName = album.Title
	default:
		return respond.Err(c, fiber.StatusBadRequest, "Invalid item type")
	}

	return c.Render("playlists/add_to_playlist_modal", fiber.Map{
		"Playlists": playlists,
		"ItemType":  itemType,
		"ItemID":    itemID,
		"ItemName":  itemName,
	})
}

// ExportM3U handles exporting a playlist to an M3U file.
func (h *Handler) ExportM3U(c *fiber.Ctx) error {
	slog.Debug("ExportM3U handler called", "id", c.Params("id"))

	playlist, err := h.service.GetPlaylist(c.Context(), c.Params("id"))
	if err != nil {
		slog.Error("Error loading playlist for export", "error", err, "id", c.Params("id"))
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to load playlist")
	}
	if playlist == nil {
		return c.Status(fiber.StatusNotFound).SendString("Playlist not found")
	}

	var builder strings.Builder
	builder.WriteString("#EXTM3U\n")

	for _, track := range playlist.Tracks {
		duration := track.Metadata.Duration
		artists := make([]string, len(track.Artists))
		for i, ar := range track.Artists {
			if ar.Artist != nil {
				artists[i] = ar.Artist.Name
			}
		}
		artistStr := strings.Join(artists, ", ")
		builder.WriteString(fmt.Sprintf("#EXTINF:%d,%s - %s\n", duration, artistStr, track.Title))
		builder.WriteString(track.Path + "\n")
	}

	filename := fmt.Sprintf("%s.m3u", playlist.Name)
	c.Set("Content-Type", "text/plain")
	c.Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", filename))
	return c.SendString(builder.String())
}
