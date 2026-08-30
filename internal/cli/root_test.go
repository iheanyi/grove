package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCLIRejectsInvalidLogRetention(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("log_retention: forever\n"), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	previousCfgFile := cfgFile
	previousCfg := cfg
	previousCfgErr := cfgErr
	t.Cleanup(func() {
		cfgFile = previousCfgFile
		cfg = previousCfg
		cfgErr = previousCfgErr
		rootCmd.SetArgs(nil)
	})

	rootCmd.SetArgs([]string{"--config", configPath, "version"})
	err := Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want invalid log_retention error")
	}
	if !strings.Contains(err.Error(), `invalid log_retention "forever"`) {
		t.Fatalf("Execute() error = %q, want invalid value in message", err)
	}
}

func TestInvalidRetentionDoesNotBlockBoundedLogWriter(t *testing.T) {
	previousCfgErr := cfgErr
	t.Cleanup(func() { cfgErr = previousCfgErr })
	cfgErr = os.ErrInvalid

	if err := rootCmd.PersistentPreRunE(logWriterCmd, nil); err != nil {
		t.Fatalf("bounded log writer pre-run error = %v, want nil", err)
	}
}

func TestCLILifecycleRemovesExpiredStoppedLogs(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "logs")
	if err := os.Mkdir(logDir, 0755); err != nil {
		t.Fatalf("create log directory: %v", err)
	}
	logFile := filepath.Join(logDir, "stopped-server.log")
	if err := os.WriteFile(logFile, []byte("stale output"), 0644); err != nil {
		t.Fatalf("write stale log: %v", err)
	}
	old := time.Now().Add(-8 * 24 * time.Hour)
	if err := os.Chtimes(logFile, old, old); err != nil {
		t.Fatalf("age stale log: %v", err)
	}

	configPath := filepath.Join(tempDir, "config.yaml")
	configContents := "log_dir: " + logDir + "\nlog_retention: 7d\n"
	if err := os.WriteFile(configPath, []byte(configContents), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	previousCfgFile := cfgFile
	previousCfg := cfg
	previousCfgErr := cfgErr
	t.Cleanup(func() {
		cfgFile = previousCfgFile
		cfg = previousCfg
		cfgErr = previousCfgErr
		rootCmd.SetArgs(nil)
	})

	rootCmd.SetArgs([]string{"--config", configPath, "version"})
	if err := Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if _, err := os.Stat(logFile); !os.IsNotExist(err) {
		t.Fatalf("expired stopped log still exists; stat error = %v", err)
	}
}
