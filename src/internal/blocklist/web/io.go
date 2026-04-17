package web

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"veda-anchor-engine/src/internal/config"
)

// LoadWebBlocklist reads the web blocklist file from the user's cache directory.
// It returns a slice of strings, with all entries normalized to lowercase for case-insensitive matching.
// If the file doesn't exist, it returns an empty list, which is not considered an error.
func LoadWebBlocklist() ([]string, error) {
	p, err := config.GetWebBlocklistPath()
	if err != nil {
		return nil, err
	}

	// If the blocklist file doesn't exist, return an empty list.
	b, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}

	var list []string
	if err := json.Unmarshal(b, &list); err != nil {
		return nil, fmt.Errorf("failed to unmarshal web blocklist: %w", err)
	}

	// Normalize all entries to lowercase for case-insensitive comparison.
	for i := range list {
		list[i] = strings.ToLower(list[i])
	}
	return list, nil
}

// SaveWebBlocklist writes the given list of strings to the web blocklist file.
// It normalizes all entries to lowercase before saving to ensure consistency.
func SaveWebBlocklist(list []string) error {
	// Normalize all entries to lowercase to ensure consistency.
	for i := range list {
		list[i] = strings.ToLower(list[i])
	}

	p, err := config.GetWebBlocklistPath()
	if err != nil {
		return err
	}
	_ = os.MkdirAll(filepath.Dir(p), 0755)

	// Marshal the list to JSON with indentation for readability.
	b, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal web blocklist: %w", err)
	}
	return os.WriteFile(p, b, 0644)
}
