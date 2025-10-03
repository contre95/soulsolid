package downloading

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/contre95/soulsolid/src/features/jobs"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// TelegramNotifier handles Telegram notifications for downloads
type TelegramNotifier struct {
	jobService jobs.JobService
}

// TelegramHandler handles Telegram commands for downloading
type TelegramHandler struct {
	service *Service
}

// NewTelegramHandler creates a new Telegram handler for downloading
func NewTelegramHandler(service *Service) *TelegramHandler {
	return &TelegramHandler{
		service: service,
	}
}

// HandleCommand processes Telegram commands for downloading
func (h *TelegramHandler) HandleCommand(bot *tgbotapi.BotAPI, chatID int64, command string, args string) error {
	switch command {
	case "search":
		return h.handleSearch(bot, chatID, args)
	case "download":
		return h.handleDownload(bot, chatID, args)
	default:
		msg := tgbotapi.NewMessage(chatID, "❌ Unknown download command. Use /search <query> or /download <type> <id>")
		bot.Send(msg)
		return nil
	}
}

// handleSearch handles search commands
func (h *TelegramHandler) handleSearch(bot *tgbotapi.BotAPI, chatID int64, query string) error {
	if strings.TrimSpace(query) == "" {
		msg := tgbotapi.NewMessage(chatID, "❌ Please provide a search query.\n\n*Usage:* `/search <tracks|albums|artists> <query>`")
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	// Detect type filter
	parts := strings.Fields(query)
	searchType := "all"
	if len(parts) > 1 {
		switch strings.ToLower(parts[0]) {
		case "track", "tracks":
			searchType = "tracks"
			query = strings.Join(parts[1:], " ")
		case "album", "albums":
			searchType = "albums"
			query = strings.Join(parts[1:], " ")
		case "artist", "artists":
			searchType = "artists"
			query = strings.Join(parts[1:], " ")
		}
	}

	hasResults := false

	// --- TRACKS ---
	if searchType == "all" || searchType == "tracks" {
		tracks, err := h.service.SearchTracks("dummy", query, 5)
		if err != nil {
			slog.Error("Failed to search tracks", "error", err)
		}
		if len(tracks) > 0 {
			hasResults = true
			total := len(tracks)
			if total > 5 {
				tracks = tracks[:5]
			}

			message := fmt.Sprintf("🎵 *Top Track Results for:* `%s`\n📊 Showing %d of %d\n\n", query, len(tracks), total)
			for i, track := range tracks {
				artistName := "Unknown Artist"
				if len(track.Artists) > 0 && track.Artists[0].Artist != nil {
					artistName = track.Artists[0].Artist.Name
				}
				message += fmt.Sprintf(
					"*%d.* %s — %s\n🆔 `%s`\n⬇️ `/download track %s`\n\n",
					i+1, track.Title, artistName, track.ID, track.ID,
				)
			}

			msg := tgbotapi.NewMessage(chatID, message+"──────────────────────")
			msg.ParseMode = tgbotapi.ModeMarkdown
			bot.Send(msg)
		}
	}

	// --- ALBUMS ---
	if searchType == "all" || searchType == "albums" {
		albums, err := h.service.SearchAlbums("dummy", query, 5)
		if err != nil {
			slog.Error("Failed to search albums", "error", err)
		}
		if len(albums) > 0 {
			hasResults = true
			total := len(albums)
			if total > 3 {
				albums = albums[:3]
			}

			summary := fmt.Sprintf("💿 *Album Results for:* `%s`\n📊 Showing %d of %d\n──────────────────────", query, len(albums), total)
			summaryMsg := tgbotapi.NewMessage(chatID, summary)
			summaryMsg.ParseMode = tgbotapi.ModeMarkdown
			bot.Send(summaryMsg)

			for _, album := range albums {
				artistName := "Unknown Artist"
				if len(album.Artists) > 0 && album.Artists[0].Artist != nil {
					artistName = album.Artists[0].Artist.Name
				}

				message := fmt.Sprintf(
					"💿 *%s*\n👤 %s\n🆔 `%s`\n\n",
					album.Title, artistName, album.ID,
				)

				if albumTracks, err := h.service.GetAlbumTracks("dummy", album.ID); err == nil && len(albumTracks) > 0 {
					message += "*📋 Track List:*\n"
					for j, track := range albumTracks {
						if j >= 5 {
							message += fmt.Sprintf("...and %d more\n", len(albumTracks)-5)
							break
						}
						message += fmt.Sprintf("%d. %s\n", j+1, track.Title)
					}
				} else {
					message += "_📋 Track list unavailable_\n"
				}

				message += fmt.Sprintf("\n⬇️ *Download:* `/download album %s`", album.ID)

				if album.ImageMedium != "" {
					photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileURL(album.ImageMedium))
					photo.Caption = message
					photo.ParseMode = tgbotapi.ModeMarkdown
					bot.Send(photo)
				} else {
					msg := tgbotapi.NewMessage(chatID, message)
					msg.ParseMode = tgbotapi.ModeMarkdown
					bot.Send(msg)
				}
			}
		}
	}

	// --- NO RESULTS ---
	if !hasResults {
		message := fmt.Sprintf("❌ No results found for: `%s`", query)
		msg := tgbotapi.NewMessage(chatID, message)
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
	}

	return nil
}

// handleDownload handles download commands
func (h *TelegramHandler) handleDownload(bot *tgbotapi.BotAPI, chatID int64, args string) error {
	parts := strings.Fields(args)
	if len(parts) < 2 {
		msg := tgbotapi.NewMessage(chatID, "❌ Invalid format. Usage: /download <type> <id>\nTypes: track, album, artist")
		bot.Send(msg)
		return nil
	}

	downloadType := strings.ToLower(parts[0])
	id := parts[1]

	var jobID string
	var err error
	var itemType string

	switch downloadType {
	case "track":
		jobID, err = h.service.DownloadTrack("dummy", id)
		itemType = "track"
	case "album":
		jobID, err = h.service.DownloadAlbum("dummy", id)
		itemType = "album"
	default:
		msg := tgbotapi.NewMessage(chatID, "❌ Invalid type. Use: track or album")
		bot.Send(msg)
		return nil
	}

	if err != nil {
		slog.Error("Failed to start download", "error", err, "type", downloadType, "id", id)
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ Failed to start download: %s", err.Error()))
		bot.Send(msg)
		return nil
	}

	message := fmt.Sprintf("✅ Download started!\n\n*Type:* %s\n*Job ID:* `%s`\n\nUse /jobs to check status.", itemType, jobID)
	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)
	return nil
}

// GetCommands returns the available commands for downloading
func (h *TelegramHandler) GetCommands() map[string]string {
	return map[string]string{
		"search":   "Search for music (artists, albums, tracks)",
		"download": "Download music by type and ID",
	}
}

// HandleCallback handles callback queries for this feature (downloading has no callbacks)
func (h *TelegramHandler) HandleCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) bool {
	return false // Downloading feature doesn't handle any callbacks
}
