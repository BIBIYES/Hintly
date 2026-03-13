package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"hint/internal/ai"
	"hint/internal/executor"
	"hint/pkg/sysinfo"
)

type mode int

const (
	modeLoading mode = iota
	modeReady
	modeEditing
	modeConfirmDanger
	modeError
)

// Result is returned to main after UI exits.
type Result struct {
	Command   string
	Execute   bool
	Cancelled bool
}

type model struct {
	query string
	env   sysinfo.Env

	client *ai.Client
	spin   spinner.Model
	input  textinput.Model

	mode       mode
	suggestion string
	pendingCmd string
	lastErr    error
	result     Result
}

type suggestionMsg struct {
	command string
	err     error
}

func Run(query string, client *ai.Client, env sysinfo.Env) (Result, error) {
	spin := spinner.New()
	spin.Spinner = spinner.Spinner{Frames: []string{"⣾", "⣽", "⣻"}, FPS: spinner.Dot.FPS}

	input := textinput.New()
	input.Placeholder = "编辑命令后按 Enter"
	input.CharLimit = 1024
	input.Prompt = "> "

	m := model{
		query:  strings.TrimSpace(query),
		env:    env,
		client: client,
		spin:   spin,
		input:  input,
		mode:   modeLoading,
	}

	p := tea.NewProgram(m)
	out, err := p.Run()
	if err != nil {
		return Result{}, err
	}
	finalModel := out.(model)
	return finalModel.result, nil
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spin.Tick, m.fetchCmd(false))
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		s := msg.String()
		if s == "ctrl+c" || s == "esc" {
			m.result.Cancelled = true
			return m, tea.Quit
		}

		switch m.mode {
		case modeReady:
			return m.handleReadyKey(msg)
		case modeEditing:
			next, cmd := m.handleEditKey(msg)
			return next, cmd
		case modeConfirmDanger:
			return m.handleConfirmKey(msg)
		case modeError:
			if msg.String() == "r" {
				m.mode = modeLoading
				m.lastErr = nil
				return m, tea.Batch(m.spin.Tick, m.fetchCmd(true))
			}
		}

	case suggestionMsg:
		if msg.err != nil {
			m.mode = modeError
			m.lastErr = msg.err
			return m, nil
		}
		m.suggestion = msg.command
		m.mode = modeReady
		m.lastErr = nil
		return m, nil

	case spinner.TickMsg:
		if m.mode == modeLoading {
			var cmd tea.Cmd
			m.spin, cmd = m.spin.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m model) View() string {
	title := lipgloss.NewStyle().Bold(true).Render("hint")
	subtitle := fmt.Sprintf("OS: %s | Shell: %s | PWD: %s", m.env.Distro, m.env.Shell, m.env.PWD)

	switch m.mode {
	case modeLoading:
		return fmt.Sprintf("%s\n%s\n\n%s  正在向 AI 请求命令建议...", title, subtitle, m.spin.View())
	case modeError:
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
		return fmt.Sprintf("%s\n%s\n\n%s\n\n按 r 重试 | Esc/Ctrl+C 退出",
			title,
			subtitle,
			errStyle.Render("请求失败: "+m.lastErr.Error()),
		)
	case modeEditing:
		return fmt.Sprintf("%s\n%s\n\n编辑模式\n%s\n\nEnter 执行 | Esc/Ctrl+C 取消", title, subtitle, m.input.View())
	case modeConfirmDanger:
		warn := lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
		cmd := lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(m.pendingCmd)
		return fmt.Sprintf("%s\n%s\n\n%s\n%s\n\n请输入 y 确认执行，其他按键返回",
			title,
			subtitle,
			warn.Render("⚠️ 检测到危险操作"),
			cmd,
		)
	default:
		cmdStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
		headerStyle := lipgloss.NewStyle().Bold(true)
		danger := ""
		if executor.IsDangerous(m.suggestion) {
			danger = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("\n⚠️ 命令疑似高风险，Enter 后需要输入 y 确认")
		}

		return fmt.Sprintf("%s\n%s\n\n%s\n%s%s\n\nEnter 执行 | r 重试 | e 编辑 | Esc/Ctrl+C 取消",
			title,
			subtitle,
			headerStyle.Render("建议命令:"),
			cmdStyle.Render(m.suggestion),
			danger,
		)
	}
}

func (m model) fetchCmd(retry bool) tea.Cmd {
	previous := ""
	if retry {
		previous = m.suggestion
	}
	return func() tea.Msg {
		cmd, err := m.client.SuggestCommand(context.Background(), ai.Request{
			UserPrompt:      m.query,
			Env:             m.env,
			Retry:           retry,
			PreviousCommand: previous,
		})
		return suggestionMsg{command: cmd, err: err}
	}
}

func (m model) handleReadyKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		cmd := strings.TrimSpace(m.suggestion)
		if executor.IsDangerous(cmd) {
			m.mode = modeConfirmDanger
			m.pendingCmd = cmd
			return m, nil
		}
		m.result.Execute = true
		m.result.Command = cmd
		return m, tea.Quit
	case "r":
		m.mode = modeLoading
		return m, tea.Batch(m.spin.Tick, m.fetchCmd(true))
	case "e":
		m.mode = modeEditing
		m.input.SetValue(m.suggestion)
		m.input.Focus()
		return m, nil
	default:
		return m, nil
	}
}

func (m model) handleEditKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		cmd := strings.TrimSpace(m.input.Value())
		if cmd == "" {
			return m, nil
		}
		if executor.IsDangerous(cmd) {
			m.mode = modeConfirmDanger
			m.pendingCmd = cmd
			return m, nil
		}
		m.result.Execute = true
		m.result.Command = cmd
		return m, tea.Quit
	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
}

func (m model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if strings.EqualFold(msg.String(), "y") {
		m.result.Execute = true
		m.result.Command = m.pendingCmd
		return m, tea.Quit
	}
	m.mode = modeReady
	m.pendingCmd = ""
	return m, nil
}
