package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"

	"hint/internal/config"
	"hint/pkg/sysinfo"
)

// Client wraps a compatible OpenAI chat/completions endpoint.
type Client struct {
	cfg  config.Config
	http *resty.Client
}

type Request struct {
	UserPrompt      string
	Env             sysinfo.Env
	Retry           bool
	PreviousCommand string
}

type AgentObservation struct {
	Thought  string `json:"thought"`
	Command  string `json:"command"`
	Output   string `json:"output"`
	ExitCode int    `json:"exit_code"`
}

type AgentRequest struct {
	Goal  string
	Env   sysinfo.Env
	Steps []AgentObservation
}

type AgentAction struct {
	Type    string `json:"type"`
	Thought string `json:"thought"`
	Command string `json:"command,omitempty"`
	Final   string `json:"final,omitempty"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Stream   bool          `json:"stream,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func NewClient(cfg config.Config) *Client {
	http := resty.New().
		SetTimeout(60*time.Second).
		SetHeader("Authorization", "Bearer "+cfg.APIKey).
		SetHeader("Content-Type", "application/json")
	return &Client{cfg: cfg, http: http}
}

func (c *Client) SuggestCommand(ctx context.Context, req Request) (string, error) {
	userPrompt := strings.TrimSpace(req.UserPrompt)
	if req.Retry {
		if prev := strings.TrimSpace(req.PreviousCommand); prev != "" {
			userPrompt += fmt.Sprintf("\n上一次建议命令是：%s\n这条命令不满意，请换一种实现方式。", prev)
		} else {
			userPrompt += "\n上一个建议不满意，请换一种实现方式。"
		}
	}

	body := chatRequest{
		Model: c.cfg.Model,
		Messages: []chatMessage{
			{
				Role: "system",
				Content: fmt.Sprintf("你是命令行助手。仅输出一个可直接执行的原始命令，不要解释、不要 Markdown、不要代码块。\n环境信息: GOOS=%s, Distro=%s, Shell=%s, PWD=%s",
					req.Env.GOOS,
					req.Env.Distro,
					req.Env.Shell,
					req.Env.PWD,
				),
			},
			{
				Role:    "user",
				Content: userPrompt,
			},
		},
	}

	content, err := c.callChat(ctx, body)
	if err != nil {
		return "", err
	}
	cmd := sanitize(content)
	if cmd == "" {
		return "", fmt.Errorf("empty command returned by ai")
	}
	return cmd, nil
}

func (c *Client) NextAgentAction(ctx context.Context, req AgentRequest) (AgentAction, error) {
	body, err := c.buildAgentChatRequest(req, false)
	if err != nil {
		return AgentAction{}, err
	}

	content, err := c.callChat(ctx, body)
	if err != nil {
		return AgentAction{}, err
	}
	action, err := parseAgentActionText(content)
	if err != nil {
		return AgentAction{}, err
	}
	return action, nil
}

func (c *Client) NextAgentActionStream(ctx context.Context, req AgentRequest, onDelta func(string)) (AgentAction, error) {
	body, err := c.buildAgentChatRequest(req, true)
	if err != nil {
		return AgentAction{}, err
	}
	content, err := c.callChatStream(ctx, body, onDelta)
	if err != nil {
		return AgentAction{}, err
	}
	action, err := parseAgentActionText(content)
	if err != nil {
		return AgentAction{}, err
	}
	return action, nil
}

func (c *Client) callChat(ctx context.Context, body chatRequest) (string, error) {
	endpoint := chatCompletionsEndpoint(c.cfg.BaseURL)

	var out chatResponse
	resp, err := c.http.R().
		SetContext(ctx).
		SetBody(body).
		SetResult(&out).
		Post(endpoint)
	if err != nil {
		return "", fmt.Errorf("request ai failed: %w", err)
	}
	if resp.IsError() {
		return "", fmt.Errorf("ai response status %d: %s", resp.StatusCode(), strings.TrimSpace(resp.String()))
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("ai response has no choices")
	}
	return out.Choices[0].Message.Content, nil
}

func (c *Client) callChatStream(ctx context.Context, body chatRequest, onDelta func(string)) (string, error) {
	endpoint := chatCompletionsEndpoint(c.cfg.BaseURL)
	payload, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal stream request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("create stream request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.http.GetClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("request ai stream failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("ai stream response status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1024), 1024*1024)

	var out strings.Builder
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" {
			continue
		}
		if data == "[DONE]" {
			break
		}

		var event struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}
		if len(event.Choices) == 0 {
			continue
		}
		part := event.Choices[0].Delta.Content
		if part == "" {
			continue
		}
		out.WriteString(part)
		if onDelta != nil {
			onDelta(part)
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("read ai stream: %w", err)
	}
	return out.String(), nil
}

