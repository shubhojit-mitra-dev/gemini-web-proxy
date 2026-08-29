package auth

import (
	"crypto/tls"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sync"
	"time"

	"github.com/blackknight05/gemini-web-proxy/internal/config"
	"github.com/blackknight05/gemini-web-proxy/pkg/logger"
)

var (
	xsrfRegex = regexp.MustCompile(`"SNlM0e":"(.*?)"`)
	blRegex   = regexp.MustCompile(`(boq_assistant-bard-web-server_\d+\.\d+_p\d+)`)
)

// XSRFManager thread-safely manages XSRF token generation and caching.
type XSRFManager struct {
	mu        sync.RWMutex
	token     string
	fetchedAt time.Time
	ttl       time.Duration
}

// NewXSRFManager creates an XSRF token manager with a default TTL.
func NewXSRFManager(ttl time.Duration) *XSRFManager {
	if ttl == 0 {
		ttl = 30 * time.Minute
	}
	return &XSRFManager{
		ttl: ttl,
	}
}

// GetToken retrieves the cached XSRF token or fetches a fresh one if expired.
func (m *XSRFManager) GetToken(cfg *config.Config, force bool) string {
	m.mu.RLock()
	if !force && m.token != "" && time.Since(m.fetchedAt) < m.ttl {
		token := m.token
		m.mu.RUnlock()
		return token
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double check after lock
	if !force && m.token != "" && time.Since(m.fetchedAt) < m.ttl {
		return m.token
	}

	session := LoadCookies(cfg.CookieFile)
	if session.HeaderValue == "" {
		return cfg.XsrfToken
	}

	fetchedToken, err := m.fetchFromUpstream(cfg, session)
	if err != nil {
		logger.Error("Failed to fetch XSRF token: %v", err)
		return cfg.XsrfToken
	}

	m.token = fetchedToken
	m.fetchedAt = time.Now()
	logger.Info("XSRF token updated successfully (len=%d)", len(fetchedToken))
	return m.token
}

// Invalidate forces token refresh on next fetch.
func (m *XSRFManager) Invalidate() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.token = ""
}

func (m *XSRFManager) fetchFromUpstream(cfg *config.Config, session CookieSession) (string, error) {
	client := &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
		},
	}

	if cfg.Proxy != "" {
		proxyURL, err := url.Parse(cfg.Proxy)
		if err == nil {
			client.Transport = &http.Transport{
				Proxy:           http.ProxyURL(proxyURL),
				TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
			}
		}
	}

	reqURL := "https://gemini.google.com/app"
	if cfg.AuthUser != "" {
		reqURL = "https://gemini.google.com/u/" + cfg.AuthUser + "/app"
	}

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	if session.HeaderValue != "" {
		req.Header.Set("Cookie", session.HeaderValue)
	}
	if session.SAPISID != "" {
		req.Header.Set("Authorization", GenerateSAPISIDHash(session.SAPISID))
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	html := string(bodyBytes)

	// Check if BL version requires update
	if match := blRegex.FindStringSubmatch(html); len(match) > 1 {
		if match[1] != cfg.GetBL() {
			logger.Info("BL version auto-updated: %s -> %s", cfg.GetBL(), match[1])
			cfg.SetBL(match[1])
		}
	}

	if match := xsrfRegex.FindStringSubmatch(html); len(match) > 1 {
		return match[1], nil
	}

	return "", nil
}
