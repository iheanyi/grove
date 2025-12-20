package worktree

import (
	"regexp"
	"strings"
)

var (
	// Match any character that's not alphanumeric or hyphen
	invalidChars = regexp.MustCompile(`[^a-z0-9-]`)
	// Match multiple consecutive hyphens
	multipleHyphens = regexp.MustCompile(`-+`)
)

// Sanitize converts a branch name to a URL-safe name
// Examples:
//   - "feature/auth" -> "feature-auth"
//   - "bugfix/JIRA-123" -> "bugfix-jira-123"
//   - "feature/user_profile" -> "feature-user-profile"
//   - "main" -> "main"
func Sanitize(name string) string {
	// Convert to lowercase
	result := strings.ToLower(name)

	// Replace common separators with hyphens
	result = strings.ReplaceAll(result, "/", "-")
	result = strings.ReplaceAll(result, "_", "-")
	result = strings.ReplaceAll(result, ".", "-")

	// Remove any remaining invalid characters
	result = invalidChars.ReplaceAllString(result, "")

	// Collapse multiple hyphens into one
	result = multipleHyphens.ReplaceAllString(result, "-")

	// Trim leading/trailing hyphens
	result = strings.Trim(result, "-")

	// If empty after sanitization, use a default
	if result == "" {
		result = "default"
	}

	return result
}

// IsValidName checks if a name is valid for use in URLs
func IsValidName(name string) bool {
	if name == "" {
		return false
	}

	// Must start with a letter
	if !strings.ContainsAny(string(name[0]), "abcdefghijklmnopqrstuvwxyz") {
		return false
	}

	// Must only contain lowercase letters, numbers, and hyphens
	for _, c := range name {
		if !strings.ContainsRune("abcdefghijklmnopqrstuvwxyz0123456789-", c) {
			return false
		}
	}

	// Must not end with a hyphen
	if name[len(name)-1] == '-' {
		return false
	}

	// Must not contain consecutive hyphens
	if strings.Contains(name, "--") {
		return false
	}

	return true
}
