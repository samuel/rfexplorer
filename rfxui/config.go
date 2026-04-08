package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// PresetOverlay associates a preset (by device index) with a named BandSet.
type PresetOverlay struct {
	PresetIndex int    `json:"preset_index"`
	BandSetName string `json:"band_set"`
}

// UIConfig holds local UI state that is not stored on the RF Explorer device.
type UIConfig struct {
	PresetOverlays []PresetOverlay `json:"preset_overlays"`
}

// BandSetFor returns the BandSet name associated with the given preset index,
// or "" if none is set.
func (c *UIConfig) BandSetFor(presetIndex int) string {
	for _, o := range c.PresetOverlays {
		if o.PresetIndex == presetIndex {
			return o.BandSetName
		}
	}
	return ""
}

// SetBandSetFor sets or clears the BandSet association for a preset index.
// Passing name="" removes the association.
func (c *UIConfig) SetBandSetFor(presetIndex int, name string) {
	for i, o := range c.PresetOverlays {
		if o.PresetIndex == presetIndex {
			if name == "" {
				c.PresetOverlays = append(c.PresetOverlays[:i], c.PresetOverlays[i+1:]...)
			} else {
				c.PresetOverlays[i].BandSetName = name
			}
			return
		}
	}
	if name != "" {
		c.PresetOverlays = append(c.PresetOverlays, PresetOverlay{PresetIndex: presetIndex, BandSetName: name})
	}
}

// Save writes the config to path, creating parent directories as needed.
func (c *UIConfig) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling UI config: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing UI config: %w", err)
	}
	return nil
}

// loadUIConfig reads the config from path. Returns an empty config (not an
// error) if the file does not exist.
func loadUIConfig(path string) (*UIConfig, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &UIConfig{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading UI config: %w", err)
	}
	var cfg UIConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing UI config: %w", err)
	}
	return &cfg, nil
}

// configDir returns the OS-appropriate application config directory.
func configDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("getting user config dir: %w", err)
	}
	return filepath.Join(dir, "rfexplorer"), nil
}
