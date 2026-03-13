package agent

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"hint/internal/ai"
	"hint/internal/executor"
	"hint/pkg/sysinfo"
)

const (
	maxSteps       = 8
	commandTimeout = 45 * time.Second
	maxOutputChars = 4000
)

type messageRole int

const (
	roleUser messageRole = iota
	roleAgent
	roleSystem
)

type chatMessage struct {
	role    messageRole
	title   string
	content string
}

type streamDeltaMsg struct {
	delta string
}

type streamDoneMsg struct {
	action ai.AgentAction
}

type streamErrMsg struct {
	err error
}

type execDoneMsg struct {
	output   string
	exitCode int
	err      error
}

type model struct {
	client *ai.Client
	env    sysinfo.Env

	input    textinput.Model
	viewport viewport.Model
	spin     spinner.Model

	width  int
	height int

	messages []chatMessage
	eventCh  chan tea.Msg

	running      bool
	streaming    bool
	executing    bool
	currentGoal  string
	steps        []ai.AgentObservation
	stepCount    int
	streamBuf    string
	streamMsgIdx int

	pendingThought string
	pendingCommand string
}

// Run launches the chat-style agent UI.
func Run(_ io.Reader, _ io.Writer, client *ai.Client, env sysinfo.Env) error {
	ti := textinput.New()
	ti.Prompt = "You > "
	ti.Placeholder = "输入运维目标，按 Enter 发送（Ctrl+C / Esc 退出）"
	ti.Focus()
	ti.CharLimit = 2048

	sp := spinner.New()
	sp.Spinner = spinner.Spinner{Frames: []string{"⣾", "⣽", "⣻"}, FPS: spinner.Dot.FPS}

	vp := viewport.New(80, 18)

	m := &model{
		client:       client,
		env:          env,
		input:        ti,
		viewport:     vp,
		spin:         sp,
		messages:     make([]chatMessage, 0, 64),
		eventCh:      make(chan tea.Msg, 256),
		streamMsgIdx: -1,
	}
	m.append(roleSystem, "System", "欢迎使用 Hintly Agent。底部输入目标并回车，Agent 会自动多步执行直到完成。")
	m.refreshViewport()

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (m *model) Init() tea.Cmd {
	return m.spin.Tick
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layout()
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			text := strings.TrimSpace(m.input.Value())
			if text == "" {
				return m, nil
			}
			m.input.SetValue("")
			if m.running {
				m.append(roleSystem, "System", "Agent 正在执行当前目标，请稍候。")
				m.refreshViewport()
				return m, nil
			}
			m.startGoal(text)
			return m, tea.Batch(waitEvent(m.eventCh), m.spin.Tick)
		case "pgup", "b":
			m.viewport.HalfViewUp()
			return m, nil
		case "pgdown", "f":
			m.viewport.HalfViewDown()
			return m, nil
		}

		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd

	case streamDeltaMsg:
		m.streamBuf += msg.delta
		m.updateStreamThought()
		return m, waitEvent(m.eventCh)

	case streamDoneMsg:
		m.streaming = false
		if m.streamMsgIdx >= 0 && m.streamMsgIdx < len(m.messages) {
			m.messages[m.streamMsgIdx].content = strings.TrimSpace(msg.action.Thought)
			if m.messages[m.streamMsgIdx].content == "" {
				m.messages[m.streamMsgIdx].content = "已完成思考。"
			}
		}
		m.streamMsgIdx = -1

		switch msg.action.Type {
		case "final":
			m.append(roleAgent, "Agent", msg.action.Final)
			m.running = false
			m.executing = false
			m.refreshViewport()
			return m, nil
		case "command":
			m.pendingThought = strings.TrimSpace(msg.action.Thought)
			m.pendingCommand = strings.TrimSpace(msg.action.Command)
			m.append(roleAgent, "Agent", "$ "+m.pendingCommand)
			if executor.IsDangerous(m.pendingCommand) {
				m.append(roleSystem, "System", "危险命令已拦截，Agent 将继续尝试其他方案。")
				m.steps = append(m.steps, ai.AgentObservation{
					Thought:  m.pendingThought,
					Command:  m.pendingCommand,
					Output:   "blocked by safety policy: dangerous command",
					ExitCode: -1,
				})
				m.refreshViewport()
				m.startPlanning()
				return m, tea.Batch(waitEvent(m.eventCh), m.spin.Tick)
			}
			m.executing = true
			m.refreshViewport()
			m.executeCommand(m.pendingCommand)
			return m, tea.Batch(waitEvent(m.eventCh), m.spin.Tick)
		default:
			m.append(roleSystem, "System", "Agent 输出了未知动作，已停止本轮任务。")
			m.running = false
			m.executing = false
			m.refreshViewport()
			return m, nil
		}

	case streamErrMsg:
		m.streaming = false
		m.executing = false
		m.running = false
		m.streamMsgIdx = -1
		m.append(roleSystem, "System", "Agent 规划失败: "+msg.err.Error())
		m.refreshViewport()
		return m, nil

	case execDoneMsg:
		m.executing = false
		ob := ai.AgentObservation{
			Thought: m.pendingThought,
			Command: m.pendingCommand,
		}
		if msg.err != nil {
			ob.ExitCode = -1
			errText := "命令执行失败"
			if strings.Contains(strings.ToLower(msg.err.Error()), "timed out") {
				errText = "命令执行超时"
			}
			if strings.TrimSpace(msg.output) == "" {
				ob.Output = errText
			} else {
				ob.Output = errText + "\n" + clipOutput(msg.output)
			}
			m.append(roleSystem, "System", ob.Output)
		} else {
			ob.ExitCode = msg.exitCode
			if strings.TrimSpace(msg.output) == "" {
				ob.Output = "(no output)"
			} else {
				ob.Output = clipOutput(msg.output)
			}
			m.append(roleSystem, "System", fmt.Sprintf("[exit=%d]\n%s", ob.ExitCode, ob.Output))
		}
		m.steps = append(m.steps, ob)
		m.refreshViewport()

		m.startPlanning()
		if m.running {
			return m, tea.Batch(waitEvent(m.eventCh), m.spin.Tick)
		}
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *model) View() string {
	header := m.renderHeader()
	body := m.viewport.View()
	footer := m.renderFooter()
	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

func (m *model) renderHeader() string {
	status := "Ready"
	if m.streaming {
		status = "Agent 思考中 " + m.spin.View()
	} else if m.executing {
		status = "Agent 执行命令中 " + m.spin.View()
	}
	head := fmt.Sprintf("Hintly Agent Chat | %s | %s | %s", m.env.Distro, m.env.Shell, status)
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("24")).
		Padding(0, 1).
		Width(max(20, m.width)).
		Render(head)
}

