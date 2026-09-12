package project

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const envFilename = ".env"

// LoadEnv sets every KEY=VALUE pair in root/.env that is not already
// present in the environment, so a subcommand (and the runner or
// server it starts) sees the project's local configuration without
// the caller exporting anything. Variables already exported win, which
// is what lets a Makefile target override a single value
// (DATABASE_URL=… gonext migrate). A missing .env is not an error.
//
// The format is the subset .env.example uses: one KEY=VALUE per line,
// blank lines and # comments ignored, an optional export prefix, and
// optional matching quotes around the value.
func LoadEnv(root string) error {
	f, err := os.Open(filepath.Join(root, envFilename))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("opening %s: %w", envFilename, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		key, value, ok := parseEnvLine(scanner.Text())
		if !ok {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("setting %s from %s: %w", key, envFilename, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading %s: %w", envFilename, err)
	}
	return nil
}

// parseEnvLine returns the key and value on line, or ok=false for a
// blank line, a comment, or a line with no "=".
func parseEnvLine(line string) (key, value string, ok bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return "", "", false
	}
	line = strings.TrimPrefix(line, "export ")
	key, value, found := strings.Cut(line, "=")
	if !found {
		return "", "", false
	}
	key = strings.TrimSpace(key)
	value = strings.TrimSpace(value)
	if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') && value[len(value)-1] == value[0] {
		value = value[1 : len(value)-1]
	}
	return key, value, key != ""
}
