package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

const defaultLogMaxBytes int64 = 100 * 1024 * 1024

var logWriterCmd = &cobra.Command{
	Use:    "__log-writer <log-file> [max-size]",
	Hidden: true,
	Args:   cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		maxSize := ""
		if len(args) > 1 {
			maxSize = args[1]
		} else if cfg != nil {
			maxSize = cfg.LogMaxSize
		}

		maxBytes, err := parseLogSize(maxSize)
		if err != nil {
			return err
		}

		return runLogWriter(args[0], maxBytes, os.Stdin)
	},
}

func runLogWriter(logFile string, maxBytes int64, input io.Reader) error {
	if maxBytes <= 0 {
		maxBytes = defaultLogMaxBytes
	}

	if err := os.MkdirAll(filepath.Dir(logFile), 0755); err != nil {
		return err
	}

	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := trimOpenLogFile(file, maxBytes); err != nil {
		return err
	}

	trimThreshold := maxBytes * 2
	if trimThreshold < maxBytes {
		trimThreshold = maxBytes
	}

	buffer := make([]byte, 32*1024)
	for {
		n, readErr := input.Read(buffer)
		if n > 0 {
			if _, err := file.Write(buffer[:n]); err != nil {
				return err
			}
			if err := trimOpenLogFileAbove(file, maxBytes, trimThreshold); err != nil {
				return err
			}
		}
		if readErr == io.EOF {
			if err := trimOpenLogFile(file, maxBytes); err != nil {
				return err
			}
			return nil
		}
		if readErr != nil {
			return readErr
		}
	}
}

func prepareBoundedLogFile(logFile, maxSize string) error {
	maxBytes, err := parseLogSize(maxSize)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(logFile), 0755); err != nil {
		return err
	}

	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	return trimOpenLogFile(file, maxBytes)
}

func trimOpenLogFile(file *os.File, maxBytes int64) error {
	return trimOpenLogFileAbove(file, maxBytes, maxBytes)
}

func trimOpenLogFileAbove(file *os.File, maxBytes, thresholdBytes int64) error {
	if maxBytes <= 0 {
		maxBytes = defaultLogMaxBytes
	}
	if thresholdBytes < maxBytes {
		thresholdBytes = maxBytes
	}

	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.Size() <= thresholdBytes {
		return nil
	}

	retained := make([]byte, maxBytes)
	n, err := file.ReadAt(retained, info.Size()-maxBytes)
	if err != nil && err != io.EOF {
		return err
	}
	retained = retained[:n]

	if err := file.Truncate(0); err != nil {
		return err
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	}

	_, err = file.Write(retained)
	return err
}

func parseLogSize(value string) (int64, error) {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	if normalized == "" {
		return defaultLogMaxBytes, nil
	}

	multiplier := int64(1)
	for _, suffix := range []struct {
		text       string
		multiplier int64
	}{
		{"GB", 1024 * 1024 * 1024},
		{"G", 1024 * 1024 * 1024},
		{"MB", 1024 * 1024},
		{"M", 1024 * 1024},
		{"KB", 1024},
		{"K", 1024},
	} {
		if strings.HasSuffix(normalized, suffix.text) {
			multiplier = suffix.multiplier
			normalized = strings.TrimSpace(strings.TrimSuffix(normalized, suffix.text))
			break
		}
	}

	size, err := strconv.ParseInt(normalized, 10, 64)
	if err != nil || size < 0 {
		return 0, fmt.Errorf("invalid log_max_size %q", value)
	}

	return size * multiplier, nil
}

func logWriterShellCommand(logFile, maxSize string) (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}

	return shellQuoteArgs([]string{executable, "__log-writer", logFile, maxSize}), nil
}
