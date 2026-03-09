package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	thingslib "github.com/alnah/things-agent/internal/things"
)

func resolveDataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	pattern := filepath.Join(home, thingsDataPattern)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", fmt.Errorf("failed to resolve Things data dir: %w", err)
	}
	sort.Strings(matches)
	for _, candidate := range matches {
		if st, err := os.Stat(filepath.Join(candidate, "main.sqlite")); err == nil && !st.IsDir() {
			return candidate, nil
		}
	}
	return "", errors.New("could not resolve Things data dir automatically; set THINGS_DATA_DIR")
}

func envOrDefault(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok && strings.TrimSpace(value) != "" {
		return value
	}
	return defaultValue
}

func escapeApple(value string) string {
	return thingslib.EscapeApple(value)
}
