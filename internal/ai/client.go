package ai

import (
	"context"
	"fmt"
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
	UserPrompt string
	Env        sysinfo.Env
	Retry      bool
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
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
	endpoint := chatCompletionsEndpoint(c.cfg.BaseURL)

	userPrompt := strings.TrimSpace(req.UserPrompt)
	if req.Retry {
		userPrompt += "\n上一个建议不满意，请换一种实现方式。"
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
	cmd := sanitize(out.Choices[0].Message.Content)
	if cmd == "" {
		return "", fmt.Errorf("empty command returned by ai")
	}
	return cmd, nil
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
