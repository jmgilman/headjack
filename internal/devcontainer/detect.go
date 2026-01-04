package devcontainer

import (
	"os"
	"path/filepath"
)

// ConfigPaths lists the standard locations for devcontainer.json files,
// in order of precedence per the Dev Container specification.
var ConfigPaths = []string{
	".devcontainer/devcontainer.json",
	".devcontainer.json",
}

// Detect checks if a devcontainer.json exists in the given workspace folder.
// Returns the path to the devcontainer.json if found, or empty string if not found.
// Checks standard locations per the Dev Container specification.
func Detect(workspaceFolder string) string {
	for _, relPath := range ConfigPaths {
		fullPath := filepath.Join(workspaceFolder, relPath)
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath
		}
	}
	return ""
}

// HasConfig returns true if the workspace folder contains a devcontainer.json.
func HasConfig(workspaceFolder string) bool {
	return Detect(workspaceFolder) != ""
}
