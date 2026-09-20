package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	appName        = "jav"
	configFileName = "config.json"
)

// Config holds the application configuration.
type Config struct {
	Cookies string `json:"cookies,omitempty"`
	Proxy   string `json:"proxy,omitempty"`
	Locale  string `json:"locale,omitempty"`
}

// GetConfigDir returns the config directory following XDG Base Directory Specification.
// Returns $XDG_CONFIG_HOME/jav if XDG_CONFIG_HOME is set, otherwise ~/.config/jav
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

	// If file doesn't exist, return empty config
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &Config{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// Save saves the configuration to file.
func (c *Config) Save() error {
	dir, err := GetConfigDir()
	if err != nil {
		return err
	}

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(dir, 0755); err != nil {
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

	if err := os.WriteFile(path, data, 0644); err != nil {
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
		return fmt.Errorf("unknown config key: %s (supported: cookies, proxy, locale)", key)
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
		return "", fmt.Errorf("unknown config key: %s", key)
	}
}

// ToMap converts the config to a map for display.
func (c *Config) ToMap() map[string]string {
	m := make(map[string]string)
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

// ApplyToEnv applies config values to environment variables.
// This allows existing code to work without changes.
func (c *Config) ApplyToEnv() {
	if c.Cookies != "" && os.Getenv("JAVDB_COOKIES") == "" {
		os.Setenv("JAVDB_COOKIES", c.Cookies)
	}
	if c.Proxy != "" && os.Getenv("SOCKS5_PROXY") == "" {
		os.Setenv("SOCKS5_PROXY", c.Proxy)
	}
	if c.Locale != "" && os.Getenv("JAVDB_LOCALE") == "" {
		os.Setenv("JAVDB_LOCALE", c.Locale)
	}
}
