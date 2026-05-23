package config

import (
	"os"
	"path/filepath"
)

// Dir returns ~/.cx directory path
func Dir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".cx"
	}
	return filepath.Join(home, ".cx")
}

// Ensure creates the config dir if needed
func Ensure() error {
	return os.MkdirAll(Dir(), 0755)
}
