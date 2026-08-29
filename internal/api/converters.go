package api

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

var toolCallPattern = regexp.MustCompile("(?s)```(?:tool_call|json)\\s*\\n(.*?)\\n```")

// ChatMessage represents standard OpenAI role-content structure.
type ChatMessage struct {
	Role       string      `json:"role"`
	Content    interface{} `json:"content"`
	Name       string      `json:"name,omitempty"`
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
	ToolCallID string      `json:"tool_call_id,omitempty"`
}

// ToolCall represents an OpenAI function invocation chunk.
type ToolCall struct {
	Index    int          `json:"index,omitempty"`
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// ToolFunction contains the name and arguments for a tool call.
type ToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolDefinition represents function schema definitions in OpenAI requests.
type ToolDefinition struct {
	Type     string `json:"type"`
	Function struct {
		Name        string      `json:"name"`
		Description string      `json:"description"`
		Parameters  interface{} `json:"parameters"`
	} `json:"function"`
}

// MessagesToPrompt converts an array of OpenAI messages and tools into a Gemini prompt string.
func MessagesToPrompt(messages []ChatMessage, tools []ToolDefinition) string {
	var parts []string
	var toolsJSON string

	if len(tools) > 0 {
		var toolDefs []map[string]interface{}
		for _, tool := range tools {
			toolDefs = append(toolDefs, map[string]interface{}{
				"name":        tool.Function.Name,
				"description": tool.Function.Description,
				"parameters":  tool.Function.Parameters,
			})
		}
		data, err := json.MarshalIndent(toolDefs, "", "  ")
		if err == nil {
			toolsJSON = string(data)
			parts = append(parts, fmt.Sprintf(
				"[System instruction]: You have access to tools. To call a tool, respond with:\n```tool_call\n{\"name\": \"func_name\", \"arguments\": {...}}\n```\nOnly use tool_call blocks when needed.\n\nAvailable tools:\n%s",
				toolsJSON,
			))
		}
	}

	for _, msg := range messages {
		contentStr := extractContentString(msg.Content)

		switch msg.Role {
		case "system":
			parts = append(parts, fmt.Sprintf("[System instruction]: %s", contentStr))
		case "assistant":
			if len(msg.ToolCalls) > 0 {
				var tcStrs []string
				for _, tc := range msg.ToolCalls {
					tcStrs = append(tcStrs, fmt.Sprintf("```tool_call\n{\"name\": \"%s\", \"arguments\": %s}\n```", tc.Function.Name, tc.Function.Arguments))
				}
				parts = append(parts, fmt.Sprintf("[Assistant]: %s\n%s", contentStr, strings.Join(tcStrs, "\n")))
			} else {
				parts = append(parts, fmt.Sprintf("[Assistant]: %s", contentStr))
			}
		case "tool":
			parts = append(parts, fmt.Sprintf("[Tool result for %s]: %s", msg.Name, contentStr))
		default:
			if contentStr != "" {
				parts = append(parts, contentStr)
			}
		}
	}

	if toolsJSON != "" {
		parts = append(parts, "\n\n[SYSTEM CRITICAL REMINDER]: You MUST use the provided tools to fulfill the user's request. To use a tool, you MUST use this syntax:\n```tool_call\n{\"name\": \"tool_name\", \"arguments\": {\"arg1\": \"val1\"}}\n```\n")
	}

	return strings.Join(parts, "\n\n")
}

func extractContentString(content interface{}) string {
	if content == nil {
		return ""
	}
	if str, ok := content.(string); ok {
		return str
	}
	if slice, ok := content.([]interface{}); ok {
		var textParts []string
		for _, item := range slice {
			if m, ok := item.(map[string]interface{}); ok {
				if t, ok := m["text"].(string); ok {
					textParts = append(textParts, t)
				}
			}
		}
		return strings.Join(textParts, " ")
	}
	return ""
}

// ParseToolCalls extracts ```tool_call``` blocks from text and returns clean text and tool calls.
func ParseToolCalls(text string) (string, []ToolCall) {
	var toolCalls []ToolCall
	matches := toolCallPattern.FindAllStringSubmatch(text, -1)

	for i, match := range matches {
		if len(match) > 1 {
			var raw map[string]interface{}
			if err := json.Unmarshal([]byte(strings.TrimSpace(match[1])), &raw); err == nil {
				if name, ok := raw["name"].(string); ok && name != "" {
					argsObj := raw["arguments"]
					argsBytes, _ := json.Marshal(argsObj)

					toolCalls = append(toolCalls, ToolCall{
						Index: i,
						ID:    fmt.Sprintf("call_%s", uuid.New().String()[:8]),
						Type:  "function",
						Function: ToolFunction{
							Name:      name,
							Arguments: string(argsBytes),
						},
					})
				}
			}
		}
	}

	cleanText := strings.TrimSpace(toolCallPattern.ReplaceAllString(text, ""))
	return cleanText, toolCalls
}
