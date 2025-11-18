package importing

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"sort"
	"strings"

	"github.com/contre95/soulsolid/src/features/config"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// TelegramHandler handles Telegram commands for importing
type TelegramHandler struct {
	service *Service
	config  *config.Manager
}

// NewTelegramHandler creates a new Telegram handler for importing
func NewTelegramHandler(service *Service, config *config.Manager) *TelegramHandler {
	return &TelegramHandler{
		service: service,
		config:  config,
	}
}

// getNextQueueItem gets the oldest item from the queue (stateless)
func (h *TelegramHandler) getNextQueueItem() (QueueItem, bool) {
	allItems := h.service.GetQueuedItems()
	if len(allItems) == 0 {
		return QueueItem{}, false
	}

	// Convert map to sorted slice by timestamp
	var items []QueueItem
	for _, item := range allItems {
		items = append(items, item)
	}

	// Sort by timestamp (oldest first)
	sort.Slice(items, func(i, j int) bool {
		return items[i].Timestamp.Before(items[j].Timestamp)
	})

	// Always return the first (oldest) item
	return items[0], true
}

// HandleQueue shows queue status and first/next item
func (h *TelegramHandler) HandleQueue(bot *tgbotapi.BotAPI, chatID int64) error {
	allItems := h.service.GetQueuedItems()
	queueCount := len(allItems)

	if queueCount == 0 {
		msg := tgbotapi.NewMessage(chatID, "📭 *Import Queue*\n\nNo items in queue")
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	// Get first item (no session needed)
	item, hasItem := h.getNextQueueItem()
	if !hasItem {
		// This should never happen if queueCount > 0, but safety check
		msg := tgbotapi.NewMessage(chatID, "📭 *Import Queue*\n\nNo items available")
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	// Send item with inline keyboard
	return h.sendQueueItem(bot, chatID, item, queueCount)
}

// HandleQueueNext gets next item explicitly
func (h *TelegramHandler) HandleQueueNext(bot *tgbotapi.BotAPI, chatID int64) error {
	item, hasItem := h.getNextQueueItem()
	if !hasItem {
		msg := tgbotapi.NewMessage(chatID, "📭 No more items in queue")
		bot.Send(msg)
		return nil
	}

	allItems := h.service.GetQueuedItems()
	return h.sendQueueItem(bot, chatID, item, len(allItems))
}

// HandleQueueClear clears entire queue
func (h *TelegramHandler) HandleQueueClear(bot *tgbotapi.BotAPI, chatID int64) error {
	err := h.service.ClearQueue()
	if err != nil {
		slog.Error("Failed to clear queue", "error", err)
		msg := tgbotapi.NewMessage(chatID, "❌ Failed to clear queue")
		bot.Send(msg)
		return nil
	}

	msg := tgbotapi.NewMessage(chatID, "🗑️ Queue cleared successfully")
	bot.Send(msg)
	return nil
}

// HandleQueueBulkArtist shows bulk processing options for artists
func (h *TelegramHandler) HandleQueueBulkArtist(bot *tgbotapi.BotAPI, chatID int64) error {
	groups := h.service.GetGroupedByArtist()

	if len(groups) == 0 {
		msg := tgbotapi.NewMessage(chatID, "📭 *Bulk Artist Processing*\n\nNo queued items to process")
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	// Send header message
	headerText := fmt.Sprintf("👥 *Bulk Artist Processing*\n\nFound %d artist(s) with queued tracks:", len(groups))
	msg := tgbotapi.NewMessage(chatID, headerText)
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)

	// Send each artist group with inline keyboard
	for artistName, items := range groups {
		h.sendBulkArtistGroup(bot, chatID, artistName, items)
	}

	return nil
}

// HandleQueueBulkAlbum shows bulk processing options for albums
func (h *TelegramHandler) HandleQueueBulkAlbum(bot *tgbotapi.BotAPI, chatID int64) error {
	groups := h.service.GetGroupedByAlbum()

	if len(groups) == 0 {
		msg := tgbotapi.NewMessage(chatID, "📭 *Bulk Album Processing*\n\nNo queued items to process")
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	// Send header message
	headerText := fmt.Sprintf("💿 *Bulk Album Processing*\n\nFound %d album(s) with queued tracks:", len(groups))
	msg := tgbotapi.NewMessage(chatID, headerText)
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)

	// Send each album group with inline keyboard
	for albumName, items := range groups {
		h.sendBulkAlbumGroup(bot, chatID, albumName, items)
	}

	return nil
}

// sendBulkArtistGroup sends a single artist group with inline keyboard
func (h *TelegramHandler) sendBulkArtistGroup(bot *tgbotapi.BotAPI, chatID int64, artistName string, items []QueueItem) {
	text := h.formatBulkArtistMessage(artistName, items)
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ReplyMarkup = h.createBulkArtistKeyboard(artistName, items)
	bot.Send(msg)
}

// sendBulkAlbumGroup sends a single album group with inline keyboard
func (h *TelegramHandler) sendBulkAlbumGroup(bot *tgbotapi.BotAPI, chatID int64, albumName string, items []QueueItem) {
	text := h.formatBulkAlbumMessage(albumName, items)
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ReplyMarkup = h.createBulkAlbumKeyboard(albumName, items)
	bot.Send(msg)
}

// formatBulkArtistMessage formats artist group information for Telegram
func (h *TelegramHandler) formatBulkArtistMessage(artistName string, items []QueueItem) string {
	escapedArtist := h.escapeMarkdown(artistName)
	return fmt.Sprintf("👤 *%s*\n%d track(s) queued", escapedArtist, len(items))
}

// formatBulkAlbumMessage formats album group information for Telegram
func (h *TelegramHandler) formatBulkAlbumMessage(albumName string, items []QueueItem) string {
	escapedAlbum := h.escapeMarkdown(albumName)
	return fmt.Sprintf("💿 *%s*\n%d track(s) queued", escapedAlbum, len(items))
}

// createBulkArtistKeyboard creates inline keyboard for artist bulk actions
func (h *TelegramHandler) createBulkArtistKeyboard(artistName string, items []QueueItem) tgbotapi.InlineKeyboardMarkup {
	// URL encode the artist name for callback data
	encodedArtist := url.QueryEscape(artistName)

	buttons := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("✅ Import All", fmt.Sprintf("queue_bulk_artist_import_%s", encodedArtist)),
			tgbotapi.NewInlineKeyboardButtonData("⏭️ Skip All", fmt.Sprintf("queue_bulk_artist_cancel_%s", encodedArtist)),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("🗑️ Delete All", fmt.Sprintf("queue_bulk_artist_delete_%s", encodedArtist)),
		},
	}

	return tgbotapi.NewInlineKeyboardMarkup(buttons...)
}

