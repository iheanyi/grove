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
