package log

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

// Config mirrors the basic shape of netease-cloud-music/pkg/log config,
// but uses Go's structured logger (log/slog) under the hood.
//
// Format: "json" (default) or "text".
// Level: "debug" | "info" | "warn" | "error".
// Stdout: when true, logs go to stdout (else stderr).
// File: optional file path (append); if set and Stdout is true, logs go to both.
type Config struct {
	Level     string `json:"level" mapstructure:"level"`
	Format    string `json:"format" mapstructure:"format"`
	Stdout    bool   `json:"stdout" mapstructure:"stdout"`
	File      string `json:"file" mapstructure:"file"`
	AddSource bool   `json:"addSource" mapstructure:"addSource"`
}

// Logger is a thin wrapper around slog.Logger that keeps a printf-ish API
// for existing call-sites, while still emitting structured records.
type Logger struct {
	l *slog.Logger
}

// Default is the process-wide default logger, matching the upstream pattern
// used in this repo (e.g. `log.Default = log.New(...)`).
var Default *Logger

func init() {
	Default = New(&Config{Level: "info", Format: "json", Stdout: true})
}

// New constructs a Logger from Config.
func New(cfg *Config) *Logger {
	if cfg == nil {
		cfg = &Config{}
	}

	level := parseLevel(cfg.Level)
	format := strings.ToLower(strings.TrimSpace(cfg.Format))
	if format == "" {
		format = "json"
	}

	var base io.Writer
	if cfg.Stdout {
		base = os.Stdout
	} else {
		base = os.Stderr
	}

	w := base
	if strings.TrimSpace(cfg.File) != "" {
		path := os.ExpandEnv(strings.TrimSpace(cfg.File))
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err == nil {
			if cfg.Stdout {
				w = io.MultiWriter(base, f)
			} else {
				w = f
			}
		}
	}

	hopts := &slog.HandlerOptions{
		Level:     level,
		AddSource: cfg.AddSource,
	}

	var h slog.Handler
	switch format {
	case "text":
		h = slog.NewTextHandler(w, hopts)
	default:
		h = slog.NewJSONHandler(w, hopts)
	}

	return &Logger{l: slog.New(h)}
}

func parseLevel(s string) slog.Level {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	case "info", "":
		return slog.LevelInfo
	default:
		return slog.LevelInfo
	}
}

// With returns a new logger with additional structured attributes.
func (l *Logger) With(args ...any) *Logger {
	if l == nil || l.l == nil {
		return Default.With(args...)
	}
	return &Logger{l: l.l.With(args...)}
}

// Named returns a new logger with a `logger` attribute.
func (l *Logger) Named(name string) *Logger {
	return l.With("logger", name)
}

// Debug logs a DEBUG record.
// If `format` contains '%' it behaves like printf.
// Otherwise, if args look like key/value pairs, it logs them as structured attrs.
func (l *Logger) Debug(format string, args ...any) { l.log(slog.LevelDebug, format, args...) }

// Info logs an INFO record.
func (l *Logger) Info(format string, args ...any) { l.log(slog.LevelInfo, format, args...) }

// Warn logs a WARN record.
func (l *Logger) Warn(format string, args ...any) { l.log(slog.LevelWarn, format, args...) }

// Error logs an ERROR record.
func (l *Logger) Error(format string, args ...any) { l.log(slog.LevelError, format, args...) }

// Debugw logs a DEBUG record with explicit key/value attrs.
func (l *Logger) Debugw(msg string, args ...any) { l.kv(slog.LevelDebug, msg, args...) }

// Infow logs an INFO record with explicit key/value attrs.
func (l *Logger) Infow(msg string, args ...any) { l.kv(slog.LevelInfo, msg, args...) }

// Warnw logs a WARN record with explicit key/value attrs.
func (l *Logger) Warnw(msg string, args ...any) { l.kv(slog.LevelWarn, msg, args...) }

// Errorw logs an ERROR record with explicit key/value attrs.
func (l *Logger) Errorw(msg string, args ...any) { l.kv(slog.LevelError, msg, args...) }

// Fatal logs an ERROR record and exits with code 1.
func (l *Logger) Fatal(format string, args ...any) {
	l.log(slog.LevelError, format, args...)
	os.Exit(1)
}

// Fatalf is an alias for Fatal.
func (l *Logger) Fatalf(format string, args ...any) { l.Fatal(format, args...) }

// Printf logs at INFO level (compat helper).
func (l *Logger) Printf(format string, args ...any) { l.Info(format, args...) }

// Println logs at INFO level (compat helper).
func (l *Logger) Println(args ...any) { l.Info("%s", fmt.Sprint(args...)) }

func (l *Logger) log(level slog.Level, format string, args ...any) {
	if l == nil || l.l == nil {
		Default.log(level, format, args...)
		return
	}

	format = strings.TrimSpace(format)
	if format == "" {
		format = "(empty message)"
	}

	// printf-style if format looks like printf.
	if strings.Contains(format, "%") {
		l.l.Log(context.Background(), level, fmt.Sprintf(format, args...))
		return
	}

	// structured if args look like key/value pairs.
	if looksLikeKeyValues(args) {
		l.l.Log(context.Background(), level, format, args...)
		return
	}

	// fallback: treat as fmt.Sprintf with space-joined args.
	if len(args) > 0 {
		l.l.Log(context.Background(), level, fmt.Sprintf("%s %s", format, fmt.Sprint(args...)))
		return
	}
	l.l.Log(context.Background(), level, format)
}

func (l *Logger) kv(level slog.Level, msg string, args ...any) {
	if !looksLikeKeyValues(args) {
		// keep behavior predictable: don't panic, but make it obvious.
		l.log(level, msg, "attrs_error", errors.New("expected key/value pairs"), "attrs", fmt.Sprint(args...))
		return
	}
	l.l.Log(context.Background(), level, msg, args...)
}

func looksLikeKeyValues(args []any) bool {
	if len(args) == 0 || len(args)%2 != 0 {
		return false
	}
	for i := 0; i < len(args); i += 2 {
		k, ok := args[i].(string)
		if !ok || strings.TrimSpace(k) == "" {
			return false
		}
	}
	return true
}

// ---- Global helpers using Default ----

func With(args ...any) *Logger          { return Default.With(args...) }
func Named(name string) *Logger         { return Default.Named(name) }
func Debug(format string, args ...any)  { Default.Debug(format, args...) }
func Info(format string, args ...any)   { Default.Info(format, args...) }
func Warn(format string, args ...any)   { Default.Warn(format, args...) }
func Error(format string, args ...any)  { Default.Error(format, args...) }
func Debugw(msg string, args ...any)    { Default.Debugw(msg, args...) }
func Infow(msg string, args ...any)     { Default.Infow(msg, args...) }
func Warnw(msg string, args ...any)     { Default.Warnw(msg, args...) }
func Errorw(msg string, args ...any)    { Default.Errorw(msg, args...) }
func Fatal(format string, args ...any)  { Default.Fatal(format, args...) }
func Fatalf(format string, args ...any) { Default.Fatalf(format, args...) }
func Printf(format string, args ...any) { Default.Printf(format, args...) }
func Println(args ...any)               { Default.Println(args...) }