// createBulkAlbumKeyboard creates inline keyboard for album bulk actions
func (h *TelegramHandler) createBulkAlbumKeyboard(albumName string, items []QueueItem) tgbotapi.InlineKeyboardMarkup {
	// URL encode the album name for callback data
	encodedAlbum := url.QueryEscape(albumName)

	buttons := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("✅ Import All", fmt.Sprintf("queue_bulk_album_import_%s", encodedAlbum)),
			tgbotapi.NewInlineKeyboardButtonData("⏭️ Skip All", fmt.Sprintf("queue_bulk_album_cancel_%s", encodedAlbum)),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("🗑️ Delete All", fmt.Sprintf("queue_bulk_album_delete_%s", encodedAlbum)),
		},
	}

	return tgbotapi.NewInlineKeyboardMarkup(buttons...)
}

// HandleBulkGroupAction processes bulk group actions from callback queries
func (h *TelegramHandler) HandleBulkGroupAction(bot *tgbotapi.BotAPI, chatID int64, groupType, action, encodedGroupKey string, callback *tgbotapi.CallbackQuery) {
	// URL decode the group key
	groupKey, err := url.QueryUnescape(encodedGroupKey)
	if err != nil {
		slog.Error("Failed to decode group key", "error", err, "encodedKey", encodedGroupKey)
		msg := tgbotapi.NewMessage(chatID, "❌ Failed to process group action: invalid group key")
		bot.Send(msg)
		return
	}

	ctx := context.Background()
	err = h.service.ProcessGroup(ctx, groupKey, groupType, action)
	if err != nil {
		slog.Error("Failed to process group", "error", err, "groupKey", groupKey, "groupType", groupType, "action", action)
		escapedError := h.escapeMarkdown(err.Error())
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ Failed to process group '%s': %s", h.escapeMarkdown(groupKey), escapedError))
		bot.Send(msg)
		return
	}

	// Send success message
	actionMsg := "processed"
	switch action {
	case "import":
		actionMsg = "imported"
	case "cancel":
		actionMsg = "skipped"
	case "delete":
		actionMsg = "deleted"
	}

	groupTypeMsg := "artist"
	if groupType == "album" {
		groupTypeMsg = "album"
	}

	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("✅ All tracks by %s '%s' %s successfully", groupTypeMsg, h.escapeMarkdown(groupKey), actionMsg))
	bot.Send(msg)
}

// sendQueueItem sends a queue item with inline keyboard
func (h *TelegramHandler) sendQueueItem(bot *tgbotapi.BotAPI, chatID int64, item QueueItem, totalCount int) error {
	text := h.formatQueueItemMessage(item, totalCount)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ReplyMarkup = h.createInlineKeyboard(item)

	_, err := bot.Send(msg)
	return err
}

