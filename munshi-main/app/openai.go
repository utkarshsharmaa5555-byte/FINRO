package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"
)

const chatModel = "gpt-4.1-mini"

var aiClient = &http.Client{Timeout: 90 * time.Second}

type ChatMsg struct {
	Role         string         `json:"role"`
	Content      any            `json:"content"` // string or []map (vision parts)
	ToolCalls    []ToolCall     `json:"tool_calls,omitempty"`
	ToolCallID   string         `json:"tool_call_id,omitempty"`
	ExtraContent map[string]any `json:"extra_content,omitempty"`
}

type ToolCall struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	ExtraContent map[string]any `json:"extra_content,omitempty"`
	Function     struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type openaiError struct {
	Message string `json:"message"`
}

type chatResp struct {
	Choices []struct {
		Message ChatMsg `json:"message"`
	} `json:"choices"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
	Error *openaiError `json:"error"`
}

func getAPIKey() string {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		key = os.Getenv("OPEN_API_KEY") // name used in the Vercel project
	}
	return key
}

func getBaseURL(key string) string {
	if u := os.Getenv("OPENAI_BASE_URL"); u != "" {
		return strings.TrimSuffix(u, "/")
	}
	if strings.HasPrefix(key, "AQ.") || strings.HasPrefix(key, "AIza") {
		return "https://generativelanguage.googleapis.com/v1beta/openai"
	}
	return "https://api.openai.com/v1"
}

func getChatModels(key string) []string {
	if m := os.Getenv("OPENAI_MODEL"); m != "" {
		return []string{m}
	}
	if strings.HasPrefix(key, "AQ.") || strings.HasPrefix(key, "AIza") {
		return []string{"gemini-flash-latest", "gemini-3.5-flash-lite", "gemini-3.6-flash"}
	}
	return []string{chatModel}
}

func openaiDo(ctx context.Context, path, contentType string, body io.Reader) (*http.Response, error) {
	key := getAPIKey()
	if key == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY not set")
	}
	baseURL := getBaseURL(key)
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", contentType)
	return aiClient.Do(req)
}

// chat calls chat completions. schema (optional) forces strict JSON output.
func chat(ctx context.Context, kind string, msgs []ChatMsg, tools []map[string]any, schema map[string]any) (ChatMsg, error) {
	start := time.Now()
	key := getAPIKey()
	models := getChatModels(key)
	var lastErr error

	for _, model := range models {
		req := map[string]any{"model": model, "messages": msgs, "temperature": 0.3}
		if len(tools) > 0 {
			req["tools"] = tools
		}
		if schema != nil {
			req["response_format"] = map[string]any{"type": "json_schema", "json_schema": map[string]any{"name": kind, "strict": true, "schema": schema}}
		}
		b, _ := json.Marshal(req)
		resp, err := openaiDo(ctx, "/chat/completions", "application/json", bytes.NewReader(b))
		if err != nil {
			lastErr = err
			continue
		}
		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}
		var out chatResp
		trimmed := bytes.TrimSpace(respBody)
		if len(trimmed) > 0 && trimmed[0] == '[' {
			var errArr []struct {
				Error *openaiError `json:"error"`
			}
			if json.Unmarshal(trimmed, &errArr) == nil && len(errArr) > 0 && errArr[0].Error != nil {
				lastErr = fmt.Errorf("ai: %s", errArr[0].Error.Message)
				continue
			}
			var list []chatResp
			if err := json.Unmarshal(trimmed, &list); err != nil {
				lastErr = err
				continue
			}
			if len(list) > 0 {
				out = list[0]
			}
		} else {
			if err := json.Unmarshal(trimmed, &out); err != nil {
				lastErr = err
				continue
			}
		}
		if out.Error != nil || len(out.Choices) == 0 {
			if out.Error != nil {
				lastErr = fmt.Errorf("openai: %s", out.Error.Message)
			} else {
				lastErr = fmt.Errorf("openai: empty response (status %d)", resp.StatusCode)
			}
			continue
		}
		logAI(kind, start, out.Usage.TotalTokens, true)
		return out.Choices[0].Message, nil
	}
	logAI(kind, start, 0, false)
	return ChatMsg{}, lastErr
}

// chatJSON runs a strict-schema completion and decodes into v. user may be a string or vision parts.
func chatJSON(ctx context.Context, kind, system string, user any, schema map[string]any, v any) error {
	m, err := chat(ctx, kind, []ChatMsg{{Role: "system", Content: system}, {Role: "user", Content: user}}, nil, schema)
	if err != nil {
		return err
	}
	out, _ := m.Content.(string)
	// Postgres jsonb rejects NUL and models occasionally emit one, which would otherwise fail an
	// insert far from here; strip it at this shared boundary so every AI feature is covered.
	out = strings.ReplaceAll(out, "\\u0000", "")
	out = strings.ReplaceAll(out, "\x00", "")
	return json.Unmarshal([]byte(out), v)
}

// transcribe tries the newer transcription model, then whisper-1 (keys differ in model access).
func transcribe(ctx context.Context, audio io.Reader, filename, lang string) (string, error) {
	data, err := io.ReadAll(audio)
	if err != nil {
		return "", err
	}
	var lastErr error
	for _, model := range []string{"gpt-4o-mini-transcribe", "whisper-1"} {
		start := time.Now()
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		mw.WriteField("model", model)
		if lang != "" {
			mw.WriteField("language", lang)
		}
		fw, _ := mw.CreateFormFile("file", filename)
		fw.Write(data)
		mw.Close()
		resp, err := openaiDo(ctx, "/audio/transcriptions", mw.FormDataContentType(), &buf)
		if err != nil {
			logAI("transcribe", start, 0, false)
			return "", err
		}
		var out struct {
			Text string `json:"text"`
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		resp.Body.Close()
		if resp.StatusCode == 200 && json.Unmarshal(body, &out) == nil {
			logAI("transcribe", start, 0, true)
			return out.Text, nil
		}
		logAI("transcribe", start, 0, false)
		lastErr = fmt.Errorf("transcribe %s failed (%d): %s", model, resp.StatusCode, truncate(string(body), 300))
		log.Print(lastErr)
	}
	return "", lastErr
}

// speak tries the expressive TTS model first, then the classic one (keys differ in model access).
func speak(ctx context.Context, text string) (io.ReadCloser, error) {
	var lastErr error
	for _, model := range []string{"gpt-4o-mini-tts", "tts-1"} {
		start := time.Now()
		req := map[string]any{"model": model, "voice": "sage", "input": text, "response_format": "mp3"}
		if model == "gpt-4o-mini-tts" {
			req["instructions"] = "Speak warmly and slowly, like a trusted neighbourhood accountant explaining things to a small shopkeeper."
		} else {
			req["voice"] = "nova"
		}
		b, _ := json.Marshal(req)
		resp, err := openaiDo(ctx, "/audio/speech", "application/json", bytes.NewReader(b))
		if err != nil {
			logAI("tts", start, 0, false)
			return nil, err
		}
		if resp.StatusCode == 200 {
			logAI("tts", start, 0, true)
			return resp.Body, nil
		}
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 500))
		resp.Body.Close()
		logAI("tts", start, 0, false)
		lastErr = fmt.Errorf("tts %s failed (%d): %s", model, resp.StatusCode, msg)
		log.Print(lastErr)
	}
	return nil, lastErr
}

func logAI(kind string, start time.Time, tokens int, ok bool) {
	db, err := DB()
	if err != nil {
		return
	}
	db.Exec(context.Background(), `INSERT INTO ai_calls(kind, ms, tokens, ok) VALUES($1,$2,$3,$4)`,
		kind, time.Since(start).Milliseconds(), tokens, ok)
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// Strict-schema helpers keep schema literals short. Strict mode requires every property listed as required.
func obj(props map[string]any) map[string]any {
	req := make([]string, 0, len(props))
	for k := range props {
		req = append(req, k)
	}
	return map[string]any{"type": "object", "properties": props, "required": req, "additionalProperties": false}
}
func str(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }
func arr(items map[string]any) map[string]any {
	return map[string]any{"type": "array", "items": items}
}
func enum(vals ...string) map[string]any { return map[string]any{"type": "string", "enum": vals} }
func integer(desc string) map[string]any {
	return map[string]any{"type": "integer", "description": desc}
}
