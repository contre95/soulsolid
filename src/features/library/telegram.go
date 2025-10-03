package library

import (
	"context"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// TelegramHandler handles Telegram commands for the library feature
type TelegramHandler struct {
	service *Service
}

// NewTelegramHandler creates a new Telegram handler for the library feature
func NewTelegramHandler(service *Service) *TelegramHandler {
	return &TelegramHandler{service: service}
}

// HandleCommand processes library-related Telegram commands
func (h *TelegramHandler) HandleCommand(bot *tgbotapi.BotAPI, chatID int64, command string, args string) error {
	switch command {
	case "stats":
		return h.handleStats(bot, chatID)
	case "tree":
		return h.handleTree(bot, chatID, args)
	default:
		bot.Send(tgbotapi.NewMessage(chatID, "❌ Unknown library command. Use /stats or /tree [downloads]"))
		return nil
	}
}

// GetCommands returns the available commands for this handler
func (h *TelegramHandler) GetCommands() map[string]string {
	return map[string]string{
		"stats": "Show library statistics",
		"tree":  "Show library or downloads file tree (/tree or /tree downloads)",
	}
}

// HandleCallback handles callback queries for this feature (library has no callbacks)
func (h *TelegramHandler) HandleCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) bool {
	return false // Library feature doesn't handle any callbacks
}

// handleStats shows library statistics
func (h *TelegramHandler) handleStats(bot *tgbotapi.BotAPI, chatID int64) error {
	ctx := context.Background()

	tracksCount, err := h.service.GetTracksCount(ctx)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "❌ Failed to get tracks count")
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return err
	}

	artistsCount, err := h.service.GetArtistsCount(ctx)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "❌ Failed to get artists count")
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return err
	}

	albumsCount, err := h.service.GetAlbumsCount(ctx)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "❌ Failed to get albums count")
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return err
	}

	message := fmt.Sprintf("📊 *Library Statistics*\n\n"+
		"🎵 Tracks: `%d`\n---\n"+
		"👤 Artists: `%d`\n---\n"+
		"💿 Albums: `%d`", tracksCount, artistsCount, albumsCount)

	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)
	return nil
}

// handleTree shows library or downloads file tree
func (h *TelegramHandler) handleTree(bot *tgbotapi.BotAPI, chatID int64, args string) error {
	var tree string
	var err error
	var treeType string

	// Determine which tree to show based on args
	if args == "downloads" {
		tree, err = h.service.GetDownloadsFileTree()
		treeType = "downloads"
	} else {
		tree, err = h.service.GetLibraryFileTree()
		treeType = "library"
	}

	if err != nil {
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ Failed to get %s file tree: %s", treeType, err.Error()))
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return err
	}

	// Handle message length limits
	if len(tree) > 4000 {
		tree = tree[:4000] + "\n... (truncated - use web interface for full tree)"
		doc := tgbotapi.NewDocument(chatID, tgbotapi.FileBytes{
			Name:  fmt.Sprintf("%s_tree.txt", treeType),
			Bytes: []byte(tree),
		})
		bot.Send(doc)
	} else {
		message := fmt.Sprintf("🌳 *%s File Tree*\n\n```\n%s\n```", strings.Title(treeType), tree)
		msg := tgbotapi.NewMessage(chatID, message)
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
	}
	return nil
}
