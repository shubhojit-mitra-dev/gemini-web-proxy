package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/blackknight05/gemini-web-proxy/internal/config"
	"github.com/blackknight05/gemini-web-proxy/internal/gemini"
	"github.com/blackknight05/gemini-web-proxy/internal/models"
	"github.com/blackknight05/gemini-web-proxy/pkg/logger"
	"github.com/google/uuid"
)

// OpenAIHandler processes OpenAI specification API requests.
type OpenAIHandler struct {
	cfg          *config.Config
	geminiClient *gemini.Client
}

// NewOpenAIHandler creates a new OpenAIHandler instance.
func NewOpenAIHandler(cfg *config.Config, geminiClient *gemini.Client) *OpenAIHandler {
	return &OpenAIHandler{
		cfg:          cfg,
		geminiClient: geminiClient,
	}
}

// HandleModels responds to GET /v1/models with the registry list.
func (h *OpenAIHandler) HandleModels(w http.ResponseWriter, r *http.Request) {
	var modelList []models.ModelSpec
	for _, spec := range models.Registry {
		modelList = append(modelList, spec)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"object": "list",
		"data":   modelList,
		"models": modelList,
	})
}

type chatCompletionRequest struct {
	Model    string           `json:"model"`
	Messages []ChatMessage    `json:"messages"`
	Tools    []ToolDefinition `json:"tools"`
	Stream   bool             `json:"stream"`
}

// HandleChatCompletions processes POST /v1/chat/completions requests.
func (h *OpenAIHandler) HandleChatCompletions(w http.ResponseWriter, r *http.Request) {
	var req chatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	resolved, err := models.Resolve(req.Model, h.cfg.DefaultModel)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	prompt := MessagesToPrompt(req.Messages, req.Tools)
	if prompt == "" {
		writeJSONError(w, http.StatusBadRequest, "Prompt cannot be empty")
		return
	}

	cid := fmt.Sprintf("chatcmpl-%s", uuid.New().String()[:12])

	if req.Stream {
		h.streamChatCompletions(w, r, req, resolved, prompt, cid)
		return
	}

	h.syncChatCompletions(w, req, resolved, prompt, cid)
}

func (h *OpenAIHandler) streamChatCompletions(w http.ResponseWriter, r *http.Request, req chatCompletionRequest, resolved models.ResolvedModel, prompt string, cid string) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "close")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSONError(w, http.StatusInternalServerError, "Streaming unsupported")
		return
	}

	// First chunk with role="assistant"
	firstChunk := map[string]interface{}{
		"id":      cid,
		"object":  "chat.completion.chunk",
		"created": time.Now().Unix(),
		"model":   resolved.Name,
		"choices": []map[string]interface{}{
			{
				"index":         0,
				"delta":         map[string]interface{}{"role": "assistant", "content": ""},
				"finish_reason": nil,
			},
		},
	}
	data, _ := json.Marshal(firstChunk)
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()

	deltaChan, errChan := h.geminiClient.StreamGenerateIter(r.Context(), prompt, resolved.ModeID, resolved.ThinkMode, nil)

	var fullChunks []string
	for delta := range deltaChan {
		fullChunks = append(fullChunks, delta)
		chunk := map[string]interface{}{
			"id":      cid,
			"object":  "chat.completion.chunk",
			"created": time.Now().Unix(),
			"model":   resolved.Name,
			"choices": []map[string]interface{}{
				{
					"index":         0,
					"delta":         map[string]interface{}{"content": delta},
					"finish_reason": nil,
				},
			},
		}
		cData, _ := json.Marshal(chunk)
		fmt.Fprintf(w, "data: %s\n\n", cData)
		flusher.Flush()
	}

	if err := <-errChan; err != nil {
		logger.Error("Streaming error: %v", err)
		errChunk := map[string]interface{}{
			"id":      cid,
			"object":  "chat.completion.chunk",
			"created": time.Now().Unix(),
			"model":   resolved.Name,
			"choices": []map[string]interface{}{
				{
					"index":         0,
					"delta":         map[string]interface{}{"content": fmt.Sprintf("\n\n[Proxy Error: %v]", err)},
					"finish_reason": "stop",
				},
			},
		}
		cData, _ := json.Marshal(errChunk)
		fmt.Fprintf(w, "data: %s\n\n", cData)
		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
		return
	}

	fullText := ""
	for _, s := range fullChunks {
		fullText += s
	}

	var toolCalls []ToolCall
	if len(req.Tools) > 0 {
		_, toolCalls = ParseToolCalls(fullText)
	}

	if len(toolCalls) > 0 {
		tcChunk := map[string]interface{}{
			"id":      cid,
			"object":  "chat.completion.chunk",
			"created": time.Now().Unix(),
			"model":   resolved.Name,
			"choices": []map[string]interface{}{
				{
					"index":         0,
					"delta":         map[string]interface{}{"tool_calls": toolCalls},
					"finish_reason": "tool_calls",
				},
			},
		}
		cData, _ := json.Marshal(tcChunk)
		fmt.Fprintf(w, "data: %s\n\n", cData)
	} else {
		finalChunk := map[string]interface{}{
			"id":      cid,
			"object":  "chat.completion.chunk",
			"created": time.Now().Unix(),
			"model":   resolved.Name,
			"choices": []map[string]interface{}{
				{
					"index":         0,
					"delta":         map[string]interface{}{},
					"finish_reason": "stop",
				},
			},
		}
		cData, _ := json.Marshal(finalChunk)
		fmt.Fprintf(w, "data: %s\n\n", cData)
	}

	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()
}

