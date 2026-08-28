package logging

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"[PROJECT-NAME]/backend/internal/config"
)

func TestNewLogger_SuppressesBelowConfiguredLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := newLogger(config.Config{LogLevel: "info", LogFormat: "json"}, &buf)

	logger.Debug("debug message")
	if buf.Len() != 0 {
		t.Fatalf("expected debug message to be suppressed, got: %s", buf.String())
	}

	logger.Info("info message")
	if !strings.Contains(buf.String(), "info message") {
		t.Fatalf("expected info message to be logged, got: %s", buf.String())
	}
}

func TestNewLogger_JSONFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := newLogger(config.Config{LogLevel: "info", LogFormat: "json"}, &buf)

	logger.Info("hello", "key", "value")

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("expected valid JSON, got error: %v, body: %s", err, buf.String())
	}
	if entry["msg"] != "hello" || entry["key"] != "value" {
		t.Fatalf("unexpected decoded log entry: %+v", entry)
	}
}

func TestNewLogger_PrettyFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := newLogger(config.Config{LogLevel: "info", LogFormat: "pretty"}, &buf)

	logger.Info("hello", "key", "value")

	out := buf.String()
	if !strings.Contains(out, "hello") || !strings.Contains(out, "key=value") {
		t.Fatalf("unexpected pretty output: %s", out)
	}
}
