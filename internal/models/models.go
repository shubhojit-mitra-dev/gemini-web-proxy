package models

import (
	"fmt"
	"strconv"
	"strings"
)

// ModelSpec defines the internal routing and metadata attributes for a model.
type ModelSpec struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	DisplayName string `json:"display_name"`
	Object      string `json:"object"`
	Created     int64  `json:"created"`
	OwnedBy     string `json:"owned_by"`
	Description string `json:"description"`

	Mode  int `json:"-"`
	Think int `json:"-"`

	// Strict metadata fields required by Codex v0.151.0 and modern client TUIs
	SupportedReasoningLevels []string         `json:"supported_reasoning_levels"`
	ShellType                string           `json:"shell_type"`
	Visibility               string           `json:"visibility"`
	SupportedInAPI           bool             `json:"supported_in_api"`
	Priority                 int              `json:"priority"`
	SupportVerbosity         bool             `json:"support_verbosity"`
	TruncationPolicy         TruncationConfig `json:"truncation_policy"`
	ModeString               string           `json:"mode"`
	IsAlias                  bool             `json:"-"`
}

// TruncationConfig describes model truncation settings for Codex schema validation.
type TruncationConfig struct {
	Type string `json:"type"`
}

// Registry maintains available Gemini models and OpenAI fallback alias mappings.
var Registry = map[string]ModelSpec{
	"gemini-3.7-flash": {
		ID: "gemini-3.7-flash", Slug: "gemini-3.7-flash", DisplayName: "Gemini 3.7 Flash",
		Object: "model", Created: 1700000000, OwnedBy: "google", Description: "Latest all-around model (Gemini 3.7 Flash)",
		Mode: 1, Think: 4, SupportedReasoningLevels: []string{}, ShellType: "default",
		Visibility: "list", SupportedInAPI: true, Priority: 0, SupportVerbosity: false,
		TruncationPolicy: TruncationConfig{Type: "auto"}, ModeString: "chat",
	},
	"gemini-3.6-flash": {
		ID: "gemini-3.6-flash", Slug: "gemini-3.6-flash", DisplayName: "Gemini 3.6 Flash",
		Object: "model", Created: 1700000000, OwnedBy: "google", Description: "All-around model (Gemini 3.6 Flash)",
		Mode: 1, Think: 4, SupportedReasoningLevels: []string{}, ShellType: "default",
		Visibility: "list", SupportedInAPI: true, Priority: 0, SupportVerbosity: false,
		TruncationPolicy: TruncationConfig{Type: "auto"}, ModeString: "chat",
	},
	"gemini-3.5-flash": {
		ID: "gemini-3.5-flash", Slug: "gemini-3.5-flash", DisplayName: "Gemini 3.5 Flash",
		Object: "model", Created: 1700000000, OwnedBy: "google", Description: "Alias for gemini-3.6-flash",
		Mode: 1, Think: 4, SupportedReasoningLevels: []string{}, ShellType: "default",
		Visibility: "list", SupportedInAPI: true, Priority: 0, SupportVerbosity: false,
		TruncationPolicy: TruncationConfig{Type: "auto"}, ModeString: "chat",
	},
	"gemini-3.5-flash-thinking": {
		ID: "gemini-3.5-flash-thinking", Slug: "gemini-3.5-flash-thinking", DisplayName: "Gemini 3.5 Flash Thinking",
		Object: "model", Created: 1700000000, OwnedBy: "google", Description: "Deep thinking mode",
		Mode: 2, Think: 0, SupportedReasoningLevels: []string{}, ShellType: "default",
		Visibility: "list", SupportedInAPI: true, Priority: 0, SupportVerbosity: false,
		TruncationPolicy: TruncationConfig{Type: "auto"}, ModeString: "chat",
	},
	"gemini-3.1-pro": {
		ID: "gemini-3.1-pro", Slug: "gemini-3.1-pro", DisplayName: "Gemini 3.1 Pro",
		Object: "model", Created: 1700000000, OwnedBy: "google", Description: "Pro model (requires session cookies)",
		Mode: 3, Think: 4, SupportedReasoningLevels: []string{}, ShellType: "default",
		Visibility: "list", SupportedInAPI: true, Priority: 0, SupportVerbosity: false,
		TruncationPolicy: TruncationConfig{Type: "auto"}, ModeString: "chat",
	},
	"gemini-auto": {
		ID: "gemini-auto", Slug: "gemini-auto", DisplayName: "Gemini Auto Selection",
		Object: "model", Created: 1700000000, OwnedBy: "google", Description: "Auto model selection",
		Mode: 4, Think: 4, SupportedReasoningLevels: []string{}, ShellType: "default",
		Visibility: "list", SupportedInAPI: true, Priority: 0, SupportVerbosity: false,
		TruncationPolicy: TruncationConfig{Type: "auto"}, ModeString: "chat",
	},
	"gemini-3.5-flash-thinking-lite": {
		ID: "gemini-3.5-flash-thinking-lite", Slug: "gemini-3.5-flash-thinking-lite", DisplayName: "Gemini Flash Thinking Lite",
		Object: "model", Created: 1700000000, OwnedBy: "google", Description: "Dynamic thinking with adaptive depth",
		Mode: 5, Think: 4, SupportedReasoningLevels: []string{}, ShellType: "default",
		Visibility: "list", SupportedInAPI: true, Priority: 0, SupportVerbosity: false,
		TruncationPolicy: TruncationConfig{Type: "auto"}, ModeString: "chat",
	},
	"gemini-flash-lite": {
		ID: "gemini-flash-lite", Slug: "gemini-flash-lite", DisplayName: "Gemini Flash Lite",
		Object: "model", Created: 1700000000, OwnedBy: "google", Description: "Lightweight fast model",
		Mode: 6, Think: 4, SupportedReasoningLevels: []string{}, ShellType: "default",
		Visibility: "list", SupportedInAPI: true, Priority: 0, SupportVerbosity: false,
		TruncationPolicy: TruncationConfig{Type: "auto"}, ModeString: "chat",
	},

	// OpenAI mock aliases mapped to exact Codex CLI TUI dropdown choices
	"gpt-5.6-sol": {
		ID: "gpt-5.6-sol", Slug: "gpt-5.6-sol", DisplayName: "GPT-5.6 Sol (Gemini 3.1 Pro)",
		Object: "model", Created: 1700000000, OwnedBy: "google", Description: "Frontier agentic model (Mapped to Gemini 3.1 Pro)",
		Mode: 3, Think: 4, SupportedReasoningLevels: []string{}, ShellType: "default",
		Visibility: "list", SupportedInAPI: true, Priority: 0, SupportVerbosity: false,
		TruncationPolicy: TruncationConfig{Type: "auto"}, ModeString: "chat", IsAlias: true,
	},
	"gpt-5.6-terra": {
		ID: "gpt-5.6-terra", Slug: "gpt-5.6-terra", DisplayName: "GPT-5.6 Terra (Gemini 3.5 Flash Thinking)",
		Object: "model", Created: 1700000000, OwnedBy: "google", Description: "Balanced agentic model (Mapped to Gemini 3.5 Flash Thinking)",
		Mode: 2, Think: 0, SupportedReasoningLevels: []string{}, ShellType: "default",
		Visibility: "list", SupportedInAPI: true, Priority: 0, SupportVerbosity: false,
		TruncationPolicy: TruncationConfig{Type: "auto"}, ModeString: "chat", IsAlias: true,
	},
	"gpt-5.6-luna": {
		ID: "gpt-5.6-luna", Slug: "gpt-5.6-luna", DisplayName: "GPT-5.6 Luna (Gemini Flash Thinking Lite)",
		Object: "model", Created: 1700000000, OwnedBy: "google", Description: "Fast agentic model (Mapped to Gemini Flash Thinking Lite)",
		Mode: 5, Think: 4, SupportedReasoningLevels: []string{}, ShellType: "default",
		Visibility: "list", SupportedInAPI: true, Priority: 0, SupportVerbosity: false,
		TruncationPolicy: TruncationConfig{Type: "auto"}, ModeString: "chat", IsAlias: true,
	},
	"gpt-5.5": {
		ID: "gpt-5.5", Slug: "gpt-5.5", DisplayName: "GPT-5.5 (Gemini 3.7 Flash)",
		Object: "model", Created: 1700000000, OwnedBy: "google", Description: "Complex coding model (Mapped to Gemini 3.7 Flash)",
		Mode: 1, Think: 4, SupportedReasoningLevels: []string{}, ShellType: "default",
		Visibility: "list", SupportedInAPI: true, Priority: 0, SupportVerbosity: false,
		TruncationPolicy: TruncationConfig{Type: "auto"}, ModeString: "chat", IsAlias: true,
	},
	"gpt-5.4": {
		ID: "gpt-5.4", Slug: "gpt-5.4", DisplayName: "GPT-5.4 (Gemini 3.6 Flash)",
		Object: "model", Created: 1700000000, OwnedBy: "google", Description: "Everyday model (Mapped to Gemini 3.6 Flash)",
		Mode: 1, Think: 4, SupportedReasoningLevels: []string{}, ShellType: "default",
		Visibility: "list", SupportedInAPI: true, Priority: 0, SupportVerbosity: false,
		TruncationPolicy: TruncationConfig{Type: "auto"}, ModeString: "chat", IsAlias: true,
	},
	"gpt-5.4-mini": {
		ID: "gpt-5.4-mini", Slug: "gpt-5.4-mini", DisplayName: "GPT-5.4 Mini (Gemini Flash Lite)",
		Object: "model", Created: 1700000000, OwnedBy: "google", Description: "Small fast model (Mapped to Gemini Flash Lite)",
		Mode: 6, Think: 4, SupportedReasoningLevels: []string{}, ShellType: "default",
		Visibility: "list", SupportedInAPI: true, Priority: 0, SupportVerbosity: false,
		TruncationPolicy: TruncationConfig{Type: "auto"}, ModeString: "chat", IsAlias: true,
	},
}

