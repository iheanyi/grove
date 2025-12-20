package loghighlight

import (
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func init() {
	// Force color output for tests
	lipgloss.SetColorProfile(termenv.TrueColor)
	os.Setenv("CLICOLOR_FORCE", "1")
}

func TestHighlight_LogLevels(t *testing.T) {
	tests := []struct {
		input    string
		contains string
	}{
		{"ERROR: something failed", "ERROR"},
		{"WARN: something suspicious", "WARN"},
		{"INFO: server started", "INFO"},
		{"DEBUG: verbose output", "DEBUG"},
	}

	for _, tt := range tests {
		result := Highlight(tt.input)
		if !strings.Contains(result, tt.contains) {
			t.Errorf("Highlight(%q) should contain %q, got %q", tt.input, tt.contains, result)
		}
	}
}

func TestHighlight_HTTPMethods(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE"}

	for _, method := range methods {
		input := "Started " + method + " /users"
		result := Highlight(input)
		if !strings.Contains(result, method) {
			t.Errorf("Highlight(%q) should contain %q, got %q", input, method, result)
		}
	}
}

func TestHighlight_StatusCodes(t *testing.T) {
	tests := []struct {
		input string
		code  string
	}{
		{"Completed 200 OK in 12ms", "200"},
		{"Completed 201 Created", "201"},
		{"Completed 302 Found", "302"},
		{"Completed 404 Not Found", "404"},
		{"Completed 500 Internal Server Error", "500"},
	}

	for _, tt := range tests {
		result := Highlight(tt.input)
		if !strings.Contains(result, tt.code) {
			t.Errorf("Highlight(%q) should contain status code %q, got %q", tt.input, tt.code, result)
		}
	}
}

func TestHighlight_Timestamps(t *testing.T) {
	tests := []string{
		"2025-01-15T10:30:15Z INFO started",
		"2025-01-15 10:30:15 -0500 INFO started",
		"[2025-01-15 10:30:15] INFO started",
		"10:30:15.123 INFO started",
	}

	for _, input := range tests {
		result := Highlight(input)
		// Verify content is preserved
		if !strings.Contains(result, "INFO") {
			t.Errorf("Highlight(%q) should preserve content", input)
		}
	}
}

func TestHighlight_Durations(t *testing.T) {
	tests := []struct {
		input    string
		contains string
	}{
		{"Completed in 12.3ms", "12.3ms"},
		{"ActiveRecord: 5.2ms", "5.2ms"},
		{"Views: 8.1ms", "8.1ms"},
		{"Request took 1.5s", "1.5s"},
	}

	for _, tt := range tests {
		result := Highlight(tt.input)
		if !strings.Contains(result, tt.contains) {
			t.Errorf("Highlight(%q) should contain %q, got %q", tt.input, tt.contains, result)
		}
	}
}

func TestHighlight_RailsPatterns(t *testing.T) {
	tests := []struct {
		input    string
		contains string
	}{
		{"Started GET / for 127.0.0.1", "Started"},
		{"Processing by UsersController#index", "UsersController#index"},
		{"Completed 200 OK in 12ms", "Completed"},
		{"Rendered users/index.html.erb", "Rendered"},
	}

	for _, tt := range tests {
		result := Highlight(tt.input)
		if !strings.Contains(result, tt.contains) {
			t.Errorf("Highlight(%q) should contain %q, got %q", tt.input, tt.contains, result)
		}
	}
}

func TestHighlight_JSON(t *testing.T) {
	input := `{"level":"info","method":"GET","status":200,"duration":12.5}`
	result := Highlight(input)

	// Should contain the original JSON content
	if !strings.Contains(result, "level") {
		t.Errorf("Highlight should preserve JSON keys, got %q", result)
	}
	if !strings.Contains(result, "info") {
		t.Errorf("Highlight should preserve JSON values, got %q", result)
	}
	if !strings.Contains(result, "200") {
		t.Errorf("Highlight should preserve numbers, got %q", result)
	}
}

func TestHighlight_PreservesContent(t *testing.T) {
	// Highlighting should not remove any text content
	inputs := []string{
		"plain text without patterns",
		"ERROR: user not found in database",
		"Started GET /api/users/123 for 127.0.0.1",
	}

	for _, input := range inputs {
		result := Highlight(input)
		// Remove ANSI codes to check content preservation
		clean := stripANSI(result)
		if clean != input {
			t.Errorf("Highlight changed content:\n  got:  %q\n  want: %q", clean, input)
		}
	}
}

func TestHighlightLines(t *testing.T) {
	lines := []string{
		"INFO: line 1",
		"ERROR: line 2",
		"DEBUG: line 3",
	}

	results := HighlightLines(lines)
	if len(results) != len(lines) {
		t.Errorf("HighlightLines returned %d lines, want %d", len(results), len(lines))
	}

	for i, result := range results {
		if !strings.Contains(result, "line") {
			t.Errorf("HighlightLines[%d] should preserve content, got %q", i, result)
		}
	}
}

func TestHighlight_EmptyString(t *testing.T) {
	result := Highlight("")
	if result != "" {
		t.Errorf("Highlight of empty string should be empty, got %q", result)
	}
}

func TestHighlight_NoPatterns(t *testing.T) {
	input := "just some regular text without any patterns"
	result := Highlight(input)
	clean := stripANSI(result)
	if clean != input {
		t.Errorf("Highlight without patterns should preserve text:\n  got:  %q\n  want: %q", clean, input)
	}
}

func TestHighlight_MixedContent(t *testing.T) {
	input := "2025-01-15T10:30:15Z INFO Started GET /users Completed 200 OK in 12.3ms"
	result := Highlight(input)

	// All content should be preserved
	expected := []string{"2025-01-15T10:30:15Z", "INFO", "GET", "/users", "200", "12.3ms"}
	for _, exp := range expected {
		if !strings.Contains(result, exp) {
			t.Errorf("Highlight(%q) should contain %q", input, exp)
		}
	}
}

// stripANSI removes ANSI escape codes from a string
func stripANSI(s string) string {
	result := s
	for strings.Contains(result, "\x1b[") {
		start := strings.Index(result, "\x1b[")
		end := start + 2
		for end < len(result) && result[end] != 'm' {
			end++
		}
		if end < len(result) {
			result = result[:start] + result[end+1:]
		} else {
			break
		}
	}
	return result
}
