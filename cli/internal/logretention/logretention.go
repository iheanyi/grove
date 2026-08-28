// Package logretention removes expired Grove log files.
package logretention

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Result describes log files removed by Cleanup.
type Result struct {
	Removed []string
}

// Cleanup removes regular .log files older than retention from logDir.
func Cleanup(logDir string, retention time.Duration, activeLogFiles []string, now time.Time) (Result, error) {
	if retention <= 0 {
		return Result{}, fmt.Errorf("log retention must be positive, got %s", retention)
	}
	return cleanup(logDir, retention, activeLogFiles, now, os.Remove)
}

func cleanup(logDir string, retention time.Duration, activeLogFiles []string, now time.Time, remove func(string) error) (Result, error) {
	active := make(map[string]struct{}, len(activeLogFiles))
	for _, path := range activeLogFiles {
		absolute, err := filepath.Abs(path)
		if err != nil {
			return Result{}, err
		}
		active[filepath.Clean(absolute)] = struct{}{}
	}

	entries, err := os.ReadDir(logDir)
	if err != nil {
		if os.IsNotExist(err) {
			return Result{}, nil
		}
		return Result{}, err
	}

	result := Result{}
	cutoff := now.Add(-retention)
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".log") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return result, err
		}
		if !info.Mode().IsRegular() || !info.ModTime().Before(cutoff) {
			continue
		}

		path := filepath.Join(logDir, entry.Name())
		absolute, err := filepath.Abs(path)
		if err != nil {
			return result, err
		}
		if _, isActive := active[filepath.Clean(absolute)]; isActive {
			continue
		}
		if err := remove(path); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return result, err
		}
		result.Removed = append(result.Removed, path)
	}
	return result, nil
}
