package config

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// Config contains validated settings consumed by the application layer.
type Config struct {
	Address           string
	ReadHeaderTimeout time.Duration
	ShutdownTimeout   time.Duration
}

// Load reads .env from the working directory. Process environment takes precedence.
// A missing .env is allowed so deployed applications can use environment only.
func Load() (Config, error) {
	values := map[string]string{}
	file, err := os.Open(".env")
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, fmt.Errorf("open .env: %w", err)
	}
	if err == nil {
		defer file.Close()
		values, err = readDotEnv(file)
		if err != nil {
			return Config{}, err
		}
	}
	// Read settings without modifying the process environment. Explicitly set
	// environment variables win even when empty, so invalid overrides fail fast.
	return FromEnv(func(key string) (string, bool) {
		if value, ok := os.LookupEnv(key); ok {
			return value, true
		}
		value, ok := values[key]
		return value, ok
	})
}

// readDotEnv supports KEY=value, optional export prefixes, and quoted values.
// Shell expansion and inline comments are intentionally unsupported.
func readDotEnv(reader io.Reader) (map[string]string, error) {
	values := make(map[string]string)
	scanner := bufio.NewScanner(reader)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, value, ok := strings.Cut(line, "=")
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		if !ok || key == "" {
			return nil, fmt.Errorf(".env line %d: expected KEY=value", lineNumber)
		}
		for i, char := range key {
			if char != '_' && !(char >= 'A' && char <= 'Z') && !(char >= 'a' && char <= 'z') && !(i > 0 && char >= '0' && char <= '9') {
				return nil, fmt.Errorf(".env line %d: invalid variable name", lineNumber)
			}
		}
		if strings.HasPrefix(value, "\"") || strings.HasPrefix(value, "'") {
			if len(value) < 2 || value[len(value)-1] != value[0] {
				return nil, fmt.Errorf(".env line %d: unterminated quoted value", lineNumber)
			}
			value = value[1 : len(value)-1]
		}
		values[key] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read .env: %w", err)
	}
	return values, nil
}

// FromEnv allows configuration to be validated without changing process environment.
func FromEnv(lookup func(string) (string, bool)) (Config, error) {
	cfg := Config{
		Address:           ":8080",
		ReadHeaderTimeout: 5 * time.Second,
		ShutdownTimeout:   10 * time.Second,
	}
	if value, ok := lookup("HTTP_ADDR"); ok {
		cfg.Address = strings.TrimSpace(value)
		if cfg.Address == "" {
			return Config{}, fmt.Errorf("HTTP_ADDR must not be empty")
		}
	}
	// Use a fixed order so validation errors are predictable when both are invalid.
	timeouts := []struct {
		key    string
		target *time.Duration
	}{
		{"HTTP_READ_HEADER_TIMEOUT", &cfg.ReadHeaderTimeout},
		{"HTTP_SHUTDOWN_TIMEOUT", &cfg.ShutdownTimeout},
	}
	for _, timeout := range timeouts {
		key := timeout.key
		if value, ok := lookup(key); ok {
			duration, err := time.ParseDuration(value)
			if err != nil || duration <= 0 {
				return Config{}, fmt.Errorf("%s must be a positive duration", key)
			}
			*timeout.target = duration
		}
	}
	return cfg, nil
}
