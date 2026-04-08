package rfxbands

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// LoadFile loads BandSets from a JSON file. Returns an empty slice (not an
// error) if the file does not exist.
func LoadFile(path string) ([]*BandSet, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading band sets file: %w", err)
	}
	var sets []*BandSet
	if err := json.Unmarshal(data, &sets); err != nil {
		return nil, fmt.Errorf("parsing band sets file: %w", err)
	}
	return sets, nil
}

// SaveFile writes sets to path as JSON, creating parent directories as needed.
func SaveFile(path string, sets []*BandSet) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}
	data, err := json.MarshalIndent(sets, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling band sets: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing band sets file: %w", err)
	}
	return nil
}

// MergeSets returns a combined slice of builtins and custom sets. Custom sets
// with the same name as a builtin replace the builtin.
func MergeSets(builtins, custom []*BandSet) []*BandSet {
	customByName := make(map[string]bool, len(custom))
	for _, bs := range custom {
		customByName[bs.Name] = true
	}
	result := make([]*BandSet, 0, len(builtins)+len(custom))
	for _, bs := range builtins {
		if !customByName[bs.Name] {
			result = append(result, bs)
		}
	}
	return append(result, custom...)
}
