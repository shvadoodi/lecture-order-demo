package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	Address           string
	ReadHeaderTimeout time.Duration
	ShutdownTimeout   time.Duration
}

func Load() (Config, error) { return FromEnv(os.LookupEnv) }

// FromEnv allows configuration to be validated without changing process environment.
func FromEnv(lookup func(string) (string, bool)) (Config, error) {
	cfg := Config{Address: ":8080", ReadHeaderTimeout: 5 * time.Second, ShutdownTimeout: 10 * time.Second}
	if value, ok := lookup("HTTP_ADDR"); ok {
		cfg.Address = strings.TrimSpace(value)
		if cfg.Address == "" {
			return Config{}, fmt.Errorf("HTTP_ADDR must not be empty")
		}
	}
	for key, target := range map[string]*time.Duration{"HTTP_READ_HEADER_TIMEOUT": &cfg.ReadHeaderTimeout, "HTTP_SHUTDOWN_TIMEOUT": &cfg.ShutdownTimeout} {
		if value, ok := lookup(key); ok {
			duration, err := time.ParseDuration(value)
			if err != nil || duration <= 0 {
				return Config{}, fmt.Errorf("%s must be a positive duration", key)
			}
			*target = duration
		}
	}
	return cfg, nil
}
