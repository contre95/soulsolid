package hosting

import (
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/contre95/soulsolid/src/features/config"
	"github.com/contre95/soulsolid/src/features/downloading"
	"github.com/contre95/soulsolid/src/features/importing"
	"github.com/contre95/soulsolid/src/features/jobs"
	"github.com/contre95/soulsolid/src/features/library"
	"github.com/contre95/soulsolid/src/features/syncdap"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// TelegramCommandHandler interface that each feature implements
type TelegramCommandHandler interface {
	HandleCommand(bot *tgbotapi.BotAPI, chatID int64, command string, args string) error
	GetCommands() map[string]string                                             // Returns command -> description mapping
	HandleCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) bool // Handle feature-specific callbacks
}

// TelegramBot handles Telegram bot operations
type TelegramBot struct {
	bot           *tgbotapi.BotAPI
	config        *config.Manager
	handlers      map[string]TelegramCommandHandler
	updates       tgbotapi.UpdatesChannel
	stopChan      chan struct{}
	pendingInputs map[string]string // chatID_messageID -> callbackData
}

// NewTelegramBot creates a new Telegram bot instance
func NewTelegramBot(cfg *config.Manager, libraryService *library.Service, jobService *jobs.Service, syncService *syncdap.Service, downloadingService *downloading.Service, importingService *importing.Service) (*TelegramBot, error) {
	telegramConfig := cfg.Get().Telegram

	if !telegramConfig.Enabled {
		return nil, fmt.Errorf("telegram bot is disabled in configuration")
	}

	if telegramConfig.Token == "" {
		return nil, fmt.Errorf("telegram bot token is not configured")
	}

	bot, err := tgbotapi.NewBotAPI(telegramConfig.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot: %w", err)
	}

	slog.Info("Telegram bot initialized", "username", bot.Self.UserName)

	// Set up update configuration
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 30

	updates := bot.GetUpdatesChan(updateConfig)

	telegramBot := &TelegramBot{
		bot:           bot,
		config:        cfg,
		handlers:      make(map[string]TelegramCommandHandler),
		updates:       updates,
		stopChan:      make(chan struct{}),
		pendingInputs: make(map[string]string),
	}

	// Register feature handlers
	telegramBot.RegisterHandler("library", library.NewTelegramHandler(libraryService))
	telegramBot.RegisterHandler("config", config.NewTelegramHandler(cfg))
	telegramBot.RegisterHandler("jobs", jobs.NewTelegramHandler(jobService))
	telegramBot.RegisterHandler("syncdap", syncdap.NewTelegramHandler(syncService))
	telegramBot.RegisterHandler("downloading", downloading.NewTelegramHandler(downloadingService))
	telegramBot.RegisterHandler("importing", importing.NewTelegramHandler(importingService, cfg))

	return telegramBot, nil
}

// RegisterHandler registers a feature's command handler
func (t *TelegramBot) RegisterHandler(feature string, handler TelegramCommandHandler) {
	t.handlers[feature] = handler
	slog.Debug("Registered Telegram handler", "feature", feature)
}

// Start begins listening for Telegram updates
func (t *TelegramBot) Start() {
	slog.Info("Starting Telegram bot listener")

	for {
		select {
		case update := <-t.updates:
			if update.Message != nil {
				go t.handleMessage(update)
			}
			if update.CallbackQuery != nil {
				go t.handleCallbackQuery(update)
			}
		case <-t.stopChan:
			slog.Info("Stopping Telegram bot listener")
			return
		}
	}
}

// Stop gracefully stops the bot
func (t *TelegramBot) Stop() {
	close(t.stopChan)
}