func (m *model) renderFooter() string {
	help := "Enter 发送 | PgUp/PgDn 滚动 | Esc/Ctrl+C 退出"
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		Padding(0, 1).
		Render(m.input.View())
	return lipgloss.JoinVertical(lipgloss.Left, box, lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(help))
}

func (m *model) append(role messageRole, title, content string) int {
	m.messages = append(m.messages, chatMessage{
		role:    role,
		title:   title,
		content: strings.TrimSpace(content),
	})
	return len(m.messages) - 1
}

func (m *model) refreshViewport() {
	if m.viewport.Width <= 0 {
		return
	}
	var chunks []string
	for _, msg := range m.messages {
		chunks = append(chunks, m.renderMessage(msg))
	}
	m.viewport.SetContent(strings.Join(chunks, "\n\n"))
	m.viewport.GotoBottom()
}

func (m *model) renderMessage(msg chatMessage) string {
	maxBubbleWidth := max(24, m.viewport.Width-8)

	titleStyle := lipgloss.NewStyle().Bold(true)
	bodyStyle := lipgloss.NewStyle().Width(maxBubbleWidth).UnsetPadding()
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1)

	switch msg.role {
	case roleUser:
		titleStyle = titleStyle.Foreground(lipgloss.Color("15"))
		bodyStyle = bodyStyle.Foreground(lipgloss.Color("15"))
		box := boxStyle.
			Background(lipgloss.Color("62")).
			BorderForeground(lipgloss.Color("63")).
			Render(titleStyle.Render(msg.title) + "\n" + bodyStyle.Render(msg.content))
		return lipgloss.PlaceHorizontal(m.viewport.Width, lipgloss.Right, box)
	case roleAgent:
		titleStyle = titleStyle.Foreground(lipgloss.Color("10"))
		bodyStyle = bodyStyle.Foreground(lipgloss.Color("15"))
		return boxStyle.
			BorderForeground(lipgloss.Color("10")).
			Render(titleStyle.Render(msg.title) + "\n" + bodyStyle.Render(msg.content))
	default:
		titleStyle = titleStyle.Foreground(lipgloss.Color("14"))
		bodyStyle = bodyStyle.Foreground(lipgloss.Color("252"))
		return boxStyle.
			BorderForeground(lipgloss.Color("8")).
			Render(titleStyle.Render(msg.title) + "\n" + bodyStyle.Render(msg.content))
	}
}

