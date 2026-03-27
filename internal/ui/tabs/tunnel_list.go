package tabs

import (
	"fmt"
	"strconv"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"

	"github.com/ssh-tun-tui/internal/tunnel"
	"github.com/ssh-tun-tui/internal/ui"
)

type TunnelListModel struct {
	table  table.Model
	width  int
	height int
}

func NewTunnelListModel() TunnelListModel {
	columns := []table.Column{
		{Title: "S", Width: 3},
		{Title: "Name", Width: 20},
		{Title: "Type", Width: 8},
		{Title: "LPort", Width: 7},
		{Title: "RHost", Width: 18},
		{Title: "RPort", Width: 7},
		{Title: "Login", Width: 30},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		Bold(true).
		Foreground(ui.ColorCyan)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("#1E1E2E")).
		Background(ui.ColorCyan).
		Bold(false)
	t.SetStyles(s)

	return TunnelListModel{table: t}
}

func (m *TunnelListModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.table.SetWidth(w)
	m.table.SetHeight(h - 4)
}

func (m *TunnelListModel) UpdateTunnels(tunnels []tunnel.Tunnel) {
	var rows []table.Row
	for _, t := range tunnels {
		state := "○"
		if t.Error {
			state = ui.StyleError.Render("✗")
		} else if t.Running {
			state = ui.StyleRunning.Render("●")
		}

		rhost := t.RemoteHost
		if rhost == "" {
			rhost = "—"
		}
		rport := "—"
		if t.Type != tunnel.TunnelDynamic {
			rport = strconv.Itoa(t.RemotePort)
		}

		rows = append(rows, table.Row{
			state,
			t.Name,
			t.Type.Display(),
			strconv.Itoa(t.LocalPort),
			rhost,
			rport,
			t.Login,
		})
	}
	m.table.SetRows(rows)
}

func (m *TunnelListModel) SelectedTunnelName() string {
	row := m.table.SelectedRow()
	if row == nil {
		return ""
	}
	return row[1]
}

func (m *TunnelListModel) Update(msg tea.Msg) (TunnelListModel, tea.Cmd) {
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return *m, cmd
}

func (m TunnelListModel) View() string {
	return ui.StyleFocused.Render(m.table.View())
}

func (m *TunnelListModel) StatusText(tunnels []tunnel.Tunnel, runningCount int) string {
	total := len(tunnels)
	return fmt.Sprintf("%d/%d tunnels running", runningCount, total)
}