// handleMessage processes incoming messages
func (t *TelegramBot) handleMessage(update tgbotapi.Update) {
	message := update.Message
	chatID := message.Chat.ID

	// Check if message is from authorized user
	allowedUsers := t.config.Get().Telegram.AllowedUsers
	if len(allowedUsers) == 0 && !t.config.Get().Demo {
		slog.Warn("No allowed users configured", "chat_id", chatID)
		t.sendMessage(chatID, "❌ Access denied: No users configured. Please add users to the config.")
		return
	}

	username := message.From.UserName
	if username == "" {
		// Fallback to first name + last name
		username = message.From.FirstName
		if message.From.LastName != "" {
			username += " " + message.From.LastName
		}
	}
	found := slices.Contains(allowedUsers, username)
	if !found && !t.config.Get().Demo {
		slog.Warn("Unauthorized user", "username", username, "chat_id", chatID)
		t.sendMessage(chatID, "Unknown user, please add your user to the config")
		return
	}

	// Handle commands
	if message.IsCommand() {
		t.handleCommand(update)
		return
	}

	// Check if this is a reply to one of our prompts
	if message.ReplyToMessage != nil {
		if t.handleReplyInput(message) {
			return // Reply was handled
		}
	}

	// Handle non-command messages
	t.sendMessage(chatID, "🤖 Send /menu or /help to see available options")
}

// handleCommand processes bot commands
func (t *TelegramBot) handleCommand(update tgbotapi.Update) {
	message := update.Message
	chatID := message.Chat.ID
	command := message.Command()
	args := message.CommandArguments()

	slog.Debug("Processing command", "command", command, "args", args, "chat_id", chatID)

	switch command {
	case "help":
		t.handleHelp(chatID)
	case "start":
		t.handleHelp(chatID) // Show menu on start
	case "menu":
		t.handleHelp(chatID) // Show menu
	default:
		// Route command to appropriate feature handler
		if err := t.routeCommand(command, args, chatID); err != nil {
			slog.Error("Failed to handle command", "command", command, "error", err)
			t.sendMessage(chatID, "❌ Failed to process command")
		}
	}
}

// routeCommand routes commands to the appropriate feature handler
func (t *TelegramBot) routeCommand(command, args string, chatID int64) error {
	// Define command to feature mapping
	commandMap := map[string]string{
		"stats":       "library",
		"tree":        "library",
		"config":      "config",
		"jobs":        "jobs",
		"dap":         "syncdap",
		"search":      "downloading",
		"download":    "downloading",
		"import":      "importing",
		"queue":       "importing",
		"queue_clear": "importing",
	}

	feature, exists := commandMap[command]
	if !exists {
		t.sendMessage(chatID, "❌ Unknown command. Send /help to see available commands.")
		return nil
	}

	handler, exists := t.handlers[feature]
	if !exists {
		escapedFeature := t.escapeMarkdown(feature)
		t.sendMessage(chatID, fmt.Sprintf("❌ %s feature not available", escapedFeature))
		return nil
	}

	return handler.HandleCommand(t.bot, chatID, command, args)
}

// escapeMarkdown escapes special characters for safe Markdown usage
func (t *TelegramBot) escapeMarkdown(text string) string {
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

// sendMessage sends a message to the specified chat
func (t *TelegramBot) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	_, err := t.bot.Send(msg)
	if err != nil {
		slog.Error("Failed to send message", "error", err, "chat_id", chatID)
	}
}

// handleCallbackQuery handles callback queries from inline keyboards
func (t *TelegramBot) handleCallbackQuery(update tgbotapi.Update) {
	callback := update.CallbackQuery

	// Handle menu callbacks first
	if strings.HasPrefix(callback.Data, "menu_") {
		t.handleMenuCallback(callback)
		return
	}

	// Route callback to appropriate feature handler
	for _, handler := range t.handlers {
		if handler.HandleCallback(t.bot, callback) {
			break // Callback was handled
		}
	}

	// Answer callback to remove loading state
	callbackResp := tgbotapi.NewCallback(callback.ID, "")
	t.bot.Request(callbackResp)
}