func (h *OpenAIHandler) syncChatCompletions(w http.ResponseWriter, req chatCompletionRequest, resolved models.ResolvedModel, prompt string, cid string) {
	text, err := h.geminiClient.StreamGenerate(nil, prompt, resolved.ModeID, resolved.ThinkMode, nil)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, fmt.Sprintf("Upstream error: %v", err))
		return
	}

	var toolCalls []ToolCall
	if len(req.Tools) > 0 {
		text, toolCalls = ParseToolCalls(text)
	}

	msg := map[string]interface{}{
		"role":    "assistant",
		"content": text,
	}
	finishReason := "stop"
	if len(toolCalls) > 0 {
		msg["tool_calls"] = toolCalls
		finishReason = "tool_calls"
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":      cid,
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   resolved.Name,
		"choices": []map[string]interface{}{
			{
				"index":         0,
				"message":       msg,
				"finish_reason": finishReason,
			},
		},
		"usage": map[string]int{
			"prompt_tokens":     len(prompt) / 4,
			"completion_tokens": len(text) / 4,
			"total_tokens":      (len(prompt) + len(text)) / 4,
		},
	})
}

// HandleResponses implements the OpenAI Responses API required for OpenAI Codex CLI compatibility.
func (h *OpenAIHandler) HandleResponses(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	modelStr, _ := req["model"].(string)
	resolved, err := models.Resolve(modelStr, h.cfg.DefaultModel)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Extract messages from instructions and input items
	var messages []ChatMessage
	if inst, ok := req["instructions"].(string); ok && inst != "" {
		messages = append(messages, ChatMessage{Role: "system", Content: inst})
	}

	if input, ok := req["input"].(string); ok {
		messages = append(messages, ChatMessage{Role: "user", Content: input})
	} else if inputItems, ok := req["input"].([]interface{}); ok {
		for _, item := range inputItems {
			if str, ok := item.(string); ok {
				messages = append(messages, ChatMessage{Role: "user", Content: str})
			} else if m, ok := item.(map[string]interface{}); ok {
				role, _ := m["role"].(string)
				if role == "" {
					role = "user"
				}
				messages = append(messages, ChatMessage{Role: role, Content: m["content"]})
			}
		}
	}

	var tools []ToolDefinition
	if toolsRaw, ok := req["tools"].([]interface{}); ok {
		toolsBytes, _ := json.Marshal(toolsRaw)
		_ = json.Unmarshal(toolsBytes, &tools)
	}

	prompt := MessagesToPrompt(messages, tools)
	text, err := h.geminiClient.StreamGenerate(r.Context(), prompt, resolved.ModeID, resolved.ThinkMode, nil)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, fmt.Sprintf("Upstream error: %v", err))
		return
	}

	rid := fmt.Sprintf("resp_%s", uuid.New().String()[:16])
	mid := fmt.Sprintf("msg_%s", uuid.New().String()[:12])

	cleanText, toolCalls := ParseToolCalls(text)

	var output []map[string]interface{}
	if len(toolCalls) > 0 {
		for _, tc := range toolCalls {
			output = append(output, map[string]interface{}{
				"type":      "function_call",
				"id":        tc.ID,
				"call_id":   tc.ID,
				"name":      tc.Function.Name,
				"arguments": tc.Function.Arguments,
				"status":    "completed",
			})
		}
	}

	if cleanText != "" || len(toolCalls) == 0 {
		output = append(output, map[string]interface{}{
			"type":   "message",
			"id":     mid,
			"role":   "assistant",
			"status": "completed",
			"content": []map[string]interface{}{
				{
					"type":        "output_text",
					"text":        cleanText,
					"annotations": []interface{}{},
				},
			},
		})
	}

	stream, _ := req["stream"].(bool)
	if stream {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		flusher, ok := w.(http.Flusher)
		if !ok {
			writeJSONError(w, http.StatusInternalServerError, "Streaming unsupported")
			return
		}

		seq := 0
		emit := func(evType string, fields map[string]interface{}) {
			seq++
			ev := map[string]interface{}{
				"type":            evType,
				"sequence_number": seq,
			}
			for k, v := range fields {
				ev[k] = v
			}
			data, _ := json.Marshal(ev)
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", evType, data)
			flusher.Flush()
		}

		usage := map[string]int{
			"input_tokens":  len(prompt) / 4,
			"output_tokens": len(text) / 4,
			"total_tokens":  (len(prompt) + len(text)) / 4,
		}

		baseResp := map[string]interface{}{
			"id":         rid,
			"object":     "response",
			"created_at": time.Now().Unix(),
			"model":      resolved.Name,
		}

		emit("response.created", map[string]interface{}{
			"response": map[string]interface{}{
				"id":         rid,
				"object":     "response",
				"created_at": time.Now().Unix(),
				"model":      resolved.Name,
				"status":     "in_progress",
				"output":     []interface{}{},
				"usage":      nil,
			},
		})

		emit("response.in_progress", map[string]interface{}{
			"response": map[string]interface{}{
				"id":         rid,
				"object":     "response",
				"created_at": time.Now().Unix(),
				"model":      resolved.Name,
				"status":     "in_progress",
				"output":     []interface{}{},
				"usage":      nil,
			},
		})

		for oi, item := range output {
			itemType, _ := item["type"].(string)
			if itemType == "function_call" {
				pending := map[string]interface{}{
					"type":      "function_call",
					"id":        item["id"],
					"call_id":   item["call_id"],
					"name":      item["name"],
					"arguments": "",
					"status":    "in_progress",
				}
				emit("response.output_item.added", map[string]interface{}{"output_index": oi, "item": pending})
				emit("response.function_call_arguments.delta", map[string]interface{}{"item_id": item["id"], "output_index": oi, "delta": item["arguments"]})
				emit("response.function_call_arguments.done", map[string]interface{}{"item_id": item["id"], "output_index": oi, "arguments": item["arguments"]})
				emit("response.output_item.done", map[string]interface{}{"output_index": oi, "item": item})
			} else if itemType == "message" {
				pending := map[string]interface{}{
					"type":    "message",
					"id":      item["id"],
					"role":    "assistant",
					"status":  "in_progress",
					"content": []interface{}{},
				}
				emit("response.output_item.added", map[string]interface{}{"output_index": oi, "item": pending})
				contentList, _ := item["content"].([]map[string]interface{})
				for ci, cp := range contentList {
					emit("response.content_part.added", map[string]interface{}{
						"item_id":       item["id"],
						"output_index":  oi,
						"content_index": ci,
						"part":          map[string]interface{}{"type": "output_text", "text": "", "annotations": []interface{}{}},
					})
					emit("response.output_text.delta", map[string]interface{}{
						"item_id":       item["id"],
						"output_index":  oi,
						"content_index": ci,
						"delta":         cp["text"],
					})
					emit("response.output_text.done", map[string]interface{}{
						"item_id":       item["id"],
						"output_index":  oi,
						"content_index": ci,
						"text":          cp["text"],
					})
					emit("response.content_part.done", map[string]interface{}{
						"item_id":       item["id"],
						"output_index":  oi,
						"content_index": ci,
						"part":          cp,
					})
				}
				emit("response.output_item.done", map[string]interface{}{"output_index": oi, "item": item})
			}
		}

		emit("response.completed", map[string]interface{}{
			"response": map[string]interface{}{
				"id":         baseResp["id"],
				"object":     baseResp["object"],
				"created_at": baseResp["created_at"],
				"model":      baseResp["model"],
				"status":     "completed",
				"output":     output,
				"usage":      usage,
			},
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":         rid,
		"object":     "response",
		"created_at": time.Now().Unix(),
		"status":     "completed",
		"model":      resolved.Name,
		"output":     output,
		"usage": map[string]int{
			"input_tokens":  len(prompt) / 4,
			"output_tokens": len(text) / 4,
			"total_tokens":  (len(prompt) + len(text)) / 4,
		},
	})
}
