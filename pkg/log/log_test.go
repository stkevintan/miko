package log

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestNew_JSON_EmitsStructuredFields(t *testing.T) {
	var buf bytes.Buffer

	cfg := &Config{Level: "debug", Format: "json", Stdout: true}
	l := New(cfg)
	// overwrite handler output to buffer for test determinism
	l.l = slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	l.Infow("hello", "user", "kevin", "count", 3)

	out := buf.String()
	if !strings.Contains(out, `"level":"INFO"`) {
		t.Fatalf("expected level INFO in output, got: %s", out)
	}
	if !strings.Contains(out, `"msg":"hello"`) {
		t.Fatalf("expected msg in output, got: %s", out)
	}
	if !strings.Contains(out, `"user":"kevin"`) || !strings.Contains(out, `"count":3`) {
		t.Fatalf("expected attrs in output, got: %s", out)
	}
}

func TestPrintfStyleStillWorks(t *testing.T) {
	var buf bytes.Buffer
	l := New(&Config{Level: "debug", Format: "json", Stdout: true})
	l.l = slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	l.Info("hi %s", "world")

	out := buf.String()
	if !strings.Contains(out, `"msg":"hi world"`) {
		t.Fatalf("expected formatted msg, got: %s", out)
	}
}
