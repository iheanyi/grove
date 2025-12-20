package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestURLModeConstants(t *testing.T) {
	// Verify URL mode constants are as expected
	if URLModePort != "port" {
		t.Errorf("URLModePort = %q, want %q", URLModePort, "port")
	}
	if URLModeSubdomain != "subdomain" {
		t.Errorf("URLModeSubdomain = %q, want %q", URLModeSubdomain, "subdomain")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := Default()

	// Verify default URL mode is port
	if cfg.URLMode != URLModePort {
		t.Errorf("Default URLMode = %q, want %q", cfg.URLMode, URLModePort)
	}

	// Verify port range defaults
	if cfg.PortMin != 3000 {
		t.Errorf("Default PortMin = %d, want %d", cfg.PortMin, 3000)
	}
	if cfg.PortMax != 3999 {
		t.Errorf("Default PortMax = %d, want %d", cfg.PortMax, 3999)
	}

	// Verify TLD default
	if cfg.TLD != "localhost" {
		t.Errorf("Default TLD = %q, want %q", cfg.TLD, "localhost")
	}
}

func TestServerURL_PortMode(t *testing.T) {
	cfg := Default()
	cfg.URLMode = URLModePort

	tests := []struct {
		name     string
		server   string
		port     int
		expected string
	}{
		{
			name:     "standard port",
			server:   "myapp",
			port:     3000,
			expected: "http://localhost:3000",
		},
		{
			name:     "different port",
			server:   "feature-auth",
			port:     3042,
			expected: "http://localhost:3042",
		},
		{
			name:     "server name with dashes",
			server:   "my-cool-feature",
			port:     3100,
			expected: "http://localhost:3100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cfg.ServerURL(tt.server, tt.port)
			if result != tt.expected {
				t.Errorf("ServerURL(%q, %d) = %q, want %q", tt.server, tt.port, result, tt.expected)
			}
		})
	}
}

func TestServerURL_SubdomainMode(t *testing.T) {
	cfg := Default()
	cfg.URLMode = URLModeSubdomain
	cfg.TLD = "localhost"

	tests := []struct {
		name     string
		server   string
		port     int
		expected string
	}{
		{
			name:     "standard server",
			server:   "myapp",
			port:     3000,
			expected: "https://myapp.localhost",
		},
		{
			name:     "feature branch",
			server:   "feature-auth",
			port:     3042,
			expected: "https://feature-auth.localhost",
		},
		{
			name:     "port ignored in subdomain mode",
			server:   "test",
			port:     9999,
			expected: "https://test.localhost",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cfg.ServerURL(tt.server, tt.port)
			if result != tt.expected {
				t.Errorf("ServerURL(%q, %d) = %q, want %q", tt.server, tt.port, result, tt.expected)
			}
		})
	}
}

func TestSubdomainURL(t *testing.T) {
	tests := []struct {
		name     string
		mode     URLMode
		server   string
		expected string
	}{
		{
			name:     "subdomain mode returns wildcard URL",
			mode:     URLModeSubdomain,
			server:   "myapp",
			expected: "https://*.myapp.localhost",
		},
		{
			name:     "port mode returns empty string",
			mode:     URLModePort,
			server:   "myapp",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			cfg.URLMode = tt.mode

			result := cfg.SubdomainURL(tt.server)
			if result != tt.expected {
				t.Errorf("SubdomainURL(%q) = %q, want %q", tt.server, result, tt.expected)
			}
		})
	}
}

func TestIsSubdomainMode(t *testing.T) {
	tests := []struct {
		name     string
		mode     URLMode
		expected bool
	}{
		{
			name:     "port mode returns false",
			mode:     URLModePort,
			expected: false,
		},
		{
			name:     "subdomain mode returns true",
			mode:     URLModeSubdomain,
			expected: true,
		},
		{
			name:     "empty mode returns false",
			mode:     "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			cfg.URLMode = tt.mode

			result := cfg.IsSubdomainMode()
			if result != tt.expected {
				t.Errorf("IsSubdomainMode() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestItoa(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{10, "10"},
		{100, "100"},
		{3000, "3000"},
		{3999, "3999"},
		{12345, "12345"},
		{-1, "-1"},
		{-100, "-100"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := itoa(tt.input)
			if result != tt.expected {
				t.Errorf("itoa(%d) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLoadConfig_DefaultsWhenNoFile(t *testing.T) {
	// Use a temp directory that doesn't contain a config file
	tmpDir := t.TempDir()
	nonExistentPath := filepath.Join(tmpDir, "nonexistent.yaml")

	cfg, err := Load(nonExistentPath)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	// Should return defaults
	if cfg.URLMode != URLModePort {
		t.Errorf("URLMode = %q, want %q", cfg.URLMode, URLModePort)
	}
	if cfg.PortMin != 3000 {
		t.Errorf("PortMin = %d, want %d", cfg.PortMin, 3000)
	}
}

func TestLoadConfig_OverridesDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Write a config with custom values
	configContent := `
url_mode: subdomain
port_min: 4000
port_max: 4999
tld: test.localhost
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if cfg.URLMode != URLModeSubdomain {
		t.Errorf("URLMode = %q, want %q", cfg.URLMode, URLModeSubdomain)
	}
	if cfg.PortMin != 4000 {
		t.Errorf("PortMin = %d, want %d", cfg.PortMin, 4000)
	}
	if cfg.PortMax != 4999 {
		t.Errorf("PortMax = %d, want %d", cfg.PortMax, 4999)
	}
	if cfg.TLD != "test.localhost" {
		t.Errorf("TLD = %q, want %q", cfg.TLD, "test.localhost")
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create a config with custom values
	cfg := Default()
	cfg.URLMode = URLModeSubdomain
	cfg.PortMin = 5000
	cfg.PortMax = 5999

	// Save it
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Load it back
	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.URLMode != cfg.URLMode {
		t.Errorf("URLMode = %q, want %q", loaded.URLMode, cfg.URLMode)
	}
	if loaded.PortMin != cfg.PortMin {
		t.Errorf("PortMin = %d, want %d", loaded.PortMin, cfg.PortMin)
	}
	if loaded.PortMax != cfg.PortMax {
		t.Errorf("PortMax = %d, want %d", loaded.PortMax, cfg.PortMax)
	}
}

func TestServerURL_CustomTLD(t *testing.T) {
	cfg := Default()
	cfg.URLMode = URLModeSubdomain
	cfg.TLD = "dev.local"

	result := cfg.ServerURL("myapp", 3000)
	expected := "https://myapp.dev.local"

	if result != expected {
		t.Errorf("ServerURL() = %q, want %q", result, expected)
	}
}
