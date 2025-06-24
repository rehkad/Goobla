package types

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/goobla/goobla/api"
	"github.com/goobla/goobla/types/model"
)

var FinishReasonToolCalls = "tool_calls"

type Error struct {
	Message string  `json:"message"`
	Type    string  `json:"type"`
	Param   any     `json:"param"`
	Code    *string `json:"code"`
}

type ErrorResponse struct {
	Error Error `json:"error"`
}

type Message struct {
	Role      string     `json:"role"`
	Content   any        `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason *string `json:"finish_reason"`
}

type ChunkChoice struct {
	Index        int     `json:"index"`
	Delta        Message `json:"delta"`
	FinishReason *string `json:"finish_reason"`
}

type CompleteChunkChoice struct {
	Text         string  `json:"text"`
	Index        int     `json:"index"`
	FinishReason *string `json:"finish_reason"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ResponseFormat struct {
	Type       string      `json:"type"`
	JsonSchema *JsonSchema `json:"json_schema,omitempty"`
}

type JsonSchema struct {
	Schema json.RawMessage `json:"schema"`
}

type EmbedRequest struct {
	Input any    `json:"input"`
	Model string `json:"model"`
}

type StreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type ChatCompletionRequest struct {
	Model            string          `json:"model"`
	Messages         []Message       `json:"messages"`
	Stream           bool            `json:"stream"`
	StreamOptions    *StreamOptions  `json:"stream_options"`
	MaxTokens        *int            `json:"max_tokens"`
	Seed             *int            `json:"seed"`
	Stop             any             `json:"stop"`
	Temperature      *float64        `json:"temperature"`
	FrequencyPenalty *float64        `json:"frequency_penalty"`
	PresencePenalty  *float64        `json:"presence_penalty"`
	TopP             *float64        `json:"top_p"`
	ResponseFormat   *ResponseFormat `json:"response_format"`
	Tools            []api.Tool      `json:"tools"`
}

type ChatCompletion struct {
	Id                string   `json:"id"`
	Object            string   `json:"object"`
	Created           int64    `json:"created"`
	Model             string   `json:"model"`
	SystemFingerprint string   `json:"system_fingerprint"`
	Choices           []Choice `json:"choices"`
	Usage             Usage    `json:"usage,omitempty"`
}

type ChatCompletionChunk struct {
	Id                string        `json:"id"`
	Object            string        `json:"object"`
	Created           int64         `json:"created"`
	Model             string        `json:"model"`
	SystemFingerprint string        `json:"system_fingerprint"`
	Choices           []ChunkChoice `json:"choices"`
	Usage             *Usage        `json:"usage,omitempty"`
}

// TODO (https://github.com/goobla/goobla/issues/5259): support []string, []int and [][]int

type CompletionRequest struct {
	Model            string         `json:"model"`
	Prompt           any            `json:"prompt"`
	FrequencyPenalty float32        `json:"frequency_penalty"`
	MaxTokens        *int           `json:"max_tokens"`
	PresencePenalty  float32        `json:"presence_penalty"`
	Seed             *int           `json:"seed"`
	Stop             any            `json:"stop"`
	Stream           bool           `json:"stream"`
	StreamOptions    *StreamOptions `json:"stream_options"`
	Temperature      *float32       `json:"temperature"`
	TopP             float32        `json:"top_p"`
	Suffix           string         `json:"suffix"`
}

type Completion struct {
	Id                string                `json:"id"`
	Object            string                `json:"object"`
	Created           int64                 `json:"created"`
	Model             string                `json:"model"`
	SystemFingerprint string                `json:"system_fingerprint"`
	Choices           []CompleteChunkChoice `json:"choices"`
	Usage             Usage                 `json:"usage,omitempty"`
}

type CompletionChunk struct {
	Id                string                `json:"id"`
	Object            string                `json:"object"`
	Created           int64                 `json:"created"`
	Choices           []CompleteChunkChoice `json:"choices"`
	Model             string                `json:"model"`
	SystemFingerprint string                `json:"system_fingerprint"`
	Usage             *Usage                `json:"usage,omitempty"`
}

type ToolCall struct {
	ID       string `json:"id"`
	Index    int    `json:"index"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type Model struct {
	Id      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

type Embedding struct {
	Object    string    `json:"object"`
	Embedding []float32 `json:"embedding"`
	Index     int       `json:"index"`
}

type ListCompletion struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

type EmbeddingList struct {
	Object string         `json:"object"`
	Data   []Embedding    `json:"data"`
	Model  string         `json:"model"`
	Usage  EmbeddingUsage `json:"usage,omitempty"`
}

type EmbeddingUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

func NewError(code int, message string) ErrorResponse {
	var etype string
	switch code {
	case http.StatusBadRequest:
		etype = "invalid_request_error"
	case http.StatusNotFound:
		etype = "not_found_error"
	default:
		etype = "api_error"
	}
	return ErrorResponse{Error{Type: etype, Message: message}}
}

func ToUsage(r api.ChatResponse) Usage {
	return Usage{
		PromptTokens:     r.PromptEvalCount,
		CompletionTokens: r.EvalCount,
		TotalTokens:      r.PromptEvalCount + r.EvalCount,
	}
}

func toolCallID() string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return "call_" + strings.ToLower(string(b))
}

func toToolCalls(tc []api.ToolCall) []ToolCall {
	toolCalls := make([]ToolCall, len(tc))
	for i, tc := range tc {
		toolCalls[i].ID = toolCallID()
		toolCalls[i].Type = "function"
		toolCalls[i].Function.Name = tc.Function.Name
		toolCalls[i].Index = tc.Function.Index
		args, err := json.Marshal(tc.Function.Arguments)
		if err != nil {
			slog.Error("could not marshall function arguments to json", "error", err)
			continue
		}
		toolCalls[i].Function.Arguments = string(args)
	}
	return toolCalls
}

func ToChatCompletion(id string, r api.ChatResponse) ChatCompletion {
	toolCalls := toToolCalls(r.Message.ToolCalls)
	return ChatCompletion{
		Id:                id,
		Object:            "chat.completion",
		Created:           r.CreatedAt.Unix(),
		Model:             r.Model,
		SystemFingerprint: "fp_goobla",
		Choices: []Choice{{
			Index:   0,
			Message: Message{Role: r.Message.Role, Content: r.Message.Content, ToolCalls: toolCalls},
			FinishReason: func(reason string) *string {
				if len(toolCalls) > 0 {
					reason = "tool_calls"
				}
				if len(reason) > 0 {
					return &reason
				}
				return nil
			}(r.DoneReason),
		}},
		Usage: ToUsage(r),
	}
}

func ToChunk(id string, r api.ChatResponse, toolCallSent bool) ChatCompletionChunk {
	toolCalls := toToolCalls(r.Message.ToolCalls)
	return ChatCompletionChunk{
		Id:                id,
		Object:            "chat.completion.chunk",
		Created:           time.Now().Unix(),
		Model:             r.Model,
		SystemFingerprint: "fp_goobla",
		Choices: []ChunkChoice{{
			Index: 0,
			Delta: Message{Role: "assistant", Content: r.Message.Content, ToolCalls: toolCalls},
			FinishReason: func(reason string) *string {
				if len(reason) > 0 {
					if toolCallSent {
						return &FinishReasonToolCalls
					}
					return &reason
				}
				return nil
			}(r.DoneReason),
		}},
	}
}

func ToUsageGenerate(r api.GenerateResponse) Usage {
	return Usage{
		PromptTokens:     r.PromptEvalCount,
		CompletionTokens: r.EvalCount,
		TotalTokens:      r.PromptEvalCount + r.EvalCount,
	}
}

func ToCompletion(id string, r api.GenerateResponse) Completion {
	return Completion{
		Id:                id,
		Object:            "text_completion",
		Created:           r.CreatedAt.Unix(),
		Model:             r.Model,
		SystemFingerprint: "fp_goobla",
		Choices: []CompleteChunkChoice{{
			Text:  r.Response,
			Index: 0,
			FinishReason: func(reason string) *string {
				if len(reason) > 0 {
					return &reason
				}
				return nil
			}(r.DoneReason),
		}},
		Usage: ToUsageGenerate(r),
	}
}

func ToCompleteChunk(id string, r api.GenerateResponse) CompletionChunk {
	return CompletionChunk{
		Id:                id,
		Object:            "text_completion",
		Created:           time.Now().Unix(),
		Model:             r.Model,
		SystemFingerprint: "fp_goobla",
		Choices: []CompleteChunkChoice{{
			Text:  r.Response,
			Index: 0,
			FinishReason: func(reason string) *string {
				if len(reason) > 0 {
					return &reason
				}
				return nil
			}(r.DoneReason),
		}},
	}
}

func ToListCompletion(r api.ListResponse) ListCompletion {
	var data []Model
	for _, m := range r.Models {
		data = append(data, Model{
			Id:      m.Name,
			Object:  "model",
			Created: m.ModifiedAt.Unix(),
			OwnedBy: model.ParseName(m.Name).Namespace,
		})
	}
	return ListCompletion{Object: "list", Data: data}
}

func ToEmbeddingList(model string, r api.EmbedResponse) EmbeddingList {
	if r.Embeddings != nil {
		var data []Embedding
		for i, e := range r.Embeddings {
			data = append(data, Embedding{Object: "embedding", Embedding: e, Index: i})
		}
		return EmbeddingList{
			Object: "list",
			Data:   data,
			Model:  model,
			Usage: EmbeddingUsage{
				PromptTokens: r.PromptEvalCount,
				TotalTokens:  r.PromptEvalCount,
			},
		}
	}
	return EmbeddingList{}
}

func ToModel(r api.ShowResponse, m string) Model {
	return Model{
		Id:      m,
		Object:  "model",
		Created: r.ModifiedAt.Unix(),
		OwnedBy: model.ParseName(m).Namespace,
	}
}

func FromChatRequest(r ChatCompletionRequest) (*api.ChatRequest, error) {
	var messages []api.Message
	for _, msg := range r.Messages {
		switch content := msg.Content.(type) {
		case string:
			messages = append(messages, api.Message{Role: msg.Role, Content: content})
		case []any:
			for _, c := range content {
				data, ok := c.(map[string]any)
				if !ok {
					return nil, errors.New("invalid message format")
				}
				switch data["type"] {
				case "text":
					text, ok := data["text"].(string)
					if !ok {
						return nil, errors.New("invalid message format")
					}
					messages = append(messages, api.Message{Role: msg.Role, Content: text})
				case "image_url":
					var url string
					if urlMap, ok := data["image_url"].(map[string]any); ok {
						if url, ok = urlMap["url"].(string); !ok {
							return nil, errors.New("invalid message format")
						}
					} else {
						if url, ok = data["image_url"].(string); !ok {
							return nil, errors.New("invalid message format")
						}
					}
					types := []string{"jpeg", "jpg", "png"}
					valid := false
					for _, t := range types {
						prefix := "data:image/" + t + ";base64,"
						if strings.HasPrefix(url, prefix) {
							url = strings.TrimPrefix(url, prefix)
							valid = true
							break
						}
					}
					if !valid {
						return nil, errors.New("invalid image input")
					}
					img, err := base64.StdEncoding.DecodeString(url)
					if err != nil {
						return nil, errors.New("invalid message format")
					}
					messages = append(messages, api.Message{Role: msg.Role, Images: []api.ImageData{img}})
				default:
					return nil, errors.New("invalid message format")
				}
			}
		default:
			if msg.ToolCalls == nil {
				return nil, fmt.Errorf("invalid message content type: %T", content)
			}
			toolCalls := make([]api.ToolCall, len(msg.ToolCalls))
			for i, tc := range msg.ToolCalls {
				toolCalls[i].Function.Name = tc.Function.Name
				err := json.Unmarshal([]byte(tc.Function.Arguments), &toolCalls[i].Function.Arguments)
				if err != nil {
					return nil, errors.New("invalid tool call arguments")
				}
			}
			messages = append(messages, api.Message{Role: msg.Role, ToolCalls: toolCalls})
		}
	}
	options := make(map[string]any)
	switch stop := r.Stop.(type) {
	case string:
		options["stop"] = []string{stop}
	case []any:
		var stops []string
		for _, s := range stop {
			if str, ok := s.(string); ok {
				stops = append(stops, str)
			}
		}
		options["stop"] = stops
	}
	if r.MaxTokens != nil {
		options["num_predict"] = *r.MaxTokens
	}
	if r.Temperature != nil {
		options["temperature"] = *r.Temperature
	} else {
		options["temperature"] = 1.0
	}
	if r.Seed != nil {
		options["seed"] = *r.Seed
	}
	if r.FrequencyPenalty != nil {
		options["frequency_penalty"] = *r.FrequencyPenalty
	}
	if r.PresencePenalty != nil {
		options["presence_penalty"] = *r.PresencePenalty
	}
	if r.TopP != nil {
		options["top_p"] = *r.TopP
	} else {
		options["top_p"] = 1.0
	}
	var format json.RawMessage
	if r.ResponseFormat != nil {
		switch strings.ToLower(strings.TrimSpace(r.ResponseFormat.Type)) {
		case "json_object":
			format = json.RawMessage(`"json"`)
		case "json_schema":
			if r.ResponseFormat.JsonSchema != nil {
				format = r.ResponseFormat.JsonSchema.Schema
			}
		}
	}
	return &api.ChatRequest{
		Model:    r.Model,
		Messages: messages,
		Format:   format,
		Options:  options,
		Stream:   &r.Stream,
		Tools:    r.Tools,
	}, nil
}

func FromCompleteRequest(r CompletionRequest) (api.GenerateRequest, error) {
	options := make(map[string]any)
	switch stop := r.Stop.(type) {
	case string:
		options["stop"] = []string{stop}
	case []any:
		var stops []string
		for _, s := range stop {
			if str, ok := s.(string); ok {
				stops = append(stops, str)
			} else {
				return api.GenerateRequest{}, fmt.Errorf("invalid type for 'stop' field: %T", s)
			}
		}
		options["stop"] = stops
	}
	if r.MaxTokens != nil {
		options["num_predict"] = *r.MaxTokens
	}
	if r.Temperature != nil {
		options["temperature"] = *r.Temperature
	} else {
		options["temperature"] = 1.0
	}
	if r.Seed != nil {
		options["seed"] = *r.Seed
	}
	options["frequency_penalty"] = r.FrequencyPenalty
	options["presence_penalty"] = r.PresencePenalty
	if r.TopP != 0.0 {
		options["top_p"] = r.TopP
	} else {
		options["top_p"] = 1.0
	}
	var prompt string
	var context []int
	switch p := r.Prompt.(type) {
	case string:
		prompt = p
	case []any:
		if len(p) == 0 {
			prompt = ""
		} else {
			switch first := p[0].(type) {
			case string:
				var sb strings.Builder
				for _, v := range p {
					s, ok := v.(string)
					if !ok {
						return api.GenerateRequest{}, fmt.Errorf("invalid type for 'prompt' field: %T", v)
					}
					sb.WriteString(s)
				}
				prompt = sb.String()
			case float64:
				for _, v := range p {
					num, ok := v.(float64)
					if !ok {
						return api.GenerateRequest{}, fmt.Errorf("invalid type for 'prompt' field: %T", v)
					}
					context = append(context, int(num))
				}
			case []any:
				var tokens []any = first
				for _, v := range tokens {
					num, ok := v.(float64)
					if !ok {
						return api.GenerateRequest{}, fmt.Errorf("invalid type for 'prompt' field: %T", v)
					}
					context = append(context, int(num))
				}
			default:
				return api.GenerateRequest{}, fmt.Errorf("invalid type for 'prompt' field: %T", first)
			}
		}
	default:
		return api.GenerateRequest{}, fmt.Errorf("invalid type for 'prompt' field: %T", r.Prompt)
	}
	return api.GenerateRequest{
		Model:   r.Model,
		Prompt:  prompt,
		Context: context,
		Options: options,
		Stream:  &r.Stream,
		Suffix:  r.Suffix,
	}, nil
}
