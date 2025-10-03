package logging

import (
	"log/slog"
	"os"
	"time"

	"github.com/charmbracelet/log"
	"github.com/contre95/soulsolid/src/features/config"
)

func SetupLogger(cfg *config.Manager) *slog.Logger {
	var formatter log.Formatter
	switch cfg.Get().Logger.Format {
	case "json":
		formatter = log.JSONFormatter
	case "text":
		formatter = log.TextFormatter
	default:
		formatter = log.LogfmtFormatter
	}

	level := log.InfoLevel
	switch cfg.Get().Logger.Level {
	case "debug":
		level = log.DebugLevel
	case "info":
		level = log.InfoLevel
		// case "warn":
		// 	level = log.WarnLevel
		// case "error":
		// 	level = log.ErrorLevel
		// case "fatal":
		// 	level = log.FatalLevel
	}

	handler := log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    true,
		ReportTimestamp: true,
		TimeFormat:      time.Kitchen,
		Prefix:          "Soulsolid",
		Formatter:       formatter,
		Level:           level,
	})

	logger := slog.New(handler)
	logger.Info("Logger initialized", "time", time.Now().Format(time.RFC3339))
	return logger
}

func Dup(logger *slog.Logger, msg string, args ...any) {
	logger.Info("[DUP] "+msg, args...)
}
