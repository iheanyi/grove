package project

import (
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents a .grove.yaml project configuration
type Config struct {
	// Name overrides the auto-detected worktree name
	Name string `yaml:"name,omitempty"`

	// Command is the default command to run (for single-service projects)
	Command string `yaml:"command,omitempty"`

	// Port overrides the hash-based port allocation
	Port int `yaml:"port,omitempty"`

	// Env contains environment variables to set
	Env map[string]string `yaml:"env,omitempty"`

	// HealthCheck configures health checking
	HealthCheck HealthCheckConfig `yaml:"health_check,omitempty"`

	// Hooks defines lifecycle hooks
	Hooks HooksConfig `yaml:"hooks,omitempty"`

	// Services defines multiple services (like docker-compose)
	Services map[string]ServiceConfig `yaml:"services,omitempty"`

	// DependsOn defines service dependencies
	DependsOn map[string][]string `yaml:"depends_on,omitempty"`
}

// HealthCheckConfig configures health checking
type HealthCheckConfig struct {
	// Path is the HTTP path to check (e.g., "/health")
	Path string `yaml:"path,omitempty"`

	// Timeout is how long to wait for the health check
	Timeout time.Duration `yaml:"timeout,omitempty"`

	// Interval is how often to check health
	Interval time.Duration `yaml:"interval,omitempty"`
}

// HooksConfig defines lifecycle hooks
type HooksConfig struct {
	// BeforeStart runs before the server starts
	BeforeStart []string `yaml:"before_start,omitempty"`

	// AfterStart runs after the server starts
	AfterStart []string `yaml:"after_start,omitempty"`

	// BeforeStop runs before the server stops
	BeforeStop []string `yaml:"before_stop,omitempty"`
}

// ServiceConfig defines a single service in a multi-service project
type ServiceConfig struct {
	// Command is the command to run
	Command string `yaml:"command"`

	// Port is the port for this service (optional for non-web services)
	Port int `yaml:"port,omitempty"`

	// Env contains environment variables
	Env map[string]string `yaml:"env,omitempty"`

	// HealthCheck configures health checking for this service
	HealthCheck HealthCheckConfig `yaml:"health_check,omitempty"`

	// Hooks defines lifecycle hooks for this service
	Hooks HooksConfig `yaml:"hooks,omitempty"`
}

// ConfigFileName is the name of the project config file
const ConfigFileName = ".grove.yaml"

// Load loads the project configuration from the given directory
func Load(dir string) (*Config, error) {
	path := filepath.Join(dir, ConfigFileName)
	return LoadFile(path)
}

// LoadFile loads the project configuration from a specific file
func LoadFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	// Set defaults
	if cfg.HealthCheck.Timeout == 0 {
		cfg.HealthCheck.Timeout = 30 * time.Second
	}
	if cfg.HealthCheck.Interval == 0 {
		cfg.HealthCheck.Interval = 2 * time.Second
	}

	return cfg, nil
}

// Exists checks if a .grove.yaml file exists in the given directory
func Exists(dir string) bool {
	path := filepath.Join(dir, ConfigFileName)
	_, err := os.Stat(path)
	return err == nil
}

// Save saves the configuration to the given directory
func (c *Config) Save(dir string) error {
	path := filepath.Join(dir, ConfigFileName)
	return c.SaveFile(path)
}

// SaveFile saves the configuration to a specific file
func (c *Config) SaveFile(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// IsSingleService returns true if this is a single-service project
func (c *Config) IsSingleService() bool {
	return len(c.Services) == 0
}

// GetEffectiveName returns the name to use (either explicit or auto-detected)
func (c *Config) GetEffectiveName(autoDetected string) string {
	if c.Name != "" {
		return c.Name
	}
	return autoDetected
}
