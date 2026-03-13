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

type AgentTurn struct {
	Goal    string             `json:"goal"`
	Outcome string             `json:"outcome"`
	Steps   []AgentObservation `json:"steps,omitempty"`
}

type AgentRequest struct {
	Goal    string
	Env     sysinfo.Env
	Steps   []AgentObservation
	History []AgentTurn
}

type AgentAction struct {
	Type    string `json:"type"`
	Thought string `json:"thought"`
	Command string `json:"command,omitempty"`
	Final   string `json:"final,omitempty"`
}

type AgentStreamEvent struct {
	Event   string      `json:"event"`
	Summary string      `json:"summary,omitempty"`
	Action  AgentAction `json:"action,omitempty"`
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

func (c *Client) NextAgentActionStream(ctx context.Context, req AgentRequest, onEvent func(AgentStreamEvent)) (AgentAction, error) {
	body, err := c.buildAgentChatRequest(req, true)
	if err != nil {
		return AgentAction{}, err
	}
	return c.callAgentEventStream(ctx, body, onEvent)
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

func (c *Client) callAgentEventStream(ctx context.Context, body chatRequest, onEvent func(AgentStreamEvent)) (AgentAction, error) {
	endpoint := chatCompletionsEndpoint(c.cfg.BaseURL)
	payload, err := json.Marshal(body)
	if err != nil {
		return AgentAction{}, fmt.Errorf("marshal stream request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return AgentAction{}, fmt.Errorf("create stream request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.http.GetClient().Do(req)
	if err != nil {
		return AgentAction{}, fmt.Errorf("request ai stream failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return AgentAction{}, fmt.Errorf("ai stream response status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1024), 1024*1024)

	var out strings.Builder
	var pending string
	var lastSummary string
	var action AgentAction
	var actionSeen bool
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
		pending += part
		events, rest, err := parseAgentStreamChunks(pending, false)
		if err != nil {
			return AgentAction{}, err
		}
		pending = rest
		for _, parsed := range events {
			if parsed.Event == "summary" {
				lastSummary = parsed.Summary
			}
			if parsed.Event == "action" && !actionSeen {
				action = parsed.Action
				if strings.TrimSpace(action.Thought) == "" {
					action.Thought = lastSummary
				}
				actionSeen = true
			}
			if onEvent != nil {
				onEvent(parsed)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return AgentAction{}, fmt.Errorf("read ai stream: %w", err)
	}
	if pending != "" {
		events, _, err := parseAgentStreamChunks(pending, true)
		if err != nil {
			return AgentAction{}, err
		}
		for _, parsed := range events {
			if parsed.Event == "summary" {
				lastSummary = parsed.Summary
			}
			if parsed.Event == "action" && !actionSeen {
				action = parsed.Action
				if strings.TrimSpace(action.Thought) == "" {
					action.Thought = lastSummary
				}
				actionSeen = true
			}
			if onEvent != nil {
				onEvent(parsed)
			}
		}
	}
	if actionSeen {
		return action, nil
	}
	return AgentAction{}, fmt.Errorf("agent stream did not produce a valid structured action event")
}

func (c *Client) buildAgentChatRequest(req AgentRequest, stream bool) (chatRequest, error) {
	stepsJSON, err := json.Marshal(req.Steps)
	if err != nil {
		return chatRequest{}, fmt.Errorf("marshal agent steps: %w", err)
	}
	historyJSON, err := json.Marshal(req.History)
	if err != nil {
		return chatRequest{}, fmt.Errorf("marshal agent history: %w", err)
	}

	return chatRequest{
		Model:  c.cfg.Model,
		Stream: stream,
		Messages: []chatMessage{
			{
				Role: "system",
				Content: fmt.Sprintf("你是运维 Agent。你会分步完成目标，必要时执行 shell 命令并根据输出继续决策。\n"+
					"你必须输出 NDJSON 事件流，每行一个 JSON 对象，不要 Markdown、不要代码块、不要任何额外文本。\n"+
					"第一行必须是 summary 事件，格式：{\"event\":\"summary\",\"summary\":\"一句简短状态\"}\n"+
					"第二行必须是 action 事件，格式之一：\n"+
					"{\"event\":\"action\",\"action\":{\"type\":\"command\",\"command\":\"单条可执行命令\"}}\n"+
					"{\"event\":\"action\",\"action\":{\"type\":\"final\",\"final\":\"结果总结\"}}\n"+
					"规则：\n"+
					"0) 这是一个持续会话，用户可能基于上文继续追问；你必须结合会话历史理解代词、省略和后续操作。\n"+
					"1) summary 必须简短、客观，不要把尚未执行的命令说成已经完成。\n"+
					"2) 未完成时 action.type 必须是 command。\n"+
					"3) 完成或无法继续时 action.type 必须是 final。\n"+
					"4) command 必须是单条可执行命令。\n"+
					"5) 除这两行 JSON 外不要输出任何其他内容。\n"+
					"环境信息: GOOS=%s, Distro=%s, Shell=%s, PWD=%s",
					req.Env.GOOS,
					req.Env.Distro,
					req.Env.Shell,
					req.Env.PWD,
				),
			},
			{
				Role: "user",
				Content: fmt.Sprintf("当前用户输入：%s\n\n本次会话历史(JSON数组)：%s\n\n当前轮执行记录(JSON数组)：%s",
					strings.TrimSpace(req.Goal),
					string(historyJSON),
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

	if _, action, err := parseAgentEventText(s); err == nil {
		return action, nil
	}
	return AgentAction{}, fmt.Errorf("agent must return structured NDJSON events")
}

func parseAgentEventText(content string) ([]AgentStreamEvent, AgentAction, error) {
	lines := strings.Split(strings.TrimSpace(content), "\n")
	events := make([]AgentStreamEvent, 0, len(lines))
	lastSummary := ""
	var action AgentAction
	actionSeen := false
	lineCount := 0

	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		lineCount++
		event, err := parseAgentEventLine(line)
		if err != nil {
			return nil, AgentAction{}, err
		}
		events = append(events, event)
		if event.Event == "summary" {
			lastSummary = event.Summary
		}
		if event.Event == "action" && !actionSeen {
			action = event.Action
			if strings.TrimSpace(action.Thought) == "" {
				action.Thought = lastSummary
			}
			actionSeen = true
		}
	}

	if lineCount == 0 {
		return nil, AgentAction{}, fmt.Errorf("empty structured response")
	}
	if !actionSeen {
		return nil, AgentAction{}, fmt.Errorf("missing action event")
	}
	return events, action, nil
}

func parseAgentStreamChunks(pending string, flush bool) ([]AgentStreamEvent, string, error) {
	var events []AgentStreamEvent
	rest := pending

	for {
		idx := strings.IndexByte(rest, '\n')
		if idx < 0 {
			break
		}
		line := strings.TrimSpace(rest[:idx])
		rest = rest[idx+1:]
		if line == "" {
			continue
		}
		event, err := parseAgentEventLine(line)
		if err != nil {
			return nil, pending, err
		}
		events = append(events, event)
	}

	if flush {
		line := strings.TrimSpace(rest)
		if line == "" {
			return events, "", nil
		}
		event, err := parseAgentEventLine(line)
		if err != nil {
			return nil, pending, err
		}
		events = append(events, event)
		return events, "", nil
	}

	return events, rest, nil
}

func parseAgentEventLine(line string) (AgentStreamEvent, error) {
	var event AgentStreamEvent
	if err := json.Unmarshal([]byte(strings.TrimSpace(line)), &event); err == nil {
		if normalized, ok := normalizeAgentStreamEvent(event); ok {
			return normalized, nil
		}
	}

	return AgentStreamEvent{}, fmt.Errorf("invalid agent stream event")
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
		action.Final = ""
		return action, true
	case "final":
		if action.Final == "" {
			return AgentAction{}, false
		}
		action.Command = ""
		return action, true
	default:
		return AgentAction{}, false
	}
}

func normalizeAgentStreamEvent(event AgentStreamEvent) (AgentStreamEvent, bool) {
	event.Event = strings.ToLower(strings.TrimSpace(event.Event))
	switch event.Event {
	case "summary", "status":
		event.Event = "summary"
		event.Summary = strings.TrimSpace(event.Summary)
		if event.Summary == "" {
			return AgentStreamEvent{}, false
		}
		event.Action = AgentAction{}
		return event, true
	case "action":
		action, ok := normalizeAgentAction(event.Action)
		if !ok {
			return AgentStreamEvent{}, false
		}
		event.Summary = ""
		event.Action = action
		return event, true
	default:
		return AgentStreamEvent{}, false
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