// NativeModels returns only primary non-alias Gemini model IDs.
func NativeModels() []string {
	var native []string
	for k, spec := range Registry {
		if !spec.IsAlias {
			native = append(native, k)
		}
	}
	return native
}

// ResolvedModel holds the final resolved model attributes for upstream request generation.
type ResolvedModel struct {
	Name      string
	ModeID    int
	ThinkMode int
}

// Resolve identifies a model from request input and handles optional @think= overrides.
func Resolve(inputName string, defaultModel string) (ResolvedModel, error) {
	if inputName == "" {
		inputName = defaultModel
	}

	thinkOverride := -1
	cleanName := inputName

	if idx := strings.LastIndex(inputName, "@think="); idx != -1 {
		cleanName = inputName[:idx]
		thinkStr := inputName[idx+len("@think="):]
		val, err := strconv.Atoi(thinkStr)
		if err == nil {
			thinkOverride = val
		}
	}

	spec, exists := Registry[cleanName]
	if !exists {
		return ResolvedModel{}, fmt.Errorf("unknown model: %s", inputName)
	}

	finalThink := spec.Think
	if thinkOverride != -1 {
		finalThink = thinkOverride
	}

	return ResolvedModel{
		Name:      cleanName,
		ModeID:    spec.Mode,
		ThinkMode: finalThink,
	}, nil
}
