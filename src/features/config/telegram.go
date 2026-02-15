package config

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// TelegramHandler handles Telegram commands for the config feature
type TelegramHandler struct {
	configManager *Manager
}

// NewTelegramHandler creates a new Telegram handler for the config feature
func NewTelegramHandler(configManager *Manager) *TelegramHandler {
	return &TelegramHandler{configManager: configManager}
}

// HandleCommand processes config-related Telegram commands
func (h *TelegramHandler) HandleCommand(bot *tgbotapi.BotAPI, chatID int64, command string, args string) error {
	switch command {
	case "config":
		return h.handleConfig(bot, chatID, args)
	default:
		msg := tgbotapi.NewMessage(chatID, "❌ Unknown config command. Use /config")
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}
}

// GetCommands returns the available commands for this handler
func (h *TelegramHandler) GetCommands() map[string]string {
	return map[string]string{
		"config": "Show configuration (use 'yaml' for YAML format)",
	}
}

// HandleCallback handles callback queries for this feature (config has no callbacks)
func (h *TelegramHandler) HandleCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) bool {
	return false // Config feature doesn't handle any callbacks
}

// handleConfig shows configuration
func (h *TelegramHandler) handleConfig(bot *tgbotapi.BotAPI, chatID int64, args string) error {
	format := "yaml" // default
	var configStr string
	configStr = h.configManager.GetYAML()
	message := fmt.Sprintf("⚙️ *Configuration (%s)*\n\n```%s\n%s\n```", format, format, configStr)
	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)
	return nil
}
