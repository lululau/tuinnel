package tabs

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"

	"github.com/lululau/tuinnel/internal/tunnel"
	"github.com/lululau/tuinnel/internal/ui"
)

type LogModel struct {
	tunnels  []tunnel.Tunnel
	cursor   int
	viewport viewport.Model
	width    int
	height   int
}

func NewLogModel() LogModel {
	vp := viewport.New(viewport.WithWidth(40), viewport.WithHeight(10))
	vp.SetContent("Select a tunnel to view logs.")
	return LogModel{viewport: vp}
}

func (m *LogModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.viewport.SetWidth(w/2 - 2)
	m.viewport.SetHeight(h - 4)
}

func (m *LogModel) UpdateTunnels(tunnels []tunnel.Tunnel) {
	m.tunnels = tunnels
	if m.cursor >= len(tunnels) {
		m.cursor = max(0, len(tunnels)-1)
	}
}

func (m *LogModel) UpdateLogs(name string, lines []string) {
	content := strings.Join(lines, "\n")
	if content == "" {
		content = "No logs yet."
	}
	m.viewport.SetContent(content)
	m.viewport.GotoBottom()
}

func (m *LogModel) SelectedTunnelName() string {
	if len(m.tunnels) == 0 {
		return ""
	}
	return m.tunnels[m.cursor].Name
}

func (m *LogModel) Update(msg tea.Msg) (LogModel, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			return *m, nil
		case "down", "j":
			if m.cursor < len(m.tunnels)-1 {
				m.cursor++
			}
			return *m, nil
		}
	}
	m.viewport, cmd = m.viewport.Update(msg)
	return *m, cmd
}

func (m LogModel) View() string {
	var tunnelList strings.Builder
	for i, t := range m.tunnels {
		cursor := " "
		if i == m.cursor {
			cursor = ">"
		}
		state := "○"
		if t.Running {
			state = "●"
		}
		tunnelList.WriteString(fmt.Sprintf("%s %s %s\n", cursor, state, t.Name))
	}

	left := lipgloss.NewStyle().
		Width(m.width/2 - 1).
		Height(m.height - 4).
		Render(tunnelList.String())

	right := ui.StyleFocused.Render(m.viewport.View())

	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}
