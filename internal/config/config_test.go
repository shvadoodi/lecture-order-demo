package config

import (
	"strings"
	"testing"
	"time"
)

func TestFromEnv(t *testing.T) {
	for _, tc := range []struct {
		name     string
		env      map[string]string
		want     Config
		errorKey string
	}{
		{name: "defaults", want: Config{Address: ":8080", ReadHeaderTimeout: 5 * time.Second, ShutdownTimeout: 10 * time.Second}},
		{name: "overrides", env: map[string]string{"HTTP_ADDR": " 127.0.0.1:9090 ", "HTTP_READ_HEADER_TIMEOUT": "2s", "HTTP_SHUTDOWN_TIMEOUT": "1m"}, want: Config{Address: "127.0.0.1:9090", ReadHeaderTimeout: 2 * time.Second, ShutdownTimeout: time.Minute}},
		{name: "blank address", env: map[string]string{"HTTP_ADDR": " "}, errorKey: "HTTP_ADDR"},
		{name: "invalid duration", env: map[string]string{"HTTP_READ_HEADER_TIMEOUT": "bad"}, errorKey: "HTTP_READ_HEADER_TIMEOUT"},
		{name: "zero duration", env: map[string]string{"HTTP_READ_HEADER_TIMEOUT": "0s"}, errorKey: "HTTP_READ_HEADER_TIMEOUT"},
		{name: "negative duration", env: map[string]string{"HTTP_SHUTDOWN_TIMEOUT": "-1s"}, errorKey: "HTTP_SHUTDOWN_TIMEOUT"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := FromEnv(func(key string) (string, bool) { v, ok := tc.env[key]; return v, ok })
			if tc.errorKey != "" {
				if err == nil || !strings.Contains(err.Error(), tc.errorKey) {
					t.Fatalf("error=%v", err)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("config=%+v err=%v want=%+v", got, err, tc.want)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	t.Setenv("HTTP_ADDR", ":9191")
	t.Setenv("HTTP_READ_HEADER_TIMEOUT", "3s")
	t.Setenv("HTTP_SHUTDOWN_TIMEOUT", "7s")
	got, err := Load()
	if err != nil || got.Address != ":9191" || got.ReadHeaderTimeout != 3*time.Second || got.ShutdownTimeout != 7*time.Second {
		t.Fatalf("config=%+v err=%v", got, err)
	}
}
