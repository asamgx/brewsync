package brewfile

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Metadata represents the .brewsync-meta file
type Metadata struct {
	Machine         string         `yaml:"machine"`
	LastDump        time.Time      `yaml:"last_dump"`
	LastSync        LastSyncInfo   `yaml:"last_sync,omitempty"`
	PackageCounts   map[string]int `yaml:"package_counts,omitempty"`
	MacOSVersion    string         `yaml:"macos_version,omitempty"`
	BrewsyncVersion string         `yaml:"brewsync_version,omitempty"`
}

// LastSyncInfo contains information about the last sync operation
type LastSyncInfo struct {
	From    string    `yaml:"from"`
	At      time.Time `yaml:"at"`
	Added   int       `yaml:"added"`
	Removed int       `yaml:"removed"`
}

// LoadMetadata loads metadata from the given path
func LoadMetadata(path string) (*Metadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var meta Metadata
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return nil, err
	}

	return &meta, nil
}

// SaveMetadata saves metadata to the given path
func SaveMetadata(path string, meta *Metadata) error {
	data, err := yaml.Marshal(meta)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// UpdateMetadata updates the metadata file with new dump information
func UpdateMetadata(path string, machine string, packages Packages, version string) error {
	meta := &Metadata{
		Machine:         machine,
		LastDump:        time.Now(),
		PackageCounts:   make(map[string]int),
		BrewsyncVersion: version,
	}

	// Try to load existing metadata to preserve last_sync info
	existing, err := LoadMetadata(path)
	if err == nil && existing != nil {
		meta.LastSync = existing.LastSync
		meta.MacOSVersion = existing.MacOSVersion
	}

	// Count packages by type
	for _, pkg := range packages {
		meta.PackageCounts[string(pkg.Type)]++
	}

	return SaveMetadata(path, meta)
}

// UpdateSyncMetadata updates the metadata file with sync information
func UpdateSyncMetadata(path string, fromMachine string, added, removed int) error {
	meta, err := LoadMetadata(path)
	if err != nil {
		// Create new metadata if file doesn't exist
		meta = &Metadata{}
	}

	meta.LastSync = LastSyncInfo{
		From:    fromMachine,
		At:      time.Now(),
		Added:   added,
		Removed: removed,
	}

	return SaveMetadata(path, meta)
}
