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

// configSaver writes the UI config on its own goroutine so saves never block
// the UI. Saves requested while one is in progress are coalesced: only the
// latest is written.
type configSaver struct {
	path    string
	onError func(error)
	pending chan []byte
	done    chan struct{}
}

// newConfigSaver starts a saver for path. onError is called from the
// saver's goroutine.
func newConfigSaver(path string, onError func(error)) *configSaver {
	s := &configSaver{
		path:    path,
		onError: onError,
		pending: make(chan []byte, 1),
		done:    make(chan struct{}),
	}
	go s.run()
	return s
}

func (s *configSaver) run() {
	defer close(s.done)
	for data := range s.pending {
		if err := writeFileAtomic(s.path, data); err != nil {
			s.onError(err)
		}
	}
}

// Save queues cfg to be written. It marshals cfg before returning, so the
// caller may keep modifying it. Only one goroutine may call Save.
func (s *configSaver) Save(cfg *UIConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling UI config: %w", err)
	}
	// Replace a save that has not started yet.
	select {
	case <-s.pending:
	default:
	}
	s.pending <- data
	return nil
}

// Close writes any queued save and stops the saver.
func (s *configSaver) Close() {
	close(s.pending)
	<-s.done
}

// writeFileAtomic replaces path with data by writing a temporary file in the
// same directory and renaming it, so a crash never leaves a partial file.
func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	f, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("writing UI config: %w", err)
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()           // the write error is the one to report
		_ = os.Remove(f.Name()) // best effort cleanup of the partial file
		return fmt.Errorf("writing UI config: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(f.Name()) // best effort cleanup of the partial file
		return fmt.Errorf("writing UI config: %w", err)
	}
	if err := os.Rename(f.Name(), path); err != nil {
		_ = os.Remove(f.Name()) // best effort cleanup
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
