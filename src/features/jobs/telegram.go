package jobs

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// TelegramHandler handles Telegram commands for the jobs feature
type TelegramHandler struct {
	service *Service
}

// NewTelegramHandler creates a new Telegram handler for the jobs feature
func NewTelegramHandler(service *Service) *TelegramHandler {
	return &TelegramHandler{service: service}
}

// HandleCommand processes jobs-related Telegram commands
func (h *TelegramHandler) HandleCommand(bot *tgbotapi.BotAPI, chatID int64, command string, args string) error {
	switch command {
	case "jobs":
		return h.handleJobs(bot, chatID)
	default:
		msg := tgbotapi.NewMessage(chatID, "❌ Unknown jobs command. Use /jobs")
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}
}

// GetCommands returns the available commands for this handler
func (h *TelegramHandler) GetCommands() map[string]string {
	return map[string]string{
		"jobs": "Show active jobs",
	}
}

// HandleCallback handles callback queries for this feature (jobs has no callbacks)
func (h *TelegramHandler) HandleCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) bool {
	return false // Jobs feature doesn't handle any callbacks
}

// handleJobs shows active jobs
func (h *TelegramHandler) handleJobs(bot *tgbotapi.BotAPI, chatID int64) error {
	jobs := h.service.GetJobs()

	if len(jobs) == 0 {
		msg := tgbotapi.NewMessage(chatID, "📋 *No active jobs*")
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
		return nil
	}

	message := "📋 *Active Jobs*\n\n"
	for _, job := range jobs {
		statusEmoji := h.getJobStatusEmoji(job.Status)
		message += fmt.Sprintf("%s `%s`: %s (%d%%)\n", statusEmoji, job.Name, job.Message, job.Progress)
	}

	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)
	return nil
}

// getJobStatusEmoji returns emoji for job status
func (h *TelegramHandler) getJobStatusEmoji(status JobStatus) string {
	switch status {
	case JobStatusPending:
		return "⏳"
	case JobStatusRunning:
		return "🔄"
	case JobStatusCompleted:
		return "✅"
	case JobStatusFailed:
		return "❌"
	case JobStatusCancelled:
		return "🚫"
	default:
		return "❓"
	}
}
