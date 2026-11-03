//go:build !windows

package main

import (
	"os"
	"path/filepath"
)

func defaultDataDirectory() (string, error) {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "Stockbook"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "Stockbook"), nil
}
