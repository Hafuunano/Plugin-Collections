// Package pluginagent provides an LLM chat plugin: reads soul/PERSONA as system prompt,
// per-user session with max 10 context turns; when exceeded, summarizes and starts a new round.
package pluginagent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Hafuunano/Core-SkillAction/types"
	"github.com/Hafuunano/Protocol-ConvertTool/protocol"
)

const (
	maxContextTurns = 10
	personaPathEnv  = "SOUL_PERSONA_PATH"
	defaultPersona  = "soul/PERSONA.md"
	sessionDataDir  = "data/llm-playground/sessions"
	envLLMURL       = "LLM_API_URL"
	envLLMKey       = "LLM_API_KEY"
	envLLMModel     = "LLM_MODEL"
	defaultURL      = "https://api.openai.com/v1"
	defaultModel    = "gpt-3.5-turbo"
)

// llmConfig holds the configured URL, API key, and model (set from env at init or via /setLLM* commands).
var llmConfig struct {
	URL   string
	Key   string
	Model string
}
var llmConfigMu sync.RWMutex

var (
	systemPrompt string
	userSessions = make(map[string]*userSession)
	sessionsMu   sync.RWMutex
)

// Meta and registration (required: use WithMeta(Meta) then chain).
var Meta = types.NewPluginEngine("plugin-agent-001", "plugin-agent", "skill", true)
var p = protocol.Engine.WithMeta(Meta)

type userSession struct {
	Messages      []chatMessage
	LatestSummary string
	Mu            sync.Mutex
}

