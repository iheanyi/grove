package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

const defaultLogMaxBytes int64 = 100 * 1024 * 1024

var logWriterMaxSize string

var logWriterCmd = &cobra.Command{
	Use:    "__log-writer <log-file>",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		maxSize := logWriterMaxSize
		if maxSize == "" && cfg != nil {
			maxSize = cfg.LogMaxSize
		}

		maxBytes, err := parseLogSize(maxSize)
		if err != nil {
			return err
		}

		return runLogWriter(args[0], maxBytes, os.Stdin)
	},
}

func init() {
	logWriterCmd.Flags().StringVar(&logWriterMaxSize, "max-size", "", "maximum bytes to retain per log file")
}

func runLogWriter(logFile string, maxBytes int64, input io.Reader) error {
	buffer := make([]byte, 32*1024)
	for {
		n, readErr := input.Read(buffer)
		if n > 0 {
			if err := appendBoundedLogChunk(logFile, buffer[:n], maxBytes); err != nil {
				return err
			}
		}
		if readErr == io.EOF {
			return nil
		}
		if readErr != nil {
			return readErr
		}
	}
}

func appendBoundedLogChunk(logFile string, chunk []byte, maxBytes int64) error {
	if maxBytes <= 0 {
		maxBytes = defaultLogMaxBytes
	}

	if err := os.MkdirAll(filepath.Dir(logFile), 0755); err != nil {
		return err
	}

	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	if _, err := file.Write(chunk); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}

	return enforceLogLimit(logFile, maxBytes)
}

func prepareBoundedLogFile(logFile, maxSize string) error {
	maxBytes, err := parseLogSize(maxSize)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(logFile), 0755); err != nil {
		return err
	}

	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}

	return enforceLogLimit(logFile, maxBytes)
}

func enforceLogLimit(logFile string, maxBytes int64) error {
	if maxBytes <= 0 {
		maxBytes = defaultLogMaxBytes
	}

	info, err := os.Stat(logFile)
	if err != nil {
		return err
	}
	if info.Size() <= maxBytes {
		return nil
	}

	retainBytes := maxBytes * 8 / 10
	if retainBytes < 1 {
		retainBytes = maxBytes
	}

	file, err := os.Open(logFile)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.Seek(info.Size()-retainBytes, io.SeekStart); err != nil {
		return err
	}

	retained, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	if newline := bytes.IndexByte(retained, '\n'); newline >= 0 && newline+1 < len(retained) {
		retained = retained[newline+1:]
	}

	return os.WriteFile(logFile, retained, 0644)
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

	return fmt.Sprintf("%s __log-writer --max-size %s %s",
		shellQuoteArgs([]string{executable}),
		shellQuoteArgs([]string{maxSize}),
		shellQuoteArgs([]string{logFile}),
	), nil
}
