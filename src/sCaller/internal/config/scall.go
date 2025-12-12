package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

// ScallConfig represents the flat scall.toml configuration
// Keys are dotted paths like "config.p2p.seeds" or "app.pruning"
type ScallConfig map[string]interface{}

// LoadScall loads scall.toml from path
func LoadScall(path string) (ScallConfig, error) {
	var cfg ScallConfig
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, fmt.Errorf("failed to load scall.toml: %w", err)
	}
	return cfg, nil
}

// ApplyToConfigToml applies config.* entries to config.toml
func (s ScallConfig) ApplyToConfigToml(configPath string) error {
	return s.applyToFile(configPath, "config.")
}

// ApplyToAppToml applies app.* entries to app.toml
func (s ScallConfig) ApplyToAppToml(appPath string) error {
	return s.applyToFile(appPath, "app.")
}

func (s ScallConfig) applyToFile(filePath, prefix string) error {
	// Load existing TOML as raw map
	var existing map[string]interface{}
	if _, err := toml.DecodeFile(filePath, &existing); err != nil {
		return fmt.Errorf("failed to read %s: %w", filePath, err)
	}

	// Apply overrides
	for key, value := range s {
		if !strings.HasPrefix(key, prefix) {
			continue
		}

		// Remove prefix: "config.p2p.seeds" -> "p2p.seeds"
		path := strings.TrimPrefix(key, prefix)
		parts := strings.Split(path, ".")

		if err := setNestedValue(existing, parts, value); err != nil {
			return fmt.Errorf("failed to set %s: %w", key, err)
		}
	}

	// Write back
	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", filePath, err)
	}
	defer f.Close()

	encoder := toml.NewEncoder(f)
	if err := encoder.Encode(existing); err != nil {
		return fmt.Errorf("failed to encode %s: %w", filePath, err)
	}

	return nil
}

// setNestedValue sets a value in a nested map using a path like ["p2p", "seeds"]
func setNestedValue(m map[string]interface{}, path []string, value interface{}) error {
	if len(path) == 0 {
		return fmt.Errorf("empty path")
	}

	if len(path) == 1 {
		m[path[0]] = value
		return nil
	}

	// Navigate/create nested maps
	key := path[0]
	rest := path[1:]

	nested, exists := m[key]
	if !exists {
		// Create new nested map
		nested = make(map[string]interface{})
		m[key] = nested
	}

	nestedMap, ok := nested.(map[string]interface{})
	if !ok {
		return fmt.Errorf("cannot set nested value: %s is not a map", key)
	}

	return setNestedValue(nestedMap, rest, value)
}

// SetValue sets a single value in the scall config
func (s ScallConfig) SetValue(key string, value interface{}) {
	s[key] = value
}