type sessionFile struct {
	Messages      []chatMessage `json:"messages"`
	LatestSummary string        `json:"latest_summary"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatReq struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatResp struct {
	Choices []struct {
		Message struct {
			Content json.RawMessage `json:"content"`
			Role    string         `json:"role"`
		} `json:"message"`
	} `json:"choices"`
}

func init() {
	loadPersona()
	llmConfig.URL = os.Getenv(envLLMURL)
	if llmConfig.URL == "" {
		llmConfig.URL = defaultURL
	}
	llmConfig.Key = os.Getenv(envLLMKey)
	llmConfig.Model = os.Getenv(envLLMModel)
	if llmConfig.Model == "" {
		llmConfig.Model = defaultModel
	}
	// Super admin only: /setLLMUrl, /setLLMKey, /setLLMModel
	p.OnMessage().IsOnlySuperAdmin().Func(handleSuperAdminCommand)
	// When @bot or reply: treat as chat (unless already handled as command)
	p.OnMessage().IsOnlyToMe().Func(handleOnlyToMe)
}

// getCommandArg returns the rest of the message after the command (prefix + command name).
func getCommandArg(ctx protocol.Context, cmd string) string {
	raw := strings.TrimSpace(ctx.PlainText())
	for _, prefix := range []string{"/", "!", "！", "."} {
		if strings.HasPrefix(raw, prefix+cmd) {
			return strings.TrimSpace(raw[len(prefix+cmd):])
		}
	}
	if strings.HasPrefix(raw, cmd) {
		return strings.TrimSpace(raw[len(cmd):])
	}
	return ""
}

// isSetLLMCommand returns true if plain text is one of /setLLMUrl, /setLLMKey, /setLLMModel.
func isSetLLMCommand(text string) bool {
	text = strings.TrimSpace(text)
	for _, prefix := range []string{"/", "!", "！", "."} {
		if strings.HasPrefix(text, prefix+"setLLMUrl") || strings.HasPrefix(text, prefix+"setLLMKey") || strings.HasPrefix(text, prefix+"setLLMModel") {
			return true
		}
	}
	return strings.HasPrefix(text, "setLLMUrl") || strings.HasPrefix(text, "setLLMKey") || strings.HasPrefix(text, "setLLMModel")
}

// hasCommandPrefix returns true if plain text starts with prefix+cmd (e.g. /setLLMUrl).
func hasCommandPrefix(text, cmd string) bool {
	text = strings.TrimSpace(text)
	for _, prefix := range []string{"/", "!", "！", "."} {
		if strings.HasPrefix(text, prefix+cmd) {
			return true
		}
	}
	return strings.HasPrefix(text, cmd)
}

func handleSuperAdminCommand(ctx protocol.Context) {
	raw := strings.TrimSpace(ctx.PlainText())
	val := getCommandArg(ctx, "setLLMUrl")
	if hasCommandPrefix(raw, "setLLMUrl") {
		if val == "" {
			_ = ctx.Reply(protocol.Message{
				protocol.Segment{Type: protocol.SegmentTypeText, Data: map[string]any{"text": "用法: /setLLMUrl <URL>，例如 /setLLMUrl https://api.openai.com/v1"}},
			})
			return
		}
		setURL := strings.TrimSuffix(strings.TrimSpace(val), "/")
		llmConfigMu.Lock()
		llmConfig.URL = setURL
		llmConfigMu.Unlock()
		_ = ctx.Reply(protocol.Message{
			protocol.Segment{Type: protocol.SegmentTypeText, Data: map[string]any{"text": "已设置 LLM URL: " + setURL}},
		})
		return
	}
	val = getCommandArg(ctx, "setLLMKey")
	if hasCommandPrefix(raw, "setLLMKey") {
		if val == "" {
			_ = ctx.Reply(protocol.Message{
				protocol.Segment{Type: protocol.SegmentTypeText, Data: map[string]any{"text": "用法: /setLLMKey <API Key>"}},
			})
			return
		}
		llmConfigMu.Lock()
		llmConfig.Key = strings.TrimSpace(val)
		llmConfigMu.Unlock()
		_ = ctx.Reply(protocol.Message{
			protocol.Segment{Type: protocol.SegmentTypeText, Data: map[string]any{"text": "已设置 LLM API Key（已隐藏）"}},
		})
		return
	}
	val = getCommandArg(ctx, "setLLMModel")
	if hasCommandPrefix(raw, "setLLMModel") {
		if val == "" {
			_ = ctx.Reply(protocol.Message{
				protocol.Segment{Type: protocol.SegmentTypeText, Data: map[string]any{"text": "用法: /setLLMModel <模型名>，例如 /setLLMModel gpt-3.5-turbo"}},
			})
			return
		}
		setModel := strings.TrimSpace(val)
		llmConfigMu.Lock()
		llmConfig.Model = setModel
		llmConfigMu.Unlock()
		_ = ctx.Reply(protocol.Message{
			protocol.Segment{Type: protocol.SegmentTypeText, Data: map[string]any{"text": "已设置 LLM 模型: " + setModel}},
		})
		return
	}
}

func handleOnlyToMe(ctx protocol.Context) {
	text := strings.TrimSpace(ctx.PlainText())
	if text == "" {
		return
	}
	// Do not treat setLLM* as chat (super admin command already handled in handleSuperAdminCommand when super admin; for non-super admin skip)
	if isSetLLMCommand(text) {
		return
	}
	handleChat(ctx)
}

func loadPersona() {
	path := os.Getenv(personaPathEnv)
	if path == "" {
		path = defaultPersona
	}
	abs := path
	if !filepath.IsAbs(path) {
		if cwd, err := os.Getwd(); err == nil {
			abs = filepath.Join(cwd, path)
		}
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		systemPrompt = "You are a helpful assistant."
		return
	}
	persona := strings.TrimSpace(string(data))
	systemPrompt = persona + "\n\n我希望你扮演我所描述的人物"
}

func sessionKey(ctx protocol.Context) string {
	return ctx.UserID()
}

func sessionFilePath(key string) string {
	dir := sessionDataDir
	if !filepath.IsAbs(dir) {
		if cwd, err := os.Getwd(); err == nil {
			dir = filepath.Join(cwd, dir)
		}
	}
	return filepath.Join(dir, key+".json")
}

func loadSession(key string) (*userSession, bool) {
	path := sessionFilePath(key)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var f sessionFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, false
	}
	s := &userSession{
		Messages:      f.Messages,
		LatestSummary: f.LatestSummary,
	}
	if s.Messages == nil {
		s.Messages = make([]chatMessage, 0)
	}
	return s, true
}

func saveSession(key string, s *userSession) {
	messages := make([]chatMessage, len(s.Messages))
	copy(messages, s.Messages)
	path := sessionFilePath(key)
	dir := filepath.Dir(path)
	_ = os.MkdirAll(dir, 0755)
	f := sessionFile{Messages: messages, LatestSummary: s.LatestSummary}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0644)
}

func getOrCreateSession(key string) *userSession {
	sessionsMu.RLock()
	s, ok := userSessions[key]
	sessionsMu.RUnlock()
	if ok {
		return s
	}
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	if s, ok = userSessions[key]; ok {
		return s
	}
	if s, ok = loadSession(key); ok {
		userSessions[key] = s
		return s
	}
	s = &userSession{Messages: make([]chatMessage, 0)}
	userSessions[key] = s
	return s
}

// cqAtRegex matches CQ code like [CQ:at,qq=123456] or [CQ:at,qq=123456,text=@nick]
var cqAtRegex = regexp.MustCompile(`\[CQ:at,qq=\d+(?:,[^\]]*)?\]`)

// stripCQAt removes CQ at segments from plain text so only real content is sent to LLM.
func stripCQAt(s string) string {
	return strings.TrimSpace(cqAtRegex.ReplaceAllString(s, ""))
}

func handleChat(ctx protocol.Context) {
	raw := ctx.PlainText()
	text := stripCQAt(raw)
	if text == "" {
		return
	}
	key := sessionKey(ctx)
	s := getOrCreateSession(key)
	s.Mu.Lock()
	defer s.Mu.Unlock()

	s.Messages = append(s.Messages, chatMessage{Role: "user", Content: text})

	messages := buildMessages(s)
	if len(s.Messages) > maxContextTurns*2 {
		summary, err := summarizeConversation(messages)
		if err == nil && summary != "" {
			s.LatestSummary = summary
			s.Messages = []chatMessage{
				{Role: "user", Content: "[Previous conversation summary]\n" + summary},
				{Role: "assistant", Content: "好的，我记住了之前的对话要点，我们继续聊吧～"},
			}
			s.Messages = append(s.Messages, chatMessage{Role: "user", Content: text})
			messages = buildMessages(s)
			saveSession(key, s)
		} else {
			trimToLastNTurns(s, maxContextTurns)
			messages = buildMessages(s)
		}
	}

	reply, err := callLLM(messages)
	if err != nil {
		log.Printf("[plugin-agent] callLLM error: %v", err)
		_ = ctx.Reply(protocol.Message{
			protocol.Segment{Type: protocol.SegmentTypeText, Data: map[string]any{"text": "呜…出错了: " + err.Error()}},
		})
		return
	}
	s.Messages = append(s.Messages, chatMessage{Role: "assistant", Content: reply})
	saveSession(key, s)
	_ = ctx.Reply(protocol.Message{
		protocol.Segment{Type: protocol.SegmentTypeText, Data: map[string]any{"text": reply}},
	})
}

func buildMessages(s *userSession) []chatMessage {
	out := make([]chatMessage, 0, len(s.Messages)+1)
	out = append(out, chatMessage{Role: "system", Content: systemPrompt})
	out = append(out, s.Messages...)
	return out
}

func trimToLastNTurns(s *userSession, n int) {
	total := 2 * n
	if len(s.Messages) <= total {
		return
	}
	s.Messages = s.Messages[len(s.Messages)-total:]
}

func summarizeConversation(messages []chatMessage) (string, error) {
	if len(messages) <= 2 {
		return "", nil
	}
	toSum := messages[1 : len(messages)-1]
	sumReq := make([]chatMessage, 0, len(toSum)+1)
	sumReq = append(sumReq, chatMessage{
		Role:    "system",
		Content: "Please summarize the following conversation in 1-3 short paragraphs in the same language, preserving key facts and tone. Output only the summary.",
	})
	sumReq = append(sumReq, toSum...)
	return callLLM(sumReq)
}

func callLLM(messages []chatMessage) (string, error) {
	llmConfigMu.RLock()
	baseURL, apiKey, model := llmConfig.URL, llmConfig.Key, llmConfig.Model
	llmConfigMu.RUnlock()
	url := strings.TrimSuffix(baseURL, "/") + "/chat/completions"
	body := chatReq{Model: model, Messages: messages}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API %d: %s", resp.StatusCode, string(data))
	}
	var r chatResp
	if err := json.Unmarshal(data, &r); err != nil {
		return "", err
	}
	if len(r.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}
	return extractContent(r.Choices[0].Message.Content)
}

// extractContent supports content as string or array of {type, text} (OpenAI/Moonshot compatible).
func extractContent(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "", fmt.Errorf("empty content")
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strings.TrimSpace(s), nil
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &parts); err != nil {
		return "", fmt.Errorf("content neither string nor array: %w", err)
	}
	var b strings.Builder
	for _, p := range parts {
		if p.Type == "text" {
			b.WriteString(p.Text)
		}
	}
	out := strings.TrimSpace(b.String())
	if out == "" {
		return "", fmt.Errorf("no text in content")
	}
	return out, nil
}
