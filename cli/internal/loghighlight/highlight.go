// Package loghighlight provides syntax highlighting for log output
package loghighlight

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Colors for different log elements
var (
	// Log levels
	ErrorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Bold(true)
	WarnStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")).Bold(true)
	InfoStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#3B82F6"))
	DebugStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))

	// HTTP Methods
	GetStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")).Bold(true)
	PostStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#3B82F6")).Bold(true)
	PutStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")).Bold(true)
	PatchStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#EAB308")).Bold(true)
	DeleteStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Bold(true)

	// Status codes
	Status2xxStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")).Bold(true)
	Status3xxStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#3B82F6"))
	Status4xxStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")).Bold(true)
	Status5xxStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Bold(true)

	// Other elements
	TimestampStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
	DurationStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#A855F7"))
	NumberStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#06B6D4"))
	StringStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981"))
	KeyStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#3B82F6"))
	ControllerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#EAB308")).Bold(true)
	PathStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#8B5CF6"))
)

// Compiled regex patterns
var (
	// Timestamps
	timestampISO      = regexp.MustCompile(`\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?`)
	timestampBracket  = regexp.MustCompile(`\[\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\]`)
	timestampTime     = regexp.MustCompile(`\d{2}:\d{2}:\d{2}\.\d+`)

	// Log levels
	levelError = regexp.MustCompile(`(?i)\b(ERROR|FATAL|CRITICAL)\b`)
	levelWarn  = regexp.MustCompile(`(?i)\b(WARN|WARNING)\b`)
	levelInfo  = regexp.MustCompile(`(?i)\bINFO\b`)
	levelDebug = regexp.MustCompile(`(?i)\b(DEBUG|TRACE)\b`)

	// HTTP
	httpGet     = regexp.MustCompile(`\bGET\b`)
	httpPost    = regexp.MustCompile(`\bPOST\b`)
	httpPut     = regexp.MustCompile(`\bPUT\b`)
	httpPatch   = regexp.MustCompile(`\bPATCH\b`)
	httpDelete  = regexp.MustCompile(`\bDELETE\b`)

	// Status codes (only in status context)
	statusCode = regexp.MustCompile(`\b([2-5]\d{2})\b`)

	// Durations
	durationMs = regexp.MustCompile(`\d+\.?\d*\s*ms\b`)
	durationS  = regexp.MustCompile(`\d+\.?\d*\s*s\b`)

	// Rails specific
	railsStarted    = regexp.MustCompile(`^Started\b`)
	railsProcessing = regexp.MustCompile(`Processing by (\w+#\w+)`)
	railsCompleted  = regexp.MustCompile(`^Completed\b`)
	railsRendered   = regexp.MustCompile(`Rendered\s+[\w/]+\.[\w.]+`)
	railsAR         = regexp.MustCompile(`ActiveRecord:\s*\d+\.?\d*ms`)
	railsViews      = regexp.MustCompile(`Views:\s*\d+\.?\d*ms`)
	railsAlloc      = regexp.MustCompile(`Allocations:\s*\d+`)

	// JSON
	jsonKey  = regexp.MustCompile(`"(\w+)":\s*`)
	jsonBool = regexp.MustCompile(`\b(true|false|null)\b`)

	// URLs/paths
	urlPath = regexp.MustCompile(`"(/[^"]*)"`)
)

// Highlight applies syntax highlighting to a log line
func Highlight(line string) string {
	// Start with the original line
	result := line

	// Apply highlights using a replacement approach
	// We need to be careful about overlapping matches

	// Timestamps (dim them)
	result = highlightPattern(result, timestampISO, TimestampStyle)
	result = highlightPattern(result, timestampBracket, TimestampStyle)
	result = highlightPattern(result, timestampTime, TimestampStyle)

	// Log levels (high priority - bold colors)
	result = highlightPattern(result, levelError, ErrorStyle)
	result = highlightPattern(result, levelWarn, WarnStyle)
	result = highlightPattern(result, levelInfo, InfoStyle)
	result = highlightPattern(result, levelDebug, DebugStyle)

	// HTTP methods
	result = highlightPattern(result, httpGet, GetStyle)
	result = highlightPattern(result, httpPost, PostStyle)
	result = highlightPattern(result, httpPut, PutStyle)
	result = highlightPattern(result, httpPatch, PatchStyle)
	result = highlightPattern(result, httpDelete, DeleteStyle)

	// Rails patterns
	result = highlightPattern(result, railsStarted, InfoStyle)
	result = highlightPattern(result, railsCompleted, Status2xxStyle)
	result = highlightPattern(result, railsRendered, PathStyle)

	// Controller#action
	if matches := railsProcessing.FindStringSubmatch(result); len(matches) > 1 {
		result = strings.Replace(result, matches[1], ControllerStyle.Render(matches[1]), 1)
	}

	// Durations
	result = highlightPattern(result, durationMs, DurationStyle)
	result = highlightPattern(result, durationS, DurationStyle)
	result = highlightPattern(result, railsAR, DurationStyle)
	result = highlightPattern(result, railsViews, DurationStyle)
	result = highlightPattern(result, railsAlloc, NumberStyle)

	// Status codes (context-aware)
	if strings.Contains(line, "Completed") || strings.Contains(line, "HTTP") || strings.Contains(line, "status") {
		result = highlightStatusCodes(result, line)
	}

	// URL paths
	if matches := urlPath.FindAllStringSubmatch(result, -1); matches != nil {
		for _, match := range matches {
			if len(match) > 1 {
				result = strings.Replace(result, `"`+match[1]+`"`, `"`+PathStyle.Render(match[1])+`"`, 1)
			}
		}
	}

	// JSON highlighting (if line looks like JSON)
	if strings.Contains(line, "{") || strings.Contains(line, "[") {
		// JSON keys
		result = highlightJSONKeys(result)
		// Booleans/null
		result = highlightPattern(result, jsonBool, NumberStyle)
	}

	return result
}

func highlightPattern(s string, pattern *regexp.Regexp, style lipgloss.Style) string {
	return pattern.ReplaceAllStringFunc(s, func(match string) string {
		return style.Render(match)
	})
}

func highlightStatusCodes(result, original string) string {
	matches := statusCode.FindAllStringSubmatchIndex(original, -1)
	if matches == nil {
		return result
	}

	// Process matches in reverse order to preserve indices
	for i := len(matches) - 1; i >= 0; i-- {
		match := matches[i]
		if len(match) >= 4 {
			start, end := match[2], match[3]
			code := original[start:end]
			var style lipgloss.Style

			switch code[0] {
			case '2':
				style = Status2xxStyle
			case '3':
				style = Status3xxStyle
			case '4':
				style = Status4xxStyle
			case '5':
				style = Status5xxStyle
			default:
				continue
			}

			result = result[:start] + style.Render(code) + result[end:]
		}
	}

	return result
}

func highlightJSONKeys(s string) string {
	return jsonKey.ReplaceAllStringFunc(s, func(match string) string {
		// Extract just the key name
		key := strings.Trim(match, `":`)
		key = strings.TrimSpace(key)
		return `"` + KeyStyle.Render(key) + `": `
	})
}

// HighlightLines highlights multiple lines
func HighlightLines(lines []string) []string {
	result := make([]string, len(lines))
	for i, line := range lines {
		result[i] = Highlight(line)
	}
	return result
}
