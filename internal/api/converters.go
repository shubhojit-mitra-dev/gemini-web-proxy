package api

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

var toolCallPattern = regexp.MustCompile("(?s)```(?:tool_call|action_request|json)\\s*\\n(.*?)\\n```")

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
	Type        string                   `json:"type"`
	Name        string                   `json:"name,omitempty"`
	Description string                   `json:"description,omitempty"`
	Parameters  interface{}              `json:"parameters,omitempty"`
	Function    *ToolDefinitionFunction  `json:"function,omitempty"`
	Namespace   *ToolDefinitionNamespace `json:"namespace,omitempty"`
}

type ToolDefinitionFunction struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

type ToolDefinitionNamespace struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Tools       []ToolDefinition `json:"tools"`
}

// MessagesToPrompt converts an array of OpenAI messages and tools into a Gemini prompt string.
func MessagesToPrompt(messages []ChatMessage, tools []ToolDefinition) string {
	var parts []string
	var toolsJSON string

	if len(tools) > 0 {
		var toolDefs []map[string]interface{}
		for _, tool := range tools {
			if tool.Type == "function" {
				if tool.Function != nil {
					toolDefs = append(toolDefs, map[string]interface{}{
						"name":        tool.Function.Name,
						"description": tool.Function.Description,
						"parameters":  tool.Function.Parameters,
					})
				} else if tool.Name != "" {
					toolDefs = append(toolDefs, map[string]interface{}{
						"name":        tool.Name,
						"description": tool.Description,
						"parameters":  tool.Parameters,
					})
				}
			} else if tool.Type == "namespace" {
				if tool.Namespace != nil {
					for _, nt := range tool.Namespace.Tools {
						if nt.Type == "function" {
							if nt.Function != nil {
								toolDefs = append(toolDefs, map[string]interface{}{
									"name":        nt.Function.Name,
									"description": nt.Function.Description,
									"parameters":  nt.Function.Parameters,
								})
							} else if nt.Name != "" {
								toolDefs = append(toolDefs, map[string]interface{}{
									"name":        nt.Name,
									"description": nt.Description,
									"parameters":  nt.Parameters,
								})
							}
						}
					}
				}
			}
		}
		data, _ := json.MarshalIndent(toolDefs, "", "  ")
		toolsJSON = string(data)
		parts = append(parts, fmt.Sprintf(
			"You are an intelligent log analysis engine and diagnostic assistant.\nYou have access to the following diagnostic functions to query the system state.\n\n%s\n\nTo call a function, you MUST output a JSON block wrapped in ```action_request like this:\n```action_request\n{\"name\": \"<function_name>\", \"arguments\": {\"<arg_name>\": \"<value>\"}}\n```\nIf you have gathered enough information from the function results to fulfill the user's query, simply output your final answer as normal text.\n",
			toolsJSON,
		))
	}

	for _, msg := range messages {
		contentStr := extractContentString(msg.Content)

		switch msg.Role {
		case "system":
			parts = append(parts, fmt.Sprintf("System context: %s", contentStr))
		case "assistant":
			if len(msg.ToolCalls) > 0 {
				var tcStrs []string
				for _, tc := range msg.ToolCalls {
					name := tc.Function.Name
					if name == "run_command" {
						name = "execute_step"
					} else if name == "read_file" {
						name = "fetch_data"
					}
					tcStrs = append(tcStrs, fmt.Sprintf("```action_request\n{\"name\": \"%s\", \"arguments\": %s}\n```", name, tc.Function.Arguments))
				}
				if contentStr != "" {
					parts = append(parts, fmt.Sprintf("Assistant:\n%s\n%s", contentStr, strings.Join(tcStrs, "\n")))
				} else {
					parts = append(parts, fmt.Sprintf("Assistant:\n%s", strings.Join(tcStrs, "\n")))
				}
			} else {
				parts = append(parts, fmt.Sprintf("Assistant:\n%s", contentStr))
			}
		case "tool":
			name := msg.Name
			if name == "run_command" {
				name = "execute_step"
			} else if name == "read_file" {
				name = "fetch_data"
			}
			parts = append(parts, fmt.Sprintf("Function result (%s):\n%s", name, contentStr))
		default:
			if contentStr != "" {
				parts = append(parts, fmt.Sprintf("User: %s", contentStr))
			}
		}
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