// formatQueueItemMessage formats track information for Telegram
func (h *TelegramHandler) formatQueueItemMessage(item QueueItem, totalCount int) string {
	track := item.Track

	var artists string
	if len(track.Artists) > 0 {
		for i, artist := range track.Artists {
			if i > 0 {
				artists += ", "
			}
			artists += artist.Artist.Name
		}
	}

	typeText := "Manual Review"
	if item.Type == "duplicate" {
		typeText = "Duplicate"
	}

	// Escape text fields for Markdown, but use code formatting for file path
	escapedTitle := h.escapeMarkdown(track.Title)
	escapedArtists := h.escapeMarkdown(artists)
	escapedAlbum := h.escapeMarkdown(track.Album.Title)

	return fmt.Sprintf(`📀 *Import Queue Item*

*Title:* %s
*Artists:* %s
*Album:* %s
*Type:* %s
*Added:* %s

*File:* `+"`%s`"+`

Queue: %d items remaining`,
		escapedTitle,
		escapedArtists,
		escapedAlbum,
		typeText,
		item.Timestamp.Format("Jan 2, 15:04"),
		track.Path,   // Raw path wrapped in backticks for clean formatting
		totalCount-1) // subtract current item
}

// escapeMarkdown escapes special characters in text for safe Markdown usage
func (h *TelegramHandler) escapeMarkdown(text string) string {
	// Escape characters that have special meaning in Markdown
	text = strings.ReplaceAll(text, "`", "\\`")
	text = strings.ReplaceAll(text, "*", "\\*")
	text = strings.ReplaceAll(text, "_", "\\_")
	text = strings.ReplaceAll(text, "[", "\\[")
	text = strings.ReplaceAll(text, "]", "\\]")
	text = strings.ReplaceAll(text, "(", "\\(")
	text = strings.ReplaceAll(text, ")", "\\)")
	text = strings.ReplaceAll(text, "~", "\\~")
	text = strings.ReplaceAll(text, ">", "\\>")
	text = strings.ReplaceAll(text, "#", "\\#")
	text = strings.ReplaceAll(text, "+", "\\+")
	text = strings.ReplaceAll(text, "-", "\\-")
	text = strings.ReplaceAll(text, "=", "\\=")
	text = strings.ReplaceAll(text, "|", "\\|")
	text = strings.ReplaceAll(text, "{", "\\{")
	text = strings.ReplaceAll(text, "}", "\\}")
	text = strings.ReplaceAll(text, ".", "\\.")
	text = strings.ReplaceAll(text, "!", "\\!")
	return text
}

// createInlineKeyboard creates appropriate buttons based on item type
func (h *TelegramHandler) createInlineKeyboard(item QueueItem) tgbotapi.InlineKeyboardMarkup {
	var buttons [][]tgbotapi.InlineKeyboardButton

	// Action buttons based on type
	if item.Type == "duplicate" {
		buttons = append(buttons, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("✴️ Replace", fmt.Sprintf("queue_replace_%s", item.ID)),
			tgbotapi.NewInlineKeyboardButtonData("⏭️ Skip", fmt.Sprintf("queue_cancel_%s", item.ID)),
			tgbotapi.NewInlineKeyboardButtonData("🗑️ Delete", fmt.Sprintf("queue_delete_%s", item.ID)),
		})
	} else {
		buttons = append(buttons, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("✅ Import", fmt.Sprintf("queue_import_%s", item.ID)),
			tgbotapi.NewInlineKeyboardButtonData("❌ Cancel", fmt.Sprintf("queue_cancel_%s", item.ID)),
			tgbotapi.NewInlineKeyboardButtonData("🗑️ Delete", fmt.Sprintf("queue_delete_%s", item.ID)),
		})
	}

	// Navigation buttons
	buttons = append(buttons, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("⏭️ Next", "queue_next"),
	})

	return tgbotapi.NewInlineKeyboardMarkup(buttons...)
}

// HandleQueueAction processes a queue item action from callback
func (h *TelegramHandler) HandleQueueAction(bot *tgbotapi.BotAPI, chatID int64, itemID, action string, callback *tgbotapi.CallbackQuery) {
	ctx := context.Background()
	err := h.service.ProcessQueueItem(ctx, itemID, action)
	if err != nil {
		slog.Error("Failed to process queue item", "error", err, "itemID", itemID, "action", action)
		// Send error message
		escapedError := h.escapeMarkdown(err.Error())
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ Failed to %s item: %s", action, escapedError))
		bot.Send(msg)
		return
	}

	// Send success message
	actionMsg := "processed"
	switch action {
	case "import":
		actionMsg = "imported"
	case "replace":
		actionMsg = "replaced"
	case "cancel":
		actionMsg = "skipped"
	case "delete":
		actionMsg = "deleted"
	}

	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("✅ Track %s successfully", actionMsg))
	bot.Send(msg)

	// Automatically send next item if available
	h.sendNextItemIfAvailable(bot, chatID)
}

