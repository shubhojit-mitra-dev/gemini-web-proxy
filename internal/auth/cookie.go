package auth

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/blackknight05/gemini-web-proxy/pkg/logger"
)

// CookieSession contains cookie string data and the extracted SAPISID token.
type CookieSession struct {
	HeaderValue string
	SAPISID     string
}

// LoadCookies reads cookies from a JSON file or raw string format.
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

	// Attempt JSON format parse
	if strings.HasPrefix(content, "{") {
		var jsonObj struct {
			Cookie  string `json:"cookie"`
			SAPISID string `json:"sapisid"`
		}
		if err := json.Unmarshal([]byte(content), &jsonObj); err == nil {
			rawCookie = jsonObj.Cookie
			sapisid = jsonObj.SAPISID
		}
	} else {
		rawCookie = content
	}

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
