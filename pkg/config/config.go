package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	appName        = "jav"
	configFileName = "config.json"
)

var supportedKeys = []string{"cookies", "proxy", "locale"}

// Config holds the application configuration.
type Config struct {
	Cookies string `json:"cookies,omitempty"`
	Proxy   string `json:"proxy,omitempty"`
	Locale  string `json:"locale,omitempty"`
}

// GetConfigDir returns the config directory following XDG Base Directory Specification.
func GetConfigDir() (string, error) {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}
		configHome = filepath.Join(home, ".config")
	}
	return filepath.Join(configHome, appName), nil
}

// GetConfigPath returns the path to the config file.
func GetConfigPath() (string, error) {
	dir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configFileName), nil
}

// Load loads the configuration from file.
// If the file doesn't exist, returns an empty config.
func Load() (*Config, error) {
	path, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}
	return &cfg, nil
}

// Save writes the configuration atomically with restrictive permissions.
func (c *Config) Save() error {
	dir, err := GetConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	path, err := GetConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(dir, configFileName+".tmp.*")
	if err != nil {
		return fmt.Errorf("failed to create temp config file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to set config file permissions: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to write config file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close config file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	return nil
}

// Set sets a configuration value by key.
func (c *Config) Set(key, value string) error {
	switch key {
	case "cookies":
		c.Cookies = value
	case "proxy":
		c.Proxy = value
	case "locale":
		c.Locale = value
	default:
		return fmt.Errorf("unknown config key: %s (supported: %s)", key, joinKeys())
	}
	return nil
}

// Get gets a configuration value by key.
func (c *Config) Get(key string) (string, error) {
	switch key {
	case "cookies":
		return c.Cookies, nil
	case "proxy":
		return c.Proxy, nil
	case "locale":
		return c.Locale, nil
	default:
		return "", fmt.Errorf("unknown config key: %s (supported: %s)", key, joinKeys())
	}
}

// ToMap converts the config to a map for display.
func (c *Config) ToMap() map[string]string {
	m := make(map[string]string, 3)
	if c.Cookies != "" {
		m["cookies"] = c.Cookies
	}
	if c.Proxy != "" {
		m["proxy"] = c.Proxy
	}
	if c.Locale != "" {
		m["locale"] = c.Locale
	}
	return m
}

func joinKeys() string {
	return strings.Join(supportedKeys, ", ")
}