// handleHelp shows main menu with inline keyboard
func (t *TelegramBot) handleHelp(chatID int64) {
	text := `*🤖 Soulsolid Main Menu*

Choose an action below or use commands directly:`

	buttons := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("📊 Library", "menu_library"),
			tgbotapi.NewInlineKeyboardButtonData("🔍 Download", "menu_download"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("📋 Jobs", "menu_jobs"),
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Config", "menu_config"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("📥 Import", "menu_import"),
			tgbotapi.NewInlineKeyboardButtonData("🔄 Sync DAP", "menu_sync"),
		},
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
	_, err := t.bot.Send(msg)
	if err != nil {
		slog.Error("Failed to send menu", "error", err, "chat_id", chatID)
	}
}

// handleMenuCallback handles main menu callback queries
func (t *TelegramBot) handleMenuCallback(callback *tgbotapi.CallbackQuery) {
	chatID := callback.Message.Chat.ID
	data := callback.Data

	// Answer callback to remove loading state
	callbackResp := tgbotapi.NewCallback(callback.ID, "")
	t.bot.Request(callbackResp)

	switch data {
	case "menu_library":
		t.showLibraryMenu(chatID)
	case "menu_download":
		t.showDownloadMenu(chatID)
	case "menu_jobs":
		t.routeMenuCommand("jobs", "", chatID)
	case "menu_config":
		t.routeMenuCommand("config", "", chatID)
	case "menu_import":
		t.showImportMenu(chatID)
	case "menu_sync":
		t.routeMenuCommand("dap", "", chatID)
	case "menu_back":
		t.handleHelp(chatID)
	case "menu_lib_stats", "menu_lib_tree", "menu_import_queue", "menu_import_clear":
		t.routeMenuCommand(data, "", chatID)
	case "menu_import_dir":
		t.promptForInput(chatID, "📁 *Import Directory*\n\nPlease reply with the directory path to import:\n\nLeave empty to use default download path, or specify a custom path like `/path/to/music`", "menu_import_dir")
	case "menu_dl_search":
		t.promptForInput(chatID, "🔍 *Search for music*\n\nPlease reply with your search query:\n`tracks <query>`, `albums <query>`, or just `<query>` for all types", "menu_dl_search")
	case "menu_dl_download":
		t.promptForInput(chatID, "⬇️ *Download music*\n\nPlease reply with:\n`/download <type> <id>`\n\nTypes: `track`, `album`\nExample: `/download track 123456`", "menu_dl_download")
	}
}

// showLibraryMenu shows library-specific options
func (t *TelegramBot) showLibraryMenu(chatID int64) {
	text := `*📊 Library Menu*

Choose a library action:`

	buttons := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("📈 Statistics", "menu_lib_stats"),
			tgbotapi.NewInlineKeyboardButtonData("🌳 File Tree", "menu_lib_tree"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Back", "menu_back"),
		},
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
	t.bot.Send(msg)
}

// showDownloadMenu shows download-specific options
func (t *TelegramBot) showDownloadMenu(chatID int64) {
	text := `*🔍 Download Menu*

Choose a download action:
• *Search* - Will prompt for search query
• *Download* - Will prompt for type & ID`

	buttons := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("🔍 Search Music", "menu_dl_search"),
			tgbotapi.NewInlineKeyboardButtonData("⬇️ Download by ID", "menu_dl_download"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Back", "menu_back"),
		},
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
	t.bot.Send(msg)
}

// promptForInput sends a message that forces user to reply with input
func (t *TelegramBot) promptForInput(chatID int64, promptText, callbackData string) {
	msg := tgbotapi.NewMessage(chatID, promptText)
	msg.ParseMode = tgbotapi.ModeMarkdown

	// Create force reply markup
	forceReply := tgbotapi.ForceReply{
		ForceReply: true,
		Selective:  false,
	}
	msg.ReplyMarkup = forceReply

	sentMsg, err := t.bot.Send(msg)
	if err != nil {
		slog.Error("Failed to send prompt", "error", err)
		return
	}

	// Store the callback data for when user replies
	// We'll handle this in handleMessage by checking if it's a reply to our prompt
	t.storePendingInput(chatID, sentMsg.MessageID, callbackData)
}

