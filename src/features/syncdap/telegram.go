package syncdap

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// TelegramHandler handles Telegram commands for the syncdap feature
type TelegramHandler struct {
	service *Service
}

// NewTelegramHandler creates a new Telegram handler for the syncdap feature
func NewTelegramHandler(service *Service) *TelegramHandler {
	return &TelegramHandler{service: service}
}

// HandleCommand processes syncdap-related Telegram commands
func (h *TelegramHandler) HandleCommand(bot *tgbotapi.BotAPI, chatID int64, command string, args string) error {
	switch command {
	case "dap":
		return h.handleDeviceList(bot, chatID)
	default:
		msg := tgbotapi.NewMessage(chatID, "❌ Unknown command. Use /dap")
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}
}

// GetCommands returns the available commands for this handler
func (h *TelegramHandler) GetCommands() map[string]string {
	return map[string]string{
		"dap": "Lists devices and their statuses",
	}
}

// HandleCallback handles callback queries for this feature (syncdap has no callbacks)
func (h *TelegramHandler) HandleCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) bool {
	return false // Syncdap feature doesn't handle any callbacks
}

// handleDeviceList shows sync device status
func (h *TelegramHandler) handleDeviceList(bot *tgbotapi.BotAPI, chatID int64) error {
	statuses := h.service.GetStatus()

	if len(statuses) == 0 {
		msg := tgbotapi.NewMessage(chatID, "🔄 *No sync devices configured*")
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	message := "🔄 *Sync Device Status*\n\n"
	for _, status := range statuses {
		mountStatus := "❌ Not mounted"
		if status.Mounted {
			mountStatus = fmt.Sprintf("✅ Mounted at `%s`", status.MountPath)
		}

		syncStatus := ""
		if _, running := h.service.findRunningSyncJob(status.UUID); running {
			syncStatus = " (🔄 Syncing...)"
		}

		message += fmt.Sprintf("💾 *%s*: %s%s\n", status.Name, mountStatus, syncStatus)

		if status.Error != "" {
			message += fmt.Sprintf("   ⚠️ Error: %s\n", status.Error)
		}
	}

	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)
	return nil
}
