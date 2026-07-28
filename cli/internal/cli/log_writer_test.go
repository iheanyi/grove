package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunLogWriterCapsFileAndKeepsNewestContent(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "server.log")
	maxBytes := int64(64)

	input := strings.NewReader(strings.Repeat("old\n", 30) + "newest-line\n")
	if err := runLogWriter(logFile, maxBytes, input); err != nil {
		t.Fatalf("run log writer: %v", err)
	}

	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}

	if int64(len(data)) > maxBytes {
		t.Fatalf("log file grew past cap: got %d bytes, want <= %d", len(data), maxBytes)
	}
	if !strings.Contains(string(data), "newest-line\n") {
		t.Fatalf("log file did not retain newest content: %q", string(data))
	}
}

func TestCompactOpenLogFileCreatesHeadroomAtCap(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "server.log")
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		t.Fatalf("open log file: %v", err)
	}
	t.Cleanup(func() {
		if err := file.Close(); err != nil {
			t.Errorf("close log file: %v", err)
		}
	})

	if _, err := file.WriteString("0123456789"); err != nil {
		t.Fatalf("seed log file: %v", err)
	}

	trimmed, err := compactOpenLogFile(file, 8, 6)
	if err != nil {
		t.Fatalf("compact log file: %v", err)
	}
	if !trimmed {
		t.Fatal("expected over-cap log file to be compacted")
	}

	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("read compacted log file: %v", err)
	}
	if got, want := string(data), "456789"; got != want {
		t.Fatalf("compacted content = %q, want %q", got, want)
	}

	if _, err := file.WriteString("ab"); err != nil {
		t.Fatalf("append within headroom: %v", err)
	}
	trimmed, err = compactOpenLogFile(file, 8, 6)
	if err != nil {
		t.Fatalf("check log file within cap: %v", err)
	}
	if trimmed {
		t.Fatal("log file compacted again before reaching the cap")
	}
}

func TestParseLogSize(t *testing.T) {
	tests := map[string]int64{
		"":     100 * 1024 * 1024,
		"10MB": 10 * 1024 * 1024,
		"5M":   5 * 1024 * 1024,
		"2GB":  2 * 1024 * 1024 * 1024,
		"512K": 512 * 1024,
		"123":  123,
	}

	for input, want := range tests {
		got, err := parseLogSize(input)
		if err != nil {
			t.Fatalf("parseLogSize(%q) returned error: %v", input, err)
		}
		if got != want {
			t.Fatalf("parseLogSize(%q) = %d, want %d", input, got, want)
		}
	}
}
