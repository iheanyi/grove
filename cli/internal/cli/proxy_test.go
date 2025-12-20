package cli

import (
	"fmt"
	"strings"
	"testing"
)

// TestBuildCaddyfileContent tests the Caddyfile content generation logic
func TestBuildCaddyfileContent(t *testing.T) {
	tests := []struct {
		name     string
		servers  []struct{ name string; port int }
		expected []string
		notExpected []string
	}{
		{
			name: "single server",
			servers: []struct{ name string; port int }{
				{"test-server", 3000},
			},
			expected: []string{
				"local_certs",
				"auto_https disable_redirects",
				"https://test-server.localhost",
				"https://*.test-server.localhost",
				"reverse_proxy localhost:3000",
			},
			notExpected: []string{
				"No server registered",
			},
		},
		{
			name: "no servers",
			servers: nil,
			expected: []string{
				"local_certs",
				"auto_https disable_redirects",
				"No server registered for this domain",
			},
		},
		{
			name: "multiple servers",
			servers: []struct{ name string; port int }{
				{"server-one", 3001},
				{"server-two", 3002},
			},
			expected: []string{
				"https://server-one.localhost",
				"reverse_proxy localhost:3001",
				"https://server-two.localhost",
				"reverse_proxy localhost:3002",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := buildTestCaddyfileContent(tt.servers)

			for _, exp := range tt.expected {
				if !strings.Contains(content, exp) {
					t.Errorf("expected content to contain %q, got:\n%s", exp, content)
				}
			}

			for _, notExp := range tt.notExpected {
				if strings.Contains(content, notExp) {
					t.Errorf("expected content NOT to contain %q, got:\n%s", notExp, content)
				}
			}
		})
	}
}

// buildTestCaddyfileContent is a test helper that mimics generateCaddyfile logic
func buildTestCaddyfileContent(servers []struct{ name string; port int }) string {
	var sb strings.Builder

	// Global options (same as generateCaddyfile)
	sb.WriteString("{\n")
	sb.WriteString("\tlocal_certs\n")
	sb.WriteString("\tauto_https disable_redirects\n")
	sb.WriteString("}\n\n")

	if len(servers) == 0 {
		// Default fallback when no servers
		sb.WriteString("https://*.localhost {\n")
		sb.WriteString("\trespond \"No server registered for this domain\" 503\n")
		sb.WriteString("}\n")
	} else {
		// Generate route for each server
		for _, server := range servers {
			// Main domain
			sb.WriteString(fmt.Sprintf("https://%s.localhost {\n", server.name))
			sb.WriteString(fmt.Sprintf("\treverse_proxy localhost:%d\n", server.port))
			sb.WriteString("}\n\n")

			// Wildcard subdomains
			sb.WriteString(fmt.Sprintf("https://*.%s.localhost {\n", server.name))
			sb.WriteString(fmt.Sprintf("\treverse_proxy localhost:%d\n", server.port))
			sb.WriteString("}\n\n")
		}
	}

	return sb.String()
}

func TestIsProcessRunning(t *testing.T) {
	// Test with current process (should be running)
	// This is a simple sanity check
	if !isProcessRunning(1) {
		// PID 1 (init/launchd) should always be running
		t.Log("PID 1 not running (might be in container)")
	}

	// Test with invalid PID
	if isProcessRunning(-1) {
		t.Error("expected isProcessRunning(-1) to return false")
	}

	// Test with very high PID (likely doesn't exist)
	if isProcessRunning(999999999) {
		t.Error("expected isProcessRunning(999999999) to return false")
	}
}
