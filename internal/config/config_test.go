package config

import (
	"os"
	"path/filepath"
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

func TestReadDotEnv(t *testing.T) {
	values, err := readDotEnv(strings.NewReader("# settings\n\nexport HTTP_ADDR = ':9090'\nHTTP_READ_HEADER_TIMEOUT=\"2s\"\n"))
	if err != nil || values["HTTP_ADDR"] != ":9090" || values["HTTP_READ_HEADER_TIMEOUT"] != "2s" {
		t.Fatalf("values=%v err=%v", values, err)
	}
	for _, input := range []string{"missing equals", "=value", "1KEY=value", "HTTP_ADDR='unfinished"} {
		if _, err := readDotEnv(strings.NewReader(input)); err == nil {
			t.Fatalf("expected error for %q", input)
		}
	}
}

func TestLoadDotEnv(t *testing.T) {
	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(original); err != nil {
			t.Error(err)
		}
	})
	for _, key := range []string{"HTTP_ADDR", "HTTP_READ_HEADER_TIMEOUT", "HTTP_SHUTDOWN_TIMEOUT"} {
		t.Setenv(key, "")
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
	}
	cfg, err := Load()
	if err != nil || cfg.Address != ":8080" {
		t.Fatalf("missing file: config=%+v err=%v", cfg, err)
	}
	file := filepath.Join(dir, ".env")
	if err := os.WriteFile(file, []byte("HTTP_ADDR=:9090\nHTTP_READ_HEADER_TIMEOUT=2s\nHTTP_SHUTDOWN_TIMEOUT=15s\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err = Load()
	if err != nil || cfg.Address != ":9090" || cfg.ReadHeaderTimeout != 2*time.Second || cfg.ShutdownTimeout != 15*time.Second {
		t.Fatalf("file settings: config=%+v err=%v", cfg, err)
	}
	t.Setenv("HTTP_ADDR", ":9191")
	cfg, err = Load()
	if err != nil || cfg.Address != ":9191" {
		t.Fatalf("environment override: config=%+v err=%v", cfg, err)
	}
	if err := os.WriteFile(file, []byte("HTTP_SHUTDOWN_TIMEOUT=bad\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(); err == nil {
		t.Fatal("expected validation error")
	}
	if err := os.WriteFile(file, []byte("invalid line\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(); err == nil {
		t.Fatal("expected parse error")
	}
}
