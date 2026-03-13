package ai

import "testing"

func TestParseAgentActionText(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    AgentAction
		wantErr bool
	}{
		{
			name:  "standard command format",
			input: "TYPE: command\nTHOUGHT: check service\nCOMMAND: systemctl status caddy",
			want: AgentAction{
				Type:    "command",
				Thought: "check service",
				Command: "systemctl status caddy",
			},
		},
		{
			name:  "missing type infers command",
			input: "THOUGHT: restart service first\nCOMMAND: systemctl restart caddy",
			want: AgentAction{
				Type:    "command",
				Thought: "restart service first",
				Command: "systemctl restart caddy",
			},
		},
		{
			name:  "supports chinese colon",
			input: "TYPE：final\nTHOUGHT：已经完成检查\nFINAL：Caddy 已成功启动。",
			want: AgentAction{
				Type:    "final",
				Thought: "已经完成检查",
				Final:   "Caddy 已成功启动。",
			},
		},
		{
			name:  "supports json payload",
			input: `{"type":"command","thought":"check status","command":"systemctl status caddy"}`,
			want: AgentAction{
				Type:    "command",
				Thought: "check status",
				Command: "systemctl status caddy",
			},
		},
		{
			name:    "rejects unsupported type",
			input:   "TYPE: plan\nTHOUGHT: think more",
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
