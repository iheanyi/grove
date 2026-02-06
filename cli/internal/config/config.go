package config

import (
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/adrg/xdg"
	"gopkg.in/yaml.v3"
)

// URLMode represents how server URLs are generated
type URLMode string

const (
	// URLModePort uses direct port access (http://localhost:PORT)
	URLModePort URLMode = "port"
	// URLModeSubdomain uses subdomain-based routing (https://name.localhost)
	URLModeSubdomain URLMode = "subdomain"
)

// Config holds the global configuration for grove
type Config struct {
	// Port allocation
	PortMin int `yaml:"port_min"`
	PortMax int `yaml:"port_max"`

	// Worktree management
	// WorktreesDir is the centralized directory for worktrees.
	// When set, new worktrees are created in: <worktrees_dir>/<project>/<branch>
	// When empty (default), worktrees are created as siblings to the main repo.
	WorktreesDir string `yaml:"worktrees_dir"`

	// URL mode: "port" (default) or "subdomain"
	// - port: http://localhost:PORT (simpler, no proxy needed)
	// - subdomain: https://name.localhost (requires proxy, may conflict with app subdomains)
	URLMode URLMode `yaml:"url_mode"`

	// Domain settings (only used in subdomain mode)
	TLD string `yaml:"tld"`

	// Proxy ports (only used in subdomain mode)
	ProxyHTTPPort  int `yaml:"proxy_http_port"`
	ProxyHTTPSPort int `yaml:"proxy_https_port"`

	// Log settings
	LogDir       string `yaml:"log_dir"`
	LogMaxSize   string `yaml:"log_max_size"`
	LogRetention string `yaml:"log_retention"`

	// Server behavior
	IdleTimeout        time.Duration `yaml:"idle_timeout"`
	HealthCheckTimeout time.Duration `yaml:"health_check_timeout"`

	// TUI settings
	TUI TUIConfig `yaml:"tui"`

	// Notifications
	Notifications NotificationConfig `yaml:"notifications"`
}

// TUIConfig holds TUI-specific settings
type TUIConfig struct {
	ShowLogs bool `yaml:"show_logs"`
	LogLines int  `yaml:"log_lines"`
}

// NotificationConfig holds notification settings
type NotificationConfig struct {
	Enabled    bool `yaml:"enabled"`
	OnStart    bool `yaml:"on_start"`
	OnStop     bool `yaml:"on_stop"`
	OnCrash    bool `yaml:"on_crash"`
	OnIdleStop bool `yaml:"on_idle_stop"`
}

// Default returns a Config with default values
func Default() *Config {
	return &Config{
		PortMin:            3000,
		PortMax:            3999,
		URLMode:            URLModePort,
		TLD:                "localhost",
		ProxyHTTPPort:      80,
		ProxyHTTPSPort:     443,
		LogDir:             filepath.Join(xdg.ConfigHome, "grove", "logs"),
		LogMaxSize:         "10MB",
		LogRetention:       "7d",
		IdleTimeout:        30 * time.Minute,
		HealthCheckTimeout: 60 * time.Second,
		TUI: TUIConfig{
			ShowLogs: true,
			LogLines: 10,
		},
		Notifications: NotificationConfig{
			Enabled:    true,
			OnStart:    true,
			OnStop:     true,
			OnCrash:    true,
			OnIdleStop: true,
		},
	}
}

// ConfigDir returns the grove configuration directory
func ConfigDir() string {
	return filepath.Join(xdg.ConfigHome, "grove")
}

// ConfigPath returns the path to the config file
func ConfigPath() string {
	return filepath.Join(ConfigDir(), "config.yaml")
}

// RegistryPath returns the path to the registry file
func RegistryPath() string {
	return filepath.Join(ConfigDir(), "registry.json")
}

// SocketPath returns the path to the Unix socket
func SocketPath() string {
	return filepath.Join(os.TempDir(), "grove.sock")
}

// Load loads configuration from the specified file, or the default location
func Load(path string) (*Config, error) {
	if path == "" {
		path = ConfigPath()
	}

	cfg := Default()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// No config file, use defaults
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Save saves the configuration to the specified file
func (c *Config) Save(path string) error {
	if path == "" {
		path = ConfigPath()
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// EnsureDirectories creates necessary directories
func EnsureDirectories() error {
	dirs := []string{
		ConfigDir(),
		filepath.Join(ConfigDir(), "logs"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return nil
}

// ServerURL returns the URL for a server based on the configured URL mode
func (c *Config) ServerURL(name string, port int) string {
	if c.URLMode == URLModeSubdomain {
		return "https://" + name + "." + c.TLD
	}
	// Default to port mode
	return "http://localhost:" + strconv.Itoa(port)
}

// SubdomainURL returns the wildcard subdomain URL (only meaningful in subdomain mode)
func (c *Config) SubdomainURL(name string) string {
	if c.URLMode == URLModeSubdomain {
		return "https://*." + name + "." + c.TLD
	}
	return ""
}

// IsSubdomainMode returns true if using subdomain-based URLs
func (c *Config) IsSubdomainMode() bool {
	return c.URLMode == URLModeSubdomain
}
