package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	ConfigDir  = ".aws-terminal"
	ConfigFile = "config.json"
)

// Profile represents an AWS SSO profile configuration
type Profile struct {
	Name    string `json:"name"`
	SSOUrl  string `json:"sso_url"`
	Region  string `json:"region,omitempty"`
	Default bool   `json:"default,omitempty"`
}

// Config represents the application configuration
type Config struct {
	Profiles []Profile `json:"profiles"`
}

// GetConfigPath returns the full path to the config file
func GetConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ConfigDir, ConfigFile), nil
}

// GetConfigDir returns the config directory path
func GetConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ConfigDir), nil
}

// Load reads the configuration from the config file
func Load() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New("config file not found")
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// Save writes the configuration to the config file
func (c *Config) Save() error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetDefaultProfile returns the default profile if one exists
func (c *Config) GetDefaultProfile() *Profile {
	for i := range c.Profiles {
		if c.Profiles[i].Default {
			return &c.Profiles[i]
		}
	}
	return nil
}

// GetProfileByName returns a profile by its name
func (c *Config) GetProfileByName(name string) *Profile {
	for i := range c.Profiles {
		if c.Profiles[i].Name == name {
			return &c.Profiles[i]
		}
	}
	return nil
}

// AddProfile adds a new profile or updates an existing one
func (c *Config) AddProfile(profile Profile) {
	// Check if profile with same URL already exists
	for i := range c.Profiles {
		if c.Profiles[i].SSOUrl == profile.SSOUrl {
			c.Profiles[i].Name = profile.Name
			if profile.Default {
				c.SetDefault(profile.Name)
			}
			return
		}
	}

	// Add new profile
	c.Profiles = append(c.Profiles, profile)
	if profile.Default {
		c.SetDefault(profile.Name)
	}
}

// SetDefault sets a profile as the default
func (c *Config) SetDefault(name string) {
	for i := range c.Profiles {
		c.Profiles[i].Default = (c.Profiles[i].Name == name)
	}
}

// ProfileExists checks if a profile with the given URL already exists
func (c *Config) ProfileExists(ssoUrl string) bool {
	for _, p := range c.Profiles {
		if p.SSOUrl == ssoUrl {
			return true
		}
	}
	return false
}

