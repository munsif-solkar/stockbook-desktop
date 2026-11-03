//go:build windows

package main

import (
	"os"
	"path/filepath"
)

// defaultDataDirectory keeps the database inside the user's local app data
// folder on Windows instead of a Unix-style ~/.local/share path.
func defaultDataDirectory() (string, error) {
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		return filepath.Join(local, "Stockbook"), nil
	}
	if config, err := os.UserConfigDir(); err == nil && config != "" {
		return filepath.Join(config, "Stockbook"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Stockbook"), nil
}
