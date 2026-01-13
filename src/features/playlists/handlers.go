package playlists

import (
	"fmt"
	"log/slog"
	"strings"

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
		// For direct navigation to specific playlist, render main with Playlist data (no Section set)
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

	// Trigger playlist refresh and return success toast
	c.Set("HX-Trigger", "refreshPlaylists")
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

	// Trigger playlist refresh and return success toast
	c.Set("HX-Trigger", "refreshPlaylists")
	return c.Render("toast/toastOk", fiber.Map{"Msg": "Playlist deleted successfully"})
}

// AddItemToPlaylist handles adding tracks, artists, or albums to a playlist.
func (h *Handler) AddItemToPlaylist(c *fiber.Ctx) error {
	playlistID := c.FormValue("playlist_id")
	itemType := c.FormValue("item_type")
	itemID := c.FormValue("item_id")

	slog.Debug("AddItemToPlaylist handler called", "playlistID", playlistID, "itemType", itemType, "itemID", itemID)

	if playlistID == "" || itemType == "" || itemID == "" {
		slog.Error("AddItemToPlaylist: missing required parameters", "playlistID", playlistID, "itemType", itemType, "itemID", itemID)
		return c.Status(fiber.StatusBadRequest).SendString("Playlist ID, item type, and item ID are required")
	}

	err := h.service.AddItemToPlaylist(c.Context(), playlistID, itemType, itemID)
	if err != nil {
		slog.Error("Error adding item to playlist", "error", err, "playlistID", playlistID, "itemType", itemType, "itemID", itemID)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to add item to playlist")
	}

	// Get item name for success message
	var itemName string
	switch itemType {
	case "track":
		if track, err := h.service.library.GetTrack(c.Context(), itemID); err == nil && track != nil {
			itemName = track.Title
		}
	case "artist":
		if artist, err := h.service.library.GetArtist(c.Context(), itemID); err == nil && artist != nil {
			itemName = artist.Name
		}
	case "album":
		if album, err := h.service.library.GetAlbum(c.Context(), itemID); err == nil && album != nil {
			itemName = album.Title
		}
	}

	var successMsg string
	switch itemType {
	case "track":
		successMsg = fmt.Sprintf("Track '%s' added to playlist", itemName)
	case "artist":
		successMsg = fmt.Sprintf("All tracks by '%s' added to playlist", itemName)
	case "album":
		successMsg = fmt.Sprintf("All tracks from '%s' added to playlist", itemName)
	default:
		successMsg = "Item added to playlist"
	}

	slog.Info("Item successfully added to playlist", "playlistID", playlistID, "itemType", itemType, "itemID", itemID)

	// Trigger playlist refresh and return success toast
	c.Set("HX-Trigger", "playlistUpdated")
	return c.Render("toast/toastOk", fiber.Map{"Msg": successMsg})
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

// GetPlaylistCreationModal returns the create playlist modal.
func (h *Handler) GetPlaylistCreationModal(c *fiber.Ctx) error {
	slog.Debug("GetCreatePlaylistModal handler called")

	return c.Render("playlists/create_playlist_modal", nil)
}

// GetPlaylistsForItem returns playlists for adding tracks, artists, or albums.
func (h *Handler) GetPlaylistsForItem(c *fiber.Ctx) error {
	itemType := c.Params("type")
	itemID := c.Params("id")

	slog.Debug("GetPlaylistsForItem handler called", "type", itemType, "id", itemID)

	playlists, err := h.service.GetAllPlaylists(c.Context())
	if err != nil {
		slog.Error("Error loading playlists", "error", err, "type", itemType, "id", itemID)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to load playlists")
	}

	// Get item name for display
	var itemName string
	switch itemType {
	case "track":
		track, err := h.service.library.GetTrack(c.Context(), itemID)
		if err != nil || track == nil {
			return c.Status(fiber.StatusNotFound).SendString("Track not found")
		}
		itemName = track.Title
	case "artist":
		artist, err := h.service.library.GetArtist(c.Context(), itemID)
		if err != nil || artist == nil {
			return c.Status(fiber.StatusNotFound).SendString("Artist not found")
		}
		itemName = artist.Name
	case "album":
		album, err := h.service.library.GetAlbum(c.Context(), itemID)
		if err != nil || album == nil {
			return c.Status(fiber.StatusNotFound).SendString("Album not found")
		}
		itemName = album.Title
	default:
		return c.Status(fiber.StatusBadRequest).SendString("Invalid item type")
	}

	data := fiber.Map{
		"Playlists": playlists,
		"ItemType":  itemType,
		"ItemID":    itemID,
		"ItemName":  itemName,
	}

	return c.Render("playlists/add_to_playlist_modal", data)
}

// ExportM3U handles exporting a playlist to an M3U file.
func (h *Handler) ExportM3U(c *fiber.Ctx) error {
	slog.Debug("ExportM3U handler called", "id", c.Params("id"))

	playlistID := c.Params("id")

	// Get playlist
	playlist, err := h.service.GetPlaylist(c.Context(), playlistID)
	if err != nil {
		slog.Error("Error loading playlist for export", "error", err, "id", playlistID)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to load playlist")
	}
	if playlist == nil {
		return c.Status(fiber.StatusNotFound).SendString("Playlist not found")
	}

	// Generate M3U content
	var builder strings.Builder
	builder.WriteString("#EXTM3U\n")

	for _, track := range playlist.Tracks {
		// Write extended M3U info
		duration := track.Metadata.Duration
		artists := make([]string, len(track.Artists))
		for i, ar := range track.Artists {
			if ar.Artist != nil {
				artists[i] = ar.Artist.Name
			}
		}
		artistStr := strings.Join(artists, ", ")

		builder.WriteString(fmt.Sprintf("#EXTINF:%d,%s - %s\n", duration, artistStr, track.Title))

		// Write file path
		builder.WriteString(track.Path + "\n")
	}

	m3uContent := builder.String()
	filename := fmt.Sprintf("%s.m3u", playlist.Name)

	// Set headers for inline display in new tab
	c.Set("Content-Type", "text/plain")
	c.Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", filename))

	return c.SendString(m3uContent)
}

// SyncPlaylistToPlayers handles syncing a playlist to external players.
func (h *Handler) SyncPlaylistToPlayers(c *fiber.Ctx) error {
	playlistID := c.Params("id")

	slog.Debug("SyncPlaylistToPlayers handler called", "playlistID", playlistID)

	err := h.service.SyncPlaylistToPlayers(c.Context(), playlistID)
	if err != nil {
		slog.Error("Error syncing playlist to players", "error", err, "playlistID", playlistID)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to sync playlist to players")
	}

	// Return success toast
	return c.Render("toast/toastOk", fiber.Map{"Msg": "Playlist synced to external players"})
}

// DeletePlaylistFromPlayers handles deleting a playlist from external players.
func (h *Handler) DeletePlaylistFromPlayers(c *fiber.Ctx) error {
	playlistID := c.Params("id")

	slog.Debug("DeletePlaylistFromPlayers handler called", "playlistID", playlistID)

	err := h.service.DeletePlaylistFromPlayers(c.Context(), playlistID)
	if err != nil {
		slog.Error("Error deleting playlist from players", "error", err, "playlistID", playlistID)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to delete playlist from players")
	}

	// Return success toast
	return c.Render("toast/toastOk", fiber.Map{"Msg": "Playlist deleted from external players"})
}