// storePendingInput stores information about pending user input
func (t *TelegramBot) storePendingInput(chatID int64, messageID int, callbackData string) {
	key := fmt.Sprintf("%d_%d", chatID, messageID)
	t.pendingInputs[key] = callbackData
}

// handleReplyInput handles replies to our input prompts
func (t *TelegramBot) handleReplyInput(message *tgbotapi.Message) bool {
	chatID := message.Chat.ID
	replyToID := message.ReplyToMessage.MessageID

	key := fmt.Sprintf("%d_%d", chatID, replyToID)
	callbackData, exists := t.pendingInputs[key]
	if !exists {
		return false // Not a reply to our prompt
	}

	// Remove the pending input
	delete(t.pendingInputs, key)

	userInput := message.Text

	switch callbackData {
	case "menu_dl_search":
		// Route to search command
		t.routeMenuCommand("menu_dl_search", userInput, chatID)
	case "menu_dl_download":
		// Handle download command - extract type and id from user input
		if strings.HasPrefix(userInput, "/download ") {
			args := strings.TrimPrefix(userInput, "/download ")
			t.routeMenuCommand("menu_dl_download", args, chatID)
		} else {
			t.sendMessage(chatID, "❌ Please use the format: `/download <type> <id>`\nExample: `/download track 123456`")
		}
	case "menu_import_dir":
		// Route to import command with user input as path
		t.routeMenuCommand("menu_import_dir", userInput, chatID)
	default:
		return false
	}

	return true
}

// showImportMenu shows import-specific options
func (t *TelegramBot) showImportMenu(chatID int64) {
	text := `*📥 Import Menu*

Choose an import action:
• *Queue* - View and process import queue
• *Import Dir* - Will prompt for directory path
• *Clear Queue* - Remove all queued items`

	buttons := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("📋 View Queue", "menu_import_queue"),
			tgbotapi.NewInlineKeyboardButtonData("📁 Import Directory", "menu_import_dir"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("🗑️ Clear Queue", "menu_import_clear"),
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Back", "menu_back"),
		},
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
	t.bot.Send(msg)
}

// routeMenuCommand routes menu selections to appropriate feature handlers
func (t *TelegramBot) routeMenuCommand(command, args string, chatID int64) {
	commandMap := map[string]string{
		"jobs":              "jobs",
		"config":            "config",
		"dap":               "syncdap",
		"menu_lib_stats":    "library",
		"menu_lib_tree":     "library",
		"menu_dl_search":    "downloading",
		"menu_dl_download":  "downloading",
		"menu_import_queue": "importing",
		"menu_import_dir":   "importing",
		"menu_import_clear": "importing",
	}

	commandMapToCmd := map[string]string{
		"menu_lib_stats":    "stats",
		"menu_lib_tree":     "tree",
		"menu_dl_search":    "search",
		"menu_dl_download":  "download",
		"menu_import_queue": "queue",
		"menu_import_dir":   "import",
		"menu_import_clear": "queue_clear",
	}

	feature, exists := commandMap[command]
	if !exists {
		t.sendMessage(chatID, "❌ Unknown menu option")
		return
	}

	handler, exists := t.handlers[feature]
	if !exists {
		escapedFeature := t.escapeMarkdown(feature)
		t.sendMessage(chatID, fmt.Sprintf("❌ %s feature not available", escapedFeature))
		return
	}

	actualCommand := commandMapToCmd[command]
	if actualCommand == "" {
		actualCommand = command
	}

	err := handler.HandleCommand(t.bot, chatID, actualCommand, args)
	if err != nil {
		slog.Error("Failed to handle menu command", "command", command, "error", err)
		t.sendMessage(chatID, "❌ Failed to process menu selection")
	}
}