func (m *model) startGoal(goal string) {
	m.running = true
	m.streaming = false
	m.executing = false
	m.currentGoal = goal
	m.steps = nil
	m.stepCount = 0
	m.streamBuf = ""
	m.streamMsgIdx = -1
	m.pendingThought = ""
	m.pendingCommand = ""

	m.append(roleUser, "You", goal)
	m.append(roleSystem, "System", fmt.Sprintf("开始执行目标（最多 %d 步）", maxSteps))
	m.refreshViewport()
	m.startPlanning()
}

func (m *model) startPlanning() {
	if !m.running {
		return
	}
	if m.stepCount >= maxSteps {
		m.running = false
		m.streaming = false
		m.executing = false
		m.append(roleSystem, "System", fmt.Sprintf("达到最大步数 %d，任务停止。", maxSteps))
		m.refreshViewport()
		return
	}

	m.stepCount++
	m.streaming = true
	m.executing = false
	m.streamBuf = ""
	m.streamMsgIdx = m.append(roleAgent, "Agent", "思考中...")
	m.refreshViewport()

	goal := m.currentGoal
	stepsCopy := append([]ai.AgentObservation(nil), m.steps...)
	env := m.env
	client := m.client
	ch := m.eventCh

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()

		action, err := client.NextAgentActionStream(ctx, ai.AgentRequest{
			Goal:  goal,
			Env:   env,
			Steps: stepsCopy,
		}, func(delta string) {
			ch <- streamDeltaMsg{delta: delta}
		})
		if err != nil {
			ch <- streamErrMsg{err: err}
			return
		}
		ch <- streamDoneMsg{action: action}
	}()
}

func (m *model) executeCommand(command string) {
	ch := m.eventCh
	go func() {
		output, exitCode, err := executor.RunCapture(command, commandTimeout)
		ch <- execDoneMsg{
			output:   output,
			exitCode: exitCode,
			err:      err,
		}
	}()
}

func (m *model) updateStreamThought() {
	if m.streamMsgIdx < 0 || m.streamMsgIdx >= len(m.messages) {
		return
	}

	thought := extractField(m.streamBuf, "THOUGHT:")
	if thought == "" {
		thought = "思考中..."
	}
	m.messages[m.streamMsgIdx].content = thought
	m.refreshViewport()
}

func (m *model) layout() {
	if m.width <= 0 || m.height <= 0 {
		return
	}
	headerHeight := 1
	footerHeight := 4
	bodyHeight := m.height - headerHeight - footerHeight
	if bodyHeight < 5 {
		bodyHeight = 5
	}

	m.viewport.Width = max(20, m.width-2)
	m.viewport.Height = bodyHeight

	inputWidth := max(10, m.width-8)
	m.input.Width = inputWidth
	m.refreshViewport()
}

func waitEvent(ch <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}

func extractField(v, field string) string {
	upper := strings.ToUpper(v)
	marker := strings.ToUpper(field)
	start := strings.Index(upper, marker)
	if start == -1 {
		return ""
	}
	raw := v[start+len(field):]
	raw = strings.TrimLeft(raw, " \t\r\n")

	limit := len(raw)
	upperRaw := strings.ToUpper(raw)
	for _, next := range []string{"\nTYPE:", "\nCOMMAND:", "\nFINAL:", "\nTHOUGHT:"} {
		if idx := strings.Index(upperRaw, next); idx >= 0 && idx < limit {
			limit = idx
		}
	}
	return strings.TrimSpace(raw[:limit])
}

func clipOutput(v string) string {
	s := strings.TrimSpace(v)
	if len(s) <= maxOutputChars {
		return s
	}
	return s[:maxOutputChars] + "\n...(truncated)"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
