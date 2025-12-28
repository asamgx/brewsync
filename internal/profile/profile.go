package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/asamgx/brewsync/internal/brewfile"
	"github.com/asamgx/brewsync/internal/config"
)

// Profile represents a curated package group
type Profile struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description,omitempty"`
	Packages    Packages `yaml:"packages"`
}

// Packages holds packages grouped by type
type Packages struct {
	Tap    []string `yaml:"tap,omitempty"`
	Brew   []string `yaml:"brew,omitempty"`
	Cask   []string `yaml:"cask,omitempty"`
	VSCode []string `yaml:"vscode,omitempty"`
	Cursor []string `yaml:"cursor,omitempty"`
	Go     []string `yaml:"go,omitempty"`
	Mas    []string `yaml:"mas,omitempty"`
}

// ToBrewfilePackages converts profile packages to brewfile.Packages
func (p *Packages) ToBrewfilePackages() brewfile.Packages {
	var result brewfile.Packages

	for _, name := range p.Tap {
		result = append(result, brewfile.NewPackage(brewfile.TypeTap, name))
	}
	for _, name := range p.Brew {
		result = append(result, brewfile.NewPackage(brewfile.TypeBrew, name))
	}
	for _, name := range p.Cask {
		result = append(result, brewfile.NewPackage(brewfile.TypeCask, name))
	}
	for _, name := range p.VSCode {
		result = append(result, brewfile.NewPackage(brewfile.TypeVSCode, name))
	}
	for _, name := range p.Cursor {
		result = append(result, brewfile.NewPackage(brewfile.TypeCursor, name))
	}
	for _, name := range p.Go {
		result = append(result, brewfile.NewPackage(brewfile.TypeGo, name))
	}
	for _, name := range p.Mas {
		result = append(result, brewfile.NewPackage(brewfile.TypeMas, name))
	}

	return result
}

// Count returns the total number of packages
func (p *Packages) Count() int {
	return len(p.Tap) + len(p.Brew) + len(p.Cask) +
		len(p.VSCode) + len(p.Cursor) + len(p.Go) + len(p.Mas)
}

// Load loads a profile by name
func Load(name string) (*Profile, error) {
	profilesDir, err := config.ProfilesDir()
	if err != nil {
		return nil, err
	}

	// Try with and without .yaml extension
	path := filepath.Join(profilesDir, name)
	if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
		if _, err := os.Stat(path + ".yaml"); err == nil {
			path = path + ".yaml"
		} else if _, err := os.Stat(path + ".yml"); err == nil {
			path = path + ".yml"
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read profile: %w", err)
	}

	var profile Profile
	if err := yaml.Unmarshal(data, &profile); err != nil {
		return nil, fmt.Errorf("failed to parse profile: %w", err)
	}

	// Use filename as name if not set
	if profile.Name == "" {
		profile.Name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}

	return &profile, nil
}

// LoadMultiple loads multiple profiles by name
func LoadMultiple(names []string) ([]*Profile, error) {
	var profiles []*Profile
	for _, name := range names {
		p, err := Load(name)
		if err != nil {
			return nil, fmt.Errorf("failed to load profile '%s': %w", name, err)
		}
		profiles = append(profiles, p)
	}
	return profiles, nil
}

// List returns all available profile names
func List() ([]string, error) {
	profilesDir, err := config.ProfilesDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(profilesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
			names = append(names, strings.TrimSuffix(strings.TrimSuffix(name, ".yaml"), ".yml"))
		}
	}

	return names, nil
}

// Save saves a profile to disk
func Save(profile *Profile) error {
	profilesDir, err := config.ProfilesDir()
	if err != nil {
		return err
	}

	// Ensure directory exists
	if err := os.MkdirAll(profilesDir, 0755); err != nil {
		return fmt.Errorf("failed to create profiles directory: %w", err)
	}

	path := filepath.Join(profilesDir, profile.Name+".yaml")

	data, err := yaml.Marshal(profile)
	if err != nil {
		return fmt.Errorf("failed to marshal profile: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write profile: %w", err)
	}

	return nil
}

// Delete removes a profile
func Delete(name string) error {
	profilesDir, err := config.ProfilesDir()
	if err != nil {
		return err
	}

	path := filepath.Join(profilesDir, name+".yaml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		path = filepath.Join(profilesDir, name+".yml")
	}

	return os.Remove(path)
}

// Exists checks if a profile exists
func Exists(name string) bool {
	profilesDir, err := config.ProfilesDir()
	if err != nil {
		return false
	}

	path := filepath.Join(profilesDir, name+".yaml")
	if _, err := os.Stat(path); err == nil {
		return true
	}

	path = filepath.Join(profilesDir, name+".yml")
	if _, err := os.Stat(path); err == nil {
		return true
	}

	return false
}

// GetPath returns the path to a profile file
func GetPath(name string) (string, error) {
	profilesDir, err := config.ProfilesDir()
	if err != nil {
		return "", err
	}

	path := filepath.Join(profilesDir, name+".yaml")
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}

	path = filepath.Join(profilesDir, name+".yml")
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}

	// Return the default path even if it doesn't exist
	return filepath.Join(profilesDir, name+".yaml"), nil
}

// MergePackages merges packages from multiple profiles
func MergePackages(profiles []*Profile) brewfile.Packages {
	seen := make(map[string]bool)
	var result brewfile.Packages

	for _, p := range profiles {
		for _, pkg := range p.Packages.ToBrewfilePackages() {
			key := pkg.ID()
			if !seen[key] {
				seen[key] = true
				result = append(result, pkg)
			}
		}
	}

	return result
}

// FromBrewfilePackages creates a Packages struct from brewfile.Packages
func FromBrewfilePackages(packages brewfile.Packages) Packages {
	var p Packages

	for _, pkg := range packages {
		switch pkg.Type {
		case brewfile.TypeTap:
			p.Tap = append(p.Tap, pkg.Name)
		case brewfile.TypeBrew:
			p.Brew = append(p.Brew, pkg.Name)
		case brewfile.TypeCask:
			p.Cask = append(p.Cask, pkg.Name)
		case brewfile.TypeVSCode:
			p.VSCode = append(p.VSCode, pkg.Name)
		case brewfile.TypeCursor:
			p.Cursor = append(p.Cursor, pkg.Name)
		case brewfile.TypeGo:
			p.Go = append(p.Go, pkg.Name)
		case brewfile.TypeMas:
			p.Mas = append(p.Mas, pkg.Name)
		}
	}

	return p
}
