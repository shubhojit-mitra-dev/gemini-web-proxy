package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/blackknight05/gemini-web-proxy/pkg/logger"
)

// Config holds global configuration settings for the proxy server.
type Config struct {
	mu                sync.RWMutex `json:"-"`
	Port              int          `json:"port"`
	Host              string       `json:"host"`
	RetryAttempts     int          `json:"retry_attempts"`
	RetryDelaySec     int          `json:"retry_delay_sec"`
	RequestTimeoutSec int          `json:"request_timeout_sec"`
	GeminiBL          string       `json:"gemini_bl"`
	AuthUser          string       `json:"auth_user"`
	XsrfToken         string       `json:"xsrf_token"`
	DefaultModel      string       `json:"default_model"`
	LogRequests       bool         `json:"log_requests"`
	CookieFile        string       `json:"cookie_file"`
	Proxy             string       `json:"proxy"`
	APIKeys           []string     `json:"api_keys"`
	TemporaryChats    bool         `json:"temporary_chats"`
}

// Default returns a new Config populated with default production parameters.
func Default() *Config {
	return &Config{
		Port:              58120,
		Host:              "0.0.0.0",
		RetryAttempts:     3,
		RetryDelaySec:     2,
		RequestTimeoutSec: 180,
		GeminiBL:          "boq_assistant-bard-web-server_20260716.08_p0",
		AuthUser:          "",
		XsrfToken:         "",
		DefaultModel:      "gemini-3.6-flash",
		LogRequests:       true,
		CookieFile:        "",
		Proxy:             "",
		APIKeys:           make([]string, 0),
		TemporaryChats:    false,
	}
}

// LoadFile reads configuration settings from a JSON file path.
func (c *Config) LoadFile(path string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, c); err != nil {
		return err
	}

	logger.Info("Configuration loaded successfully from: %s", path)
	return nil
}

// AutoLoad searches default paths and environment variables for configuration files.
func (c *Config) AutoLoad(customPath string) {
	if customPath != "" {
		if err := c.LoadFile(customPath); err != nil {
			logger.Warn("Failed to load specified config file (%s): %v", customPath, err)
		}
		return
	}

	envPath := os.Getenv("GEMINI_WEB2API_CONFIG")
	if envPath != "" {
		if err := c.LoadFile(envPath); err == nil {
			return
		}
	}

	homeDir, err := os.UserHomeDir()
	candidatePaths := []string{"./config.json"}
	if err == nil {
		candidatePaths = append(candidatePaths, filepath.Join(homeDir, ".config", "gemini-web2api", "config.json"))
	}

	for _, p := range candidatePaths {
		if _, err := os.Stat(p); err == nil {
			if err := c.LoadFile(p); err == nil {
				break
			}
		}
	}

	// Auto-discover cookies.json if CookieFile is unset
	if c.CookieFile == "" {
		cookieCandidates := []string{"./cookies.json"}
		if homeDir, err := os.UserHomeDir(); err == nil {
			cookieCandidates = append(cookieCandidates, filepath.Join(homeDir, ".config", "gemini-web-proxy", "cookies.json"))
		}
		for _, cp := range cookieCandidates {
			if _, err := os.Stat(cp); err == nil {
				c.CookieFile = cp
				logger.Info("Discovered cookie file automatically: %s", cp)
				break
			}
		}
	}
}

// GetBL thread-safely returns the current Gemini BL string.
func (c *Config) GetBL() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.GeminiBL
}

// SetBL thread-safely updates the Gemini BL string.
func (c *Config) SetBL(bl string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.GeminiBL = bl
}
