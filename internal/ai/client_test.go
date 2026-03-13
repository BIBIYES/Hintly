package ai

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"hint/internal/executor"
	"hint/pkg/sysinfo"
)

func TestParseAgentActionText(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    AgentAction
		wantErr bool
	}{
		{
			name: "supports ndjson command protocol",
			input: "{\"event\":\"summary\",\"summary\":\"先检查 fail2ban sshd 状态\"}\n" +
				"{\"event\":\"action\",\"action\":{\"type\":\"command\",\"command\":\"fail2ban-client status sshd\"}}",
			want: AgentAction{
				Type:    "command",
				Thought: "先检查 fail2ban sshd 状态",
				Command: "fail2ban-client status sshd",
			},
		},
		{
			name: "supports ndjson final protocol",
			input: "{\"event\":\"summary\",\"summary\":\"已经拿到检查结果\"}\n" +
				"{\"event\":\"action\",\"action\":{\"type\":\"final\",\"final\":\"当前 sshd jail 中共有 2 个封禁 IP。\"}}",
			want: AgentAction{
				Type:    "final",
				Thought: "已经拿到检查结果",
				Final:   "当前 sshd jail 中共有 2 个封禁 IP。",
			},
		},
		{
			name:    "rejects legacy text protocol",
			input:   "TYPE: command\nTHOUGHT: check service\nCOMMAND: systemctl status caddy",
			wantErr: true,
		},
		{
			name:    "rejects plain action json without event wrapper",
			input:   `{"type":"command","thought":"check status","command":"systemctl status caddy"}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAgentActionText(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil and action=%+v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("unexpected action:\n got: %+v\nwant: %+v", got, tt.want)
			}
		})
	}
}

func TestParseAgentStreamChunks(t *testing.T) {
	chunk1 := "{\"event\":\"summary\",\"summary\":\"先检查"
	chunk2 := " fail2ban sshd 状态\"}\n{\"event\":\"action\",\"action\":{\"type\":\"command\",\"command\":\"fail2ban-client status sshd\"}}\n"

	events, rest, err := parseAgentStreamChunks(chunk1, false)
	if err != nil {
		t.Fatalf("unexpected error on partial chunk: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected no complete events yet, got %d", len(events))
	}
	if rest != chunk1 {
		t.Fatalf("unexpected rest after partial chunk: %q", rest)
	}

	events, rest, err = parseAgentStreamChunks(chunk1+chunk2, false)
	if err != nil {
		t.Fatalf("unexpected error on complete chunks: %v", err)
	}
	if rest != "" {
		t.Fatalf("expected empty rest, got %q", rest)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Event != "summary" || events[0].Summary != "先检查 fail2ban sshd 状态" {
		t.Fatalf("unexpected summary event: %+v", events[0])
	}
	if events[1].Event != "action" || events[1].Action.Command != "fail2ban-client status sshd" {
		t.Fatalf("unexpected action event: %+v", events[1])
	}
}

func TestBuildAgentChatRequestIncludesSessionHistory(t *testing.T) {
	client := &Client{}
	req, err := client.buildAgentChatRequest(AgentRequest{
		Goal: "然后再看看日志",
		Env: sysinfo.Env{
			GOOS:   "linux",
			Distro: "ubuntu",
			Shell:  "/bin/zsh",
			PWD:    "/srv/app",
		},
		History: []AgentTurn{
			{
				Goal:    "重启 caddy",
				Outcome: "Caddy 已成功重启。",
			},
		},
	}, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(req.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(req.Messages))
	}
	userContent := req.Messages[1].Content
	for _, want := range []string{"本次会话历史(JSON数组)", "重启 caddy", "Caddy 已成功重启。", "然后再看看日志"} {
		if !strings.Contains(userContent, want) {
			t.Fatalf("expected user content to include %q, got: %s", want, userContent)
		}
	}
}

func TestSanitizePreservesMultilineCommands(t *testing.T) {
	raw := "```bash\ncat <<'EOF' > note.txt\n第一行，带中文标点：你好。\n第二行\nEOF\n```"

	got := sanitize(raw)
	want := "cat <<'EOF' > note.txt\n第一行，带中文标点：你好。\n第二行\nEOF"
	if got != want {
		t.Fatalf("unexpected sanitized command:\n got: %q\nwant: %q", got, want)
	}
}

func TestParseAgentActionTextPreservesMultilineCommand(t *testing.T) {
	input := "{\"event\":\"summary\",\"summary\":\"写入文件\"}\n" +
		"{\"event\":\"action\",\"action\":{\"type\":\"command\",\"command\":\"cat <<'EOF' > note.txt\\n第一行，带中文标点：你好。\\n第二行\\nEOF\"}}"

	got, err := parseAgentActionText(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "cat <<'EOF' > note.txt\n第一行，带中文标点：你好。\n第二行\nEOF"
	if got.Command != want {
		t.Fatalf("unexpected command:\n got: %q\nwant: %q", got.Command, want)
	}
}

func TestParseAgentActionTextSupportsFencedNdjson(t *testing.T) {
	input := "```json\n" +
		"{\"event\":\"summary\",\"summary\":\"检查完成\"}\n" +
		"{\"event\":\"action\",\"action\":{\"type\":\"final\",\"final\":\"一切正常\"}}\n" +
		"```"

	got, err := parseAgentActionText(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Type != "final" || got.Final != "一切正常" {
		t.Fatalf("unexpected action: %+v", got)
	}
}

func TestSanitizedHereDocCommandWritesFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("heredoc command is shell-specific")
	}

	dir := t.TempDir()
	target := filepath.Join(dir, "note.txt")
	raw := "```bash\ncat <<'EOF' > \"" + target + "\"\n第一行，带中文标点：你好。\n第二行\nEOF\n```"

	command := sanitize(raw)
	output, exitCode, err := executor.RunCapture(command, 5*time.Second)
	if err != nil {
		t.Fatalf("unexpected execution error: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("unexpected exit code %d with output %q", exitCode, output)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	want := "第一行，带中文标点：你好。\n第二行\n"
	if string(data) != want {
		t.Fatalf("unexpected file content:\n got: %q\nwant: %q", string(data), want)
	}
}
