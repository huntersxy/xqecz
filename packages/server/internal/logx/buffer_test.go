package logx

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestBufferSnapshotKeepsNewestEntries(t *testing.T) {
	b := NewBuffer(2)
	b.Append(Entry{Message: "one"})
	b.Append(Entry{Message: "two"})
	b.Append(Entry{Message: "three"})
	b.Append(Entry{Message: "four"})

	entries, latest, truncated := b.Snapshot(0, 10)
	if latest != 4 || truncated || len(entries) != 2 {
		t.Fatalf("snapshot metadata = latest %d truncated %v len %d", latest, truncated, len(entries))
	}
	if entries[0].ID != 3 || entries[1].Message != "four" {
		t.Fatalf("snapshot entries = %#v", entries)
	}
	entries, _, truncated = b.Snapshot(1, 10)
	if !truncated || len(entries) != 2 || entries[0].ID != 3 {
		t.Fatalf("stale cursor snapshot = entries %#v truncated %v", entries, truncated)
	}
}

func TestHandlerCopiesRenderedLineToBuffer(t *testing.T) {
	var out bytes.Buffer
	b := NewBuffer(4)
	logger := slog.New(NewWithBuffer(&out, b))
	logger.Warn("slow request", "size", 1024)

	entries, _, _ := b.Snapshot(0, 4)
	if len(entries) != 1 || entries[0].Level != "WARN" || entries[0].Message != "slow request" {
		t.Fatalf("buffered entry = %#v", entries)
	}
	if !strings.Contains(entries[0].Line, "size=1.0KB") || !strings.Contains(out.String(), entries[0].Line) {
		t.Fatalf("rendered line missing: %q", entries[0].Line)
	}
}

func TestHandlerBufferStripsTerminalColor(t *testing.T) {
	var out bytes.Buffer
	b := NewBuffer(2)
	logger := slog.New(newHandler(&out, slog.LevelInfo, true, b))
	logger.Info("colored")
	entries, _, _ := b.Snapshot(0, 2)
	if len(entries) != 1 || strings.Contains(entries[0].Line, "\x1b[") {
		t.Fatalf("buffer retained terminal escape sequence: %#v", entries)
	}
}

func TestHandlerRedactsSensitiveAttributes(t *testing.T) {
	var out bytes.Buffer
	b := NewBuffer(2)
	slog.New(NewWithBuffer(&out, b)).Info("request", "api_key", "private", "status", 200)
	entries, _, _ := b.Snapshot(0, 2)
	if len(entries) != 1 || strings.Contains(entries[0].Line, "private") || !strings.Contains(entries[0].Line, "api_key=[REDACTED]") {
		t.Fatalf("sensitive attribute was not redacted: %#v", entries)
	}
}