// sendNextItemIfAvailable sends the next queue item if available
func (h *TelegramHandler) sendNextItemIfAvailable(bot *tgbotapi.BotAPI, chatID int64) {
	item, hasItem := h.getNextQueueItem()
	if !hasItem {
		msg := tgbotapi.NewMessage(chatID, "🎉 All items processed!")
		bot.Send(msg)
		return
	}

	allItems := h.service.GetQueuedItems()
	h.sendQueueItem(bot, chatID, item, len(allItems))
}

// HandleCommand processes Telegram commands for importing
func (h *TelegramHandler) HandleCommand(bot *tgbotapi.BotAPI, chatID int64, command string, args string) error {
	switch command {
	case "import":
		return h.handleImport(bot, chatID, args)
	case "queue":
		return h.HandleQueue(bot, chatID)
	case "queue_clear":
		return h.HandleQueueClear(bot, chatID)
	case "queue_bulk_artist":
		return h.HandleQueueBulkArtist(bot, chatID)
	case "queue_bulk_album":
		return h.HandleQueueBulkAlbum(bot, chatID)
	default:
		msg := tgbotapi.NewMessage(chatID, "❌ Unknown import command. Use /help for available commands")
		bot.Send(msg)
		return nil
	}
}

// handleImport handles import commands
func (h *TelegramHandler) handleImport(bot *tgbotapi.BotAPI, chatID int64, path string) error {
	// Use downloadPath as default if no path provided
	importPath := strings.TrimSpace(path)
	if importPath == "" {
		config := h.config.Get()
		importPath = config.DownloadPath
		if importPath == "" {
			importPath = "./downloads"
		}
	}

	// Start the import job
	jobID, err := h.service.ImportDirectory(context.TODO(), importPath)
	if err != nil {
		slog.Error("Failed to start import", "error", err, "path", importPath)
		escapedError := h.escapeMarkdown(err.Error())
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ Failed to start import: %s", escapedError))
		bot.Send(msg)
		return nil
	}

	message := fmt.Sprintf("✅ Import started!\n\n*Path:* `%s`\n*Job ID:* `%s`\n\nUse /jobs to check status.", importPath, jobID)
	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)
	return nil
}

// GetCommands returns the available commands for importing
func (h *TelegramHandler) GetCommands() map[string]string {
	return map[string]string{
		"import":            "Import music from directory (defaults to downloadPath)",
		"queue":             "Show import queue and process items one by one",
		"queue_clear":       "Clear entire import queue",
		"queue_bulk_artist": "Bulk process queued items grouped by artist",
		"queue_bulk_album":  "Bulk process queued items grouped by album",
	}
}

// HandleCallback handles callback queries for this feature
func (h *TelegramHandler) HandleCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) bool {
	data := callback.Data
	if !strings.HasPrefix(data, "queue_") {
		return false // Not handled by this feature
	}

	chatID := callback.Message.Chat.ID

	// Parse action and item ID
	parts := strings.Split(data, "_")
	if len(parts) < 2 {
		return false
	}

	action := parts[1]

	// Handle bulk group actions
	if strings.HasPrefix(data, "queue_bulk_artist_") || strings.HasPrefix(data, "queue_bulk_album_") {
		if len(parts) < 4 {
			return false
		}
		groupType := parts[2]                           // "artist" or "album"
		bulkAction := parts[3]                          // "import", "cancel", or "delete"
		encodedGroupKey := strings.Join(parts[4:], "_") // Rejoin remaining parts as they may contain underscores

		h.HandleBulkGroupAction(bot, chatID, groupType, bulkAction, encodedGroupKey, callback)
		return true
	}

	var itemID string
	if len(parts) > 2 {
		itemID = parts[2]
	}

	// Handle individual queue actions
	switch action {
	case "import", "replace", "delete":
		h.HandleQueueAction(bot, chatID, itemID, action, callback)
	case "cancel":
		if itemID == "session" {
			// Session cancellation no longer needed in stateless approach
			msg := tgbotapi.NewMessage(chatID, "✅ Session management removed. Use /queue to continue.")
			bot.Send(msg)
		} else {
			h.HandleQueueAction(bot, chatID, itemID, action, callback)
		}
	case "next":
		h.HandleQueueNext(bot, chatID)
	default:
		return false
	}

	return true // Callback was handled
}
