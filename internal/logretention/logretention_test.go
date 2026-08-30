package logretention

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCleanupDeletesExpiredLogFile(t *testing.T) {
	logDir := t.TempDir()
	logFile := filepath.Join(logDir, "stopped-server.log")
	if err := os.WriteFile(logFile, []byte("stale output"), 0644); err != nil {
		t.Fatalf("write stale log: %v", err)
	}

	now := time.Date(2026, time.August, 28, 12, 0, 0, 0, time.UTC)
	if err := os.Chtimes(logFile, now.Add(-8*24*time.Hour), now.Add(-8*24*time.Hour)); err != nil {
		t.Fatalf("age stale log: %v", err)
	}

	result, err := Cleanup(logDir, 7*24*time.Hour, nil, now)
	if err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}
	if _, err := os.Stat(logFile); !os.IsNotExist(err) {
		t.Fatalf("expired log still exists; stat error = %v", err)
	}
	if len(result.Removed) != 1 || result.Removed[0] != logFile {
		t.Fatalf("Cleanup() removed = %v, want [%s]", result.Removed, logFile)
	}
}

func TestCleanupPreservesCurrentLogFiles(t *testing.T) {
	logDir := t.TempDir()
	currentLog := filepath.Join(logDir, "current-server.log")
	if err := os.WriteFile(currentLog, []byte("current output"), 0644); err != nil {
		t.Fatalf("write current log: %v", err)
	}

	now := time.Date(2026, time.August, 28, 12, 0, 0, 0, time.UTC)
	if err := os.Chtimes(currentLog, now.Add(-6*24*time.Hour), now.Add(-6*24*time.Hour)); err != nil {
		t.Fatalf("date current log: %v", err)
	}

	result, err := Cleanup(logDir, 7*24*time.Hour, nil, now)
	if err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}
	if _, err := os.Stat(currentLog); err != nil {
		t.Fatalf("current log was removed: %v", err)
	}
	if len(result.Removed) != 0 {
		t.Fatalf("Cleanup() removed = %v, want none", result.Removed)
	}
}

func TestCleanupPreservesExpiredLogForRunningServer(t *testing.T) {
	logDir := t.TempDir()
	activeLog := filepath.Join(logDir, "running-server.log")
	if err := os.WriteFile(activeLog, []byte("still streaming"), 0644); err != nil {
		t.Fatalf("write active log: %v", err)
	}

	now := time.Date(2026, time.August, 28, 12, 0, 0, 0, time.UTC)
	if err := os.Chtimes(activeLog, now.Add(-30*24*time.Hour), now.Add(-30*24*time.Hour)); err != nil {
		t.Fatalf("age active log: %v", err)
	}

	result, err := Cleanup(logDir, 7*24*time.Hour, []string{activeLog}, now)
	if err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}
	if _, err := os.Stat(activeLog); err != nil {
		t.Fatalf("running server log was removed: %v", err)
	}
	if len(result.Removed) != 0 {
		t.Fatalf("Cleanup() removed = %v, want none", result.Removed)
	}
}

func TestCleanupToleratesMissingLogDirectory(t *testing.T) {
	missingDir := filepath.Join(t.TempDir(), "not-created")

	result, err := Cleanup(missingDir, 7*24*time.Hour, nil, time.Now())
	if err != nil {
		t.Fatalf("Cleanup() missing directory error = %v, want nil", err)
	}
	if len(result.Removed) != 0 {
		t.Fatalf("Cleanup() removed = %v, want none", result.Removed)
	}
}

func TestCleanupToleratesLogRemovedDuringDeletion(t *testing.T) {
	logDir := t.TempDir()
	logFile := filepath.Join(logDir, "raced-server.log")
	if err := os.WriteFile(logFile, []byte("stale output"), 0644); err != nil {
		t.Fatalf("write stale log: %v", err)
	}

	now := time.Date(2026, time.August, 28, 12, 0, 0, 0, time.UTC)
	if err := os.Chtimes(logFile, now.Add(-8*24*time.Hour), now.Add(-8*24*time.Hour)); err != nil {
		t.Fatalf("age stale log: %v", err)
	}

	removeAfterAnotherProcess := func(path string) error {
		if err := os.Remove(path); err != nil {
			return err
		}
		return &os.PathError{Op: "remove", Path: path, Err: os.ErrNotExist}
	}
	result, err := cleanup(logDir, 7*24*time.Hour, nil, now, removeAfterAnotherProcess)
	if err != nil {
		t.Fatalf("cleanup() raced deletion error = %v, want nil", err)
	}
	if len(result.Removed) != 0 {
		t.Fatalf("cleanup() removed = %v, want none recorded for raced file", result.Removed)
	}
}

func TestCleanupLeavesNonLogFilesAndLogNamedDirectoriesAlone(t *testing.T) {
	logDir := t.TempDir()
	notes := filepath.Join(logDir, "notes.txt")
	logNamedDir := filepath.Join(logDir, "archive.log")
	if err := os.WriteFile(notes, []byte("not a Grove log"), 0644); err != nil {
		t.Fatalf("write non-log file: %v", err)
	}
	if err := os.Mkdir(logNamedDir, 0755); err != nil {
		t.Fatalf("create log-named directory: %v", err)
	}

	now := time.Date(2026, time.August, 28, 12, 0, 0, 0, time.UTC)
	old := now.Add(-30 * 24 * time.Hour)
	if err := os.Chtimes(notes, old, old); err != nil {
		t.Fatalf("age non-log file: %v", err)
	}
	if err := os.Chtimes(logNamedDir, old, old); err != nil {
		t.Fatalf("age log-named directory: %v", err)
	}

	if _, err := Cleanup(logDir, 7*24*time.Hour, nil, now); err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}
	for _, path := range []string{notes, logNamedDir} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("non-log path %s was removed: %v", path, err)
		}
	}
}

func TestCleanupRejectsNonPositiveRetention(t *testing.T) {
	if _, err := Cleanup(t.TempDir(), 0, nil, time.Now()); err == nil {
		t.Fatal("Cleanup() error = nil, want non-positive retention error")
	}
}
