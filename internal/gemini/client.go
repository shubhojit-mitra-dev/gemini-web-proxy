package gemini

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/blackknight05/gemini-web-proxy/internal/auth"
	"github.com/blackknight05/gemini-web-proxy/internal/config"
	"github.com/blackknight05/gemini-web-proxy/pkg/logger"
)

// Client handles network interactions with Google Gemini backend services.
type Client struct {
	cfg         *config.Config
	xsrfManager *auth.XSRFManager
	httpClient  *http.Client
}

// NewClient constructs a new Gemini Client instance.
func NewClient(cfg *config.Config, xsrfManager *auth.XSRFManager) *Client {
	httpClient := &http.Client{
		Timeout: time.Duration(cfg.RequestTimeoutSec) * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
		},
	}

	if cfg.Proxy != "" {
		proxyURL, err := url.Parse(cfg.Proxy)
		if err == nil {
			httpClient.Transport = &http.Transport{
				Proxy:           http.ProxyURL(proxyURL),
				TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
			}
		}
	}

	return &Client{
		cfg:         cfg,
		xsrfManager: xsrfManager,
		httpClient:  httpClient,
	}
}

func (c *Client) buildRequest(ctx context.Context, prompt string, modelID int, thinkMode int, fileRefs []string) (*http.Request, error) {
	outerJSON, err := BuildStreamGeneratePayload(prompt, modelID, thinkMode, c.cfg.TemporaryChats, fileRefs)
	if err != nil {
		return nil, err
	}

	xsrf := c.xsrfManager.GetToken(c.cfg, false)
	bodyStr := EncodeFormBody(outerJSON, xsrf)

	reqID := time.Now().UnixNano() / 1e3 % 1000000
	prefix := ""
	if c.cfg.AuthUser != "" {
		prefix = "/u/" + c.cfg.AuthUser
	}

	reqURL := fmt.Sprintf(
		"https://gemini.google.com%s/_/BardChatUi/data/assistant.lamda.BardFrontendService/StreamGenerate?bl=%s&hl=en&_reqid=%d&rt=c",
		prefix, c.cfg.GetBL(), reqID,
	)

	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, strings.NewReader(bodyStr))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "https://gemini.google.com")
	req.Header.Set("Referer", fmt.Sprintf("https://gemini.google.com%s/app", prefix))
	req.Header.Set("X-Same-Domain", "1")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	if c.cfg.AuthUser != "" {
		req.Header.Set("X-Goog-AuthUser", c.cfg.AuthUser)
	}

	session := auth.LoadCookies(c.cfg.CookieFile)
	if session.HeaderValue != "" {
		req.Header.Set("Cookie", session.HeaderValue)
	}
	if session.SAPISID != "" {
		req.Header.Set("Authorization", auth.GenerateSAPISIDHash(session.SAPISID))
	}

	return req, nil
}

// StreamGenerateIter executes a prompt against Gemini and streams incremental deltas over a Go channel.
func (c *Client) StreamGenerateIter(ctx context.Context, prompt string, modelID int, thinkMode int, fileRefs []string) (<-chan string, <-chan error) {
	deltaChan := make(chan string, 10)
	errChan := make(chan error, 1)

	go func() {
		defer close(deltaChan)
		defer close(errChan)

		var resp *http.Response
		var lastErr error

		for attempt := 0; attempt < c.cfg.RetryAttempts; attempt++ {
			req, err := c.buildRequest(ctx, prompt, modelID, thinkMode, fileRefs)
			if err != nil {
				errChan <- err
				return
			}

			res, err := c.httpClient.Do(req)
			if err != nil {
				lastErr = err
				logger.Warn("Request attempt %d failed: %v", attempt+1, err)
				time.Sleep(time.Duration(c.cfg.RetryDelaySec) * time.Second)
				continue
			}

			if res.StatusCode == http.StatusBadRequest && attempt == 0 {
				res.Body.Close()
				logger.Warn("Upstream HTTP 400 (XSRF stale), invalidating token and retrying...")
				c.xsrfManager.Invalidate()
				c.xsrfManager.GetToken(c.cfg, true)
				continue
			}

			if res.StatusCode != http.StatusOK {
				res.Body.Close()
				lastErr = fmt.Errorf("upstream returned HTTP status %d", res.StatusCode)
				logger.Warn("Request attempt %d HTTP error: %v", attempt+1, lastErr)
				time.Sleep(time.Duration(c.cfg.RetryDelaySec) * time.Second)
				continue
			}

			resp = res
			break
		}

		if resp == nil {
			if lastErr == nil {
				lastErr = fmt.Errorf("upstream request failed after max retries")
			}
			errChan <- lastErr
			return
		}
		defer resp.Body.Close()

		reader := bufio.NewReader(resp.Body)
		prevText := ""
		yielded := false

		for {
			line, err := reader.ReadString('\n')
			if len(line) > 0 {
				delta, newPrev, parseErr := ParseChunkLine(line, prevText)
				if parseErr != nil {
					errChan <- parseErr
					return
				}
				if delta != "" {
					prevText = newPrev
					yielded = true
					deltaChan <- delta
				}
			}

			if err != nil {
				if err == io.EOF {
					break
				}
				errChan <- err
				return
			}
		}

		if !yielded {
			errChan <- fmt.Errorf("gemini API returned no valid data chunks")
		}
	}()

	return deltaChan, errChan
}

// StreamGenerate executes a prompt against Gemini and returns the complete text response.
func (c *Client) StreamGenerate(ctx context.Context, prompt string, modelID int, thinkMode int, fileRefs []string) (string, error) {
	deltaChan, errChan := c.StreamGenerateIter(ctx, prompt, modelID, thinkMode, fileRefs)

	var sb strings.Builder
	for delta := range deltaChan {
		sb.WriteString(delta)
	}

	if err := <-errChan; err != nil {
		return "", err
	}

	return CleanGeminiText(sb.String(), true), nil
}
