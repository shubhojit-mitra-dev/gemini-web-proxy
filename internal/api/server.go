package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/blackknight05/gemini-web-proxy/internal/config"
	"github.com/blackknight05/gemini-web-proxy/internal/gemini"
	"github.com/blackknight05/gemini-web-proxy/internal/models"
	"github.com/blackknight05/gemini-web-proxy/pkg/logger"
)

// Server encapsulates the HTTP router and API handlers.
type Server struct {
	cfg           *config.Config
	openaiHandler *OpenAIHandler
	googleHandler *GoogleHandler
}

// NewServer creates a new API Server instance.
func NewServer(cfg *config.Config, geminiClient *gemini.Client) *Server {
	return &Server{
		cfg:           cfg,
		openaiHandler: NewOpenAIHandler(cfg, geminiClient),
		googleHandler: NewGoogleHandler(cfg, geminiClient),
	}
}

// ServeHTTP acts as the main HTTP router and middleware dispatcher.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	clientIP := r.RemoteAddr
	logger.Info("%s %s %s", clientIP, r.Method, r.URL.Path)

	// Handle CORS preflight
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Validate API keys if configured
	if strings.HasPrefix(r.URL.Path, "/v1") && !s.isAuthorized(r) {
		writeJSONError(w, http.StatusUnauthorized, "Invalid API key")
		return
	}

	// Route dispatching
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/":
		s.handleHealth(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/v1/models":
		s.openaiHandler.HandleModels(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/v1beta/models":
		s.googleHandler.HandleModelsList(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/v1/chat/completions":
		s.openaiHandler.HandleChatCompletions(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/v1/responses":
		s.openaiHandler.HandleResponses(w, r)
	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/v1beta/models"):
		s.googleHandler.HandleGenerateContent(w, r)
	default:
		writeJSONError(w, http.StatusNotFound, "Endpoint not found")
	}
}

func (s *Server) isAuthorized(r *http.Request) bool {
	keys := s.cfg.APIKeys
	if len(keys) == 0 {
		return true
	}

	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		for _, k := range keys {
			if token == k {
				return true
			}
		}
	}

	for _, h := range []string{"x-api-key", "x-goog-api-key"} {
		val := r.Header.Get(h)
		for _, k := range keys {
			if val == k {
				return true
			}
		}
	}

	if keyParam := r.URL.Query().Get("key"); keyParam != "" {
		for _, k := range keys {
			if keyParam == k {
				return true
			}
		}
	}

	return false
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	var modelNames []string
	for k := range models.Registry {
		modelNames = append(modelNames, k)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "ok",
		"version": "1.0.0",
		"models":  modelNames,
	})
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]interface{}{
		"error": map[string]string{
			"message": message,
		},
	})
}
