package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/blackknight05/gemini-web-proxy/pkg/logger"
)

// CookieItem represents a single cookie object exported by browser extensions (Cookie-Editor, etc).
type CookieItem struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// CookieSession contains sanitized cookie string data and the extracted SAPISID token.
type CookieSession struct {
	HeaderValue string
	SAPISID     string
}

// SanitizeHeaderValue strips control characters, newlines, and carriage returns to ensure net/http compliance.
func SanitizeHeaderValue(val string) string {
	var sb strings.Builder
	for _, r := range val {
		// Keep valid HTTP header characters (ASCII 32 to 126)
		if r >= 32 && r <= 126 {
			sb.WriteRune(r)
		}
	}
	return strings.TrimSpace(sb.String())
}

// LoadCookies reads cookies from JSON arrays, JSON objects, or raw string formats.
func LoadCookies(path string) CookieSession {
	if path == "" {
		return CookieSession{}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		logger.Debug("No cookie file found at path: %s", path)
		return CookieSession{}
	}

	content := strings.TrimSpace(string(data))
	if content == "" {
		return CookieSession{}
	}

	var rawCookie string
	var sapisid string

	// Format 1: JSON Array exported directly by browser extensions (Cookie-Editor, Get cookies.txt, etc.)
	if strings.HasPrefix(content, "[") {
		var items []CookieItem
		if err := json.Unmarshal([]byte(content), &items); err == nil && len(items) > 0 {
			var parts []string
			for _, item := range items {
				if item.Name != "" {
					parts = append(parts, fmt.Sprintf("%s=%s", item.Name, item.Value))
					if item.Name == "SAPISID" {
						sapisid = item.Value
					}
				}
			}
			rawCookie = strings.Join(parts, "; ")
		}
	} else if strings.HasPrefix(content, "{") {
		// Format 2: JSON Object format
		var jsonObj map[string]interface{}
		if err := json.Unmarshal([]byte(content), &jsonObj); err == nil {
			if c, ok := jsonObj["cookie"].(string); ok {
				rawCookie = c
			} else if c, ok := jsonObj["Cookie"].(string); ok {
				rawCookie = c
			}
			if s, ok := jsonObj["sapisid"].(string); ok {
				sapisid = s
			} else if s, ok := jsonObj["SAPISID"].(string); ok {
				sapisid = s
			}
		}
	} else {
		// Format 3: Raw header string
		rawCookie = content
	}

	rawCookie = SanitizeHeaderValue(rawCookie)
	sapisid = SanitizeHeaderValue(sapisid)

	// Extract SAPISID from rawCookie if not explicitly found yet
	if sapisid == "" && rawCookie != "" {
		parts := strings.Split(rawCookie, ";")
		for _, part := range parts {
			kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
			if len(kv) == 2 && kv[0] == "SAPISID" {
				sapisid = kv[1]
				break
			}
		}
	}

	return CookieSession{
		HeaderValue: rawCookie,
		SAPISID:     sapisid,
	}
}