func (c *Client) buildAgentChatRequest(req AgentRequest, stream bool) (chatRequest, error) {
	stepsJSON, err := json.Marshal(req.Steps)
	if err != nil {
		return chatRequest{}, fmt.Errorf("marshal agent steps: %w", err)
	}

	return chatRequest{
		Model:  c.cfg.Model,
		Stream: stream,
		Messages: []chatMessage{
			{
				Role: "system",
				Content: fmt.Sprintf("你是运维 Agent。你会分步完成目标，必要时执行 shell 命令并根据输出继续决策。\n"+
					"你必须使用以下纯文本格式输出，不要 Markdown、不要代码块。\n"+
					"TYPE: command 或 final\n"+
					"THOUGHT: 简短思考（1-2句）\n"+
					"COMMAND: 仅当 TYPE=command 时提供一条可执行命令\n"+
					"FINAL: 仅当 TYPE=final 时提供结果总结\n"+
					"规则：\n"+
					"1) 未完成时返回 TYPE=command。\n"+
					"2) 完成或无法继续时返回 TYPE=final。\n"+
					"3) command 必须是单条可执行命令。\n"+
					"环境信息: GOOS=%s, Distro=%s, Shell=%s, PWD=%s",
					req.Env.GOOS,
					req.Env.Distro,
					req.Env.Shell,
					req.Env.PWD,
				),
			},
			{
				Role: "user",
				Content: fmt.Sprintf("用户目标：%s\n\n历史执行记录(JSON数组)：%s",
					strings.TrimSpace(req.Goal),
					string(stepsJSON),
				),
			},
		},
	}, nil
}

func chatCompletionsEndpoint(baseURL string) string {
	base := strings.TrimSpace(strings.TrimRight(baseURL, "/"))
	lower := strings.ToLower(base)
	if strings.HasSuffix(lower, "/chat/completions") {
		return base
	}
	return base + "/chat/completions"
}

func sanitize(v string) string {
	s := strings.TrimSpace(v)
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "`")
	s = strings.TrimSpace(s)

	lines := strings.Split(s, "\n")
	if len(lines) == 0 {
		return ""
	}
	return strings.TrimSpace(lines[0])
}

func parseAgentActionText(content string) (AgentAction, error) {
	s := strings.TrimSpace(content)
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)

	var jsonAction AgentAction
	if err := json.Unmarshal([]byte(s), &jsonAction); err == nil {
		if action, ok := normalizeAgentAction(jsonAction); ok {
			return action, nil
		}
	}

	var action AgentAction
	current := ""

	lines := strings.Split(s, "\n")
	for _, raw := range lines {
		line := strings.TrimRight(raw, "\r")
		trimmed := strings.TrimSpace(line)
		if key, value, ok := splitAgentField(trimmed); ok {
			switch key {
			case "type":
				action.Type = value
				current = ""
			case "thought":
				action.Thought = value
				current = "thought"
			case "command":
				action.Command = value
				current = "command"
			case "final":
				action.Final = value
				current = "final"
			}
			continue
		}
		switch {
		default:
			if trimmed == "" {
				continue
			}
			switch current {
			case "thought":
				if action.Thought == "" {
					action.Thought = trimmed
				} else {
					action.Thought += " " + trimmed
				}
			case "command":
				if action.Command == "" {
					action.Command = trimmed
				} else {
					action.Command += " " + trimmed
				}
			case "final":
				if action.Final == "" {
					action.Final = trimmed
				} else {
					action.Final += " " + trimmed
				}
			}
		}
	}

	if normalized, ok := normalizeAgentAction(action); ok {
		return normalized, nil
	}
	return AgentAction{}, fmt.Errorf("unsupported agent action type: %s", strings.TrimSpace(action.Type))
}

func splitAgentField(line string) (key, value string, ok bool) {
	if line == "" {
		return "", "", false
	}
	for _, sep := range []string{":", "：", "="} {
		if idx := strings.Index(line, sep); idx > 0 {
			key = strings.ToLower(strings.TrimSpace(line[:idx]))
			value = strings.TrimSpace(line[idx+len(sep):])
			switch key {
			case "type", "thought", "command", "final":
				return key, value, true
			default:
				return "", "", false
			}
		}
	}
	return "", "", false
}

func normalizeAgentAction(action AgentAction) (AgentAction, bool) {
	action.Type = normalizeActionType(action.Type)
	action.Thought = strings.TrimSpace(action.Thought)
	action.Command = strings.TrimSpace(action.Command)
	action.Final = strings.TrimSpace(action.Final)

	if action.Type == "" {
		switch {
		case action.Command != "" && action.Final == "":
			action.Type = "command"
		case action.Final != "" && action.Command == "":
			action.Type = "final"
		}
	}

	switch action.Type {
	case "command":
		action.Command = sanitize(action.Command)
		if action.Command == "" {
			return AgentAction{}, false
		}
		return action, true
	case "final":
		if action.Final == "" {
			return AgentAction{}, false
		}
		return action, true
	default:
		return AgentAction{}, false
	}
}

func normalizeActionType(v string) string {
	s := strings.ToLower(strings.TrimSpace(v))
	s = strings.Trim(s, "`\"'[](){}.,;:：-")
	switch s {
	case "command", "cmd", "shell", "run":
		return "command"
	case "final", "done", "answer", "result":
		return "final"
	default:
		return s
	}
}
