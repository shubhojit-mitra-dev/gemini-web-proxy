package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/blackknight05/gemini-web-proxy/internal/config"
	"github.com/blackknight05/gemini-web-proxy/internal/gemini"
	"github.com/blackknight05/gemini-web-proxy/internal/models"
)

// GoogleHandler handles Google AI specification REST endpoints.
type GoogleHandler struct {
	cfg          *config.Config
	geminiClient *gemini.Client
}

// NewGoogleHandler constructs a new GoogleHandler instance.
func NewGoogleHandler(cfg *config.Config, geminiClient *gemini.Client) *GoogleHandler {
	return &GoogleHandler{
		cfg:          cfg,
		geminiClient: geminiClient,
	}
}

// HandleModelsList responds to GET /v1beta/models with Google Native models schemas.
func (h *GoogleHandler) HandleModelsList(w http.ResponseWriter, r *http.Request) {
	var modelList []map[string]interface{}
	for name, spec := range models.Registry {
		modelList = append(modelList, map[string]interface{}{
			"name":                        fmt.Sprintf("models/%s", name),
			"displayName":                 spec.DisplayName,
			"description":                 spec.Description,
			"supportedGenerationMethods": []string{"generateContent", "streamGenerateContent"},
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"models": modelList})
}

// HandleGenerateContent processes POST /v1beta/models/{model}:generateContent and :streamGenerateContent.
func (h *GoogleHandler) HandleGenerateContent(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	stream := strings.Contains(path, ":streamGenerateContent")

	modelName := ""
	if idx := strings.Index(path, "/models/"); idx != -1 {
		sub := path[idx+len("/models/"):]
		if colonIdx := strings.Index(sub, ":"); colonIdx != -1 {
			modelName = sub[:colonIdx]
		}
	}

	resolved, err := models.Resolve(modelName, h.cfg.DefaultModel)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	prompt := parseGoogleContents(req)
	text, err := h.geminiClient.StreamGenerate(r.Context(), prompt, resolved.ModeID, resolved.ThinkMode, nil)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, fmt.Sprintf("Upstream error: %v", err))
		return
	}

	responseObj := map[string]interface{}{
		"candidates": []map[string]interface{}{
			{
				"content": map[string]interface{}{
					"parts": []map[string]string{{"text": text}},
					"role":  "model",
				},
				"finishReason": "STOP",
				"index":        0,
			},
		},
		"usageMetadata": map[string]int{
			"promptTokenCount":     len(prompt) / 4,
			"candidatesTokenCount": len(text) / 4,
			"totalTokenCount":      (len(prompt) + len(text)) / 4,
		},
		"modelVersion": resolved.Name,
	}

	if stream {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		data, _ := json.Marshal(responseObj)
		fmt.Fprintf(w, "data: %s\n\n", data)
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		return
	}

	writeJSON(w, http.StatusOK, responseObj)
}

func parseGoogleContents(req map[string]interface{}) string {
	var parts []string
	contents, ok := req["contents"].([]interface{})
	if !ok {
		return ""
	}

	for _, content := range contents {
		m, ok := content.(map[string]interface{})
		if !ok {
			continue
		}
		role, _ := m["role"].(string)
		cParts, ok := m["parts"].([]interface{})
		if !ok {
			continue
		}

		var textParts []string
		for _, p := range cParts {
			pm, ok := p.(map[string]interface{})
			if !ok {
				continue
			}
			if t, ok := pm["text"].(string); ok {
				textParts = append(textParts, t)
			}
		}

		if role == "model" {
			parts = append(parts, fmt.Sprintf("[Assistant]: %s", strings.Join(textParts, " ")))
		} else {
			parts = append(parts, strings.Join(textParts, " "))
		}
	}

	return strings.Join(parts, "\n\n")
}
