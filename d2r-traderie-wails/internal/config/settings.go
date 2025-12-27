package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds all application configuration
type Config struct {
	// Hotkey for capturing items (e.g., "F9", "Ctrl+Shift+P")
	Hotkey string `json:"hotkey"`

	// Traderie API configuration
	Traderie TraderieConfig `json:"traderie"`

	// Overlay configuration
	Overlay OverlayConfig `json:"overlay"`

	// Notification settings
	Notifications NotificationConfig `json:"notifications"`

	// Logging level (debug, info, warn, error)
	LogLevel string `json:"log_level"`
}

// TraderieConfig holds Traderie-specific settings
type TraderieConfig struct {
	// API key or session token
	APIKey string `json:"api_key,omitempty"`
	
	// Username for authentication
	Username string `json:"username,omitempty"`
	
	// Platform (pc, playstation, xbox, switch)
	Platform string `json:"platform"`
	
	// Mode (softcore, hardcore)
	Mode string `json:"mode"`
	
	// Ladder (true, false)
	Ladder bool `json:"ladder"`
	
	// Region (Americas, Europe, Asia)
	Region string `json:"region"`
	
	// Auto-post without preview
	AutoPost bool `json:"auto_post"`

	// Auto-refresh listings enabled
	AutoRefreshEnabled bool `json:"auto_refresh_enabled"`

	// Auto-refresh interval in minutes
	AutoRefreshInterval int `json:"auto_refresh_interval"`

	// Default search range percentage
	SearchRange int `json:"search_range"`
}

// OverlayConfig holds overlay UI settings
type OverlayConfig struct {
	// Position (top-left, top-right, bottom-left, bottom-right)
	Position string `json:"position"`
	
	// X and Y offset from corner
	OffsetX int `json:"offset_x"`
	OffsetY int `json:"offset_y"`
	
	// Size
	Width  int `json:"width"`
	Height int `json:"height"`
	
	// Opacity (0.0 to 1.0)
	Opacity float32 `json:"opacity"`
}

// NotificationConfig holds notification settings
type NotificationConfig struct {
	// Enable sound notifications
	SoundEnabled bool `json:"sound_enabled"`
	
	// Show success messages
	ShowSuccess bool `json:"show_success"`
	
	// Show error messages
	ShowErrors bool `json:"show_errors"`
	
	// Auto-dismiss timeout (seconds)
	AutoDismissSeconds int `json:"auto_dismiss_seconds"`
}

// Default returns the default configuration
func Default() *Config {
	return &Config{
		Hotkey: "F9",
		Traderie: TraderieConfig{
			Platform:            "pc",
			Mode:                "softcore",
			Ladder:              true,
			Region:              "Americas",
			AutoPost:            false,
			AutoRefreshEnabled:  false,
			AutoRefreshInterval: 60, // 1 hour
			SearchRange:         20, // 20%
		},
		Overlay: OverlayConfig{
			Position: "top-right",
			OffsetX:  20,
			OffsetY:  20,
			Width:    400,
			Height:   300,
			Opacity:  0.95,
		},
		Notifications: NotificationConfig{
			SoundEnabled:       true,
			ShowSuccess:        true,
			ShowErrors:         true,
			AutoDismissSeconds: 5,
		},
		LogLevel: "info",
	}
}

// Load loads configuration from the config file
func Load() (*Config, error) {
	configPath := getConfigPath()
	
	// If config doesn't exist, return default
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfg := Default()
		// Try to save default config
		_ = cfg.Save()
		return cfg, nil
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	// Parse JSON
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Save saves the configuration to file
func (c *Config) Save() error {
	configPath := getConfigPath()
	
	// Ensure directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(configPath, data, 0644)
}

// getConfigPath returns the path to the config file
func getConfigPath() string {
	// Use user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	return filepath.Join(homeDir, ".d2r-traderie", "config.json")
}

