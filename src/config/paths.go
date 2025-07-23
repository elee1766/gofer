package config

import (
	"path/filepath"

	"github.com/adrg/xdg"
)

// StoragePaths contains paths for application storage
type StoragePaths struct {
	DatabasePath string
	ContentPath  string
}

// GetDefaultStoragePaths returns default storage paths using XDG base directories
func GetDefaultStoragePaths() StoragePaths {
	// Use XDG_STATE_HOME for runtime state data
	// This follows XDG Base Directory specification for state data
	dbPath := filepath.Join(xdg.StateHome, "gofer", "conversations.db")
	contentPath := filepath.Join(xdg.StateHome, "gofer", "conversations")
	
	return StoragePaths{
		DatabasePath: dbPath,
		ContentPath:  contentPath,
	}
}

// GetDefaultCachePath returns the default cache directory path
func GetDefaultCachePath() string {
	// Use XDG_CACHE_HOME for cached data
	return filepath.Join(xdg.CacheHome, "gofer")
}

// GetDefaultDataPath returns the default data directory path
func GetDefaultDataPath() string {
	// Use XDG_DATA_HOME for user-specific data files
	return filepath.Join(xdg.DataHome, "gofer")
}