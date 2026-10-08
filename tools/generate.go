//go:build ignore

// This generator runner pins tools and their Go toolchain independently of the app.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: go run tools/generate.go moq|swag [arguments]")
		os.Exit(1)
	}
	// Pin versions so students and CI generate identical files.
	versions := map[string]string{
		"moq":  "github.com/matryer/moq@v0.4.0",
		"swag": "github.com/swaggo/swag/cmd/swag@v1.16.6",
	}
	version, ok := versions[os.Args[1]]
	if !ok {
		fmt.Fprintln(os.Stderr, "unknown generator:", os.Args[1])
		os.Exit(1)
	}
	args := append([]string{"run", "-mod=mod", version}, os.Args[2:]...)
	cmd := exec.Command("go", args...)
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "GOTOOLCHAIN=") {
			cmd.Env = append(cmd.Env, entry)
		}
	}
	// Generators can require a newer toolchain than the application itself.
	cmd.Env = append(cmd.Env, "GOTOOLCHAIN=go1.23.12")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
