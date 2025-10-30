package logger

import (
	"io"
	"log/slog"
	"os"
)

const (
	EMPTY   = ""
	DEBUG   = "debug"
	INFO    = "info"
	WARN    = "warn"
	ERROR   = "error"
	JSON    = "json"
	SERVICE = "service"
)

type Logger struct {
	*slog.Logger
}

type Config struct {
	Level     string
	Format    string
	Output    io.Writer
	AddSource bool
	Service   string
}

func New(cfg Config) *Logger {
	if cfg.Output == nil {
		cfg.Output = os.Stdout
	}
	if cfg.Format == EMPTY {
		cfg.Format = JSON
	}
	if cfg.Level == EMPTY {
		cfg.Level = INFO
	}
	var level slog.Level
	switch cfg.Level {
	case DEBUG:
		level = slog.LevelDebug
	case INFO:
		level = slog.LevelInfo
	case WARN:
		level = slog.LevelWarn
	case ERROR:
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: cfg.AddSource,
	}
	if cfg.Format == JSON {
		handler = slog.NewJSONHandler(cfg.Output, opts)
	} else {
		handler = slog.NewTextHandler(cfg.Output, opts)
	}

	if cfg.Service != EMPTY {
		handler = handler.WithAttrs([]slog.Attr{
			slog.String(SERVICE, cfg.Service),
		})
	}

	return &Logger{Logger: slog.New(handler)}
}

// Fatal logs a critical error and exits the application with status code 1
// Use this for unrecoverable errors that prevent the application from starting or continuing
func (l *Logger) Fatal(msg string, args ...any) {
	l.Error(msg, args...)
	os.Exit(1)
}
