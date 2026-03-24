package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	DefaultServerURL = "http://localhost:3000"
)

// Config holds the CLI configuration persisted to disk.
type Config struct {
	ServerURL string `yaml:"server_url"`
	Token     string `yaml:"token,omitempty"`
	TeamID    string `yaml:"team_id,omitempty"`
}

// DefaultPath returns the default config file path: ~/.guild/config.yaml
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	return filepath.Join(home, ".guild", "config.yaml"), nil
}

// Load reads config from the given path. If the file doesn't exist,
// returns a Config with default values (not an error).
func Load(path string) (*Config, error) {
	cfg := &Config{
		ServerURL: DefaultServerURL,
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}

	// Ensure default if file existed but server_url was empty
	if cfg.ServerURL == "" {
		cfg.ServerURL = DefaultServerURL
	}

	return cfg, nil
}

// Save writes the config to the given path, creating parent directories as needed.
func (c *Config) Save(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating config directory %s: %w", dir, err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("serializing config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing config %s: %w", path, err)
	}

	return nil
}
