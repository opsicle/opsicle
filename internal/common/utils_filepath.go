package common

import (
	"fmt"
	"os/user"
	"path/filepath"
	"strings"
)

// ToAbsolutePath converts a relative file path into an absolute one,
// and expands '~' to the current user's home directory.
func ToAbsolutePath(path string) (string, error) {
	if strings.HasPrefix(path, "~") {
		usr, err := user.Current()
		if err != nil {
			return "", fmt.Errorf("failed to get current user: %w", err)
		}
		homeDir := usr.HomeDir
		if path == "~" {
			path = homeDir
		} else if strings.HasPrefix(path, "~/") {
			path = filepath.Join(homeDir, path[2:])
		}
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to convert path to absolute: %w", err)
	}

	return absPath, nil
}
