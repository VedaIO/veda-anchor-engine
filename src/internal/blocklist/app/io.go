package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"veda-anchor-engine/src/internal/config"
)

// LoadAppBlocklist reads the blocklist file from the user's cache directory.
// It returns a slice of strings, with all entries normalized to lowercase for case-insensitive matching.
// If the file doesn't exist, it returns an empty list, which is not considered an error.
func LoadAppBlocklist() ([]string, error) {
	p, err := config.GetAppBlocklistPath()
	if err != nil {
		return nil, err
	}

	b, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return nil, nil // File not existing is not an error, just an empty list.
	}
	if err != nil {
		return nil, err
	}

	var list []string
	if err := json.Unmarshal(b, &list); err != nil {
		return nil, fmt.Errorf("failed to unmarshal blocklist: %w", err)
	}

	// Normalize all entries to lowercase to ensure case-insensitive comparison.
	for i := range list {
		list[i] = strings.ToLower(list[i])
	}
	return list, nil
}

// SaveAppBlocklist writes the given list of strings to the blocklist file.
// It normalizes all entries to lowercase before saving to ensure consistency.
// It also sets appropriate file permissions to secure the file.
func SaveAppBlocklist(list []string) error {
	// Normalize all entries to lowercase to ensure consistency.
	for i := range list {
		list[i] = strings.ToLower(list[i])
	}

	p, err := config.GetAppBlocklistPath()
	if err != nil {
		return err
	}
	_ = os.MkdirAll(filepath.Dir(p), 0755)

	b, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal blocklist: %w", err)
	}
	return os.WriteFile(p, b, 0644) // readable by all, writable by SYSTEM
}
