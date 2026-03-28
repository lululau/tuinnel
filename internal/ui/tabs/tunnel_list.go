package tabs

import (
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"

	"github.com/lululau/tuinnel/internal/tunnel"
	"github.com/lululau/tuinnel/internal/ui"
)

type deleteConfirm struct {
	active     bool
	tunnelName string
}

type TunnelListModel struct {
	table           table.Model
	editor          EditorModel
	filterInput     textinput.Model
	filterActive    bool
	deleteConfirm   deleteConfirm
	allTunnels      []tunnel.Tunnel
	filteredIndices []int
	width           int
	height          int
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

	fi := textinput.New()
	fi.Prompt = "/"
	fi.CharLimit = 30
	fi.SetWidth(40)

	return TunnelListModel{
		table:       t,
		editor:      NewEditorModel(),
		filterInput: fi,
	}
}

func (m *TunnelListModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.table.SetWidth(w)
	m.table.SetHeight(h - 1) // leave room for optional filter input
	m.editor.SetSize(w, h)
	m.filterInput.SetWidth(w - 4)
	// Dynamically size Login column to fill remaining width
	fixedWidth := 3 + 20 + 8 + 7 + 18 + 7 // sum of other columns = 63
	loginWidth := max(w-fixedWidth, 10)
	m.table.SetColumns([]table.Column{
		{Title: "S", Width: 3},
		{Title: "Name", Width: 20},
		{Title: "Type", Width: 8},
		{Title: "LPort", Width: 7},
		{Title: "RHost", Width: 18},
		{Title: "RPort", Width: 7},
		{Title: "Login", Width: loginWidth},
	})
}

func (m *TunnelListModel) UpdateTunnels(tunnels []tunnel.Tunnel) {
	m.allTunnels = tunnels
	m.applyFilter()
}

func (m *TunnelListModel) applyFilter() {
	query := strings.ToLower(m.filterInput.Value())
	m.filteredIndices = nil
	for i, t := range m.allTunnels {
		if query == "" || strings.Contains(strings.ToLower(t.Name), query) {
			m.filteredIndices = append(m.filteredIndices, i)
		}
	}

	var rows []table.Row
	for _, idx := range m.filteredIndices {
		t := m.allTunnels[idx]
		state := "○"
		if t.Error {
			state = "✗"
		} else if t.Running {
			state = "●"
		} else if t.Stale {
			state = "◐"
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

func (m *TunnelListModel) FilteredCount() int {
	return len(m.filteredIndices)
}

func (m *TunnelListModel) SelectedTunnelName() string {
	row := m.table.SelectedRow()
	if row == nil {
		return ""
	}
	return row[1]
}

func (m *TunnelListModel) SelectedTunnelIndex() int {
	tableRow := m.table.Cursor()
	if tableRow < 0 || tableRow >= len(m.filteredIndices) {
		return -1
	}
	return m.filteredIndices[tableRow]
}

func (m *TunnelListModel) IsEditorActive() bool {
	return m.editor.IsEditing()
}

func (m *TunnelListModel) IsDeleteConfirmActive() bool {
	return m.deleteConfirm.active
}

func (m *TunnelListModel) IsFilterActive() bool {
	return m.filterActive
}

func (m *TunnelListModel) ShowDeleteConfirm(name string) {
	m.deleteConfirm.active = true
	m.deleteConfirm.tunnelName = name
}

func (m *TunnelListModel) CancelDeleteConfirm() {
	m.deleteConfirm.active = false
	m.deleteConfirm.tunnelName = ""
}

func (m *TunnelListModel) Editor() *EditorModel {
	return &m.editor
}

func (m *TunnelListModel) ActivateFilter() tea.Cmd {
	m.filterActive = true
	return m.filterInput.Focus()
}

func (m *TunnelListModel) Update(msg tea.Msg) (TunnelListModel, tea.Cmd) {
	var cmd tea.Cmd

	// Editor modal takes priority
	if m.editor.IsEditing() {
		m.editor, cmd = m.editor.Update(msg)
		return *m, cmd
	}

	// Delete confirm modal
	if m.deleteConfirm.active {
		if msg, ok := msg.(tea.KeyPressMsg); ok {
			switch msg.String() {
			case "y":
				return *m, func() tea.Msg { return DeleteTunnelMsg{Name: m.deleteConfirm.tunnelName} }
			case "n", "esc":
				m.CancelDeleteConfirm()
				return *m, nil
			}
		}
		return *m, nil
	}

	// Filter input
	if m.filterActive {
		if msg, ok := msg.(tea.KeyPressMsg); ok {
			switch msg.String() {
			case "esc":
				m.filterInput.SetValue("")
				m.filterActive = false
				m.applyFilter()
				return *m, nil
			case "enter":
				m.filterActive = false
				return *m, nil
			}
		}
		m.filterInput, cmd = m.filterInput.Update(msg)
		m.applyFilter()
		return *m, cmd
	}

	// Normal list operation
	m.table, cmd = m.table.Update(msg)
	return *m, cmd
}

func (m TunnelListModel) View() string {
	listView := ui.StyleFocused.Render(m.table.View())

	// Filter input below table
	if m.filterActive || m.filterInput.Value() != "" {
		listView += "\n" + m.filterInput.View()
	}

	// Delete confirm modal
	if m.deleteConfirm.active {
		confirmContent := fmt.Sprintf(
			"%s\n\n%s",
			ui.StyleError.Render(fmt.Sprintf("Tunnel %q is running.", m.deleteConfirm.tunnelName)),
			"Delete anyway? (y/n)",
		)
		panel := ui.StyleModal.Render(confirmContent)
		return lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			panel,
		)
	}

	// Editor modal
	if m.editor.IsEditing() {
		panel := ui.StyleModal.Render(m.editor.View())
		return lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			panel,
		)
	}

	return listView
}

func (m *TunnelListModel) StatusText(tunnels []tunnel.Tunnel, runningCount int) string {
	total := len(tunnels)
	visible := m.FilteredCount()

	staleCount := 0
	for _, t := range tunnels {
		if t.Stale {
			staleCount++
		}
	}

	left := fmt.Sprintf("%d/%d tunnels running", runningCount, total)
	if staleCount > 0 {
		left += fmt.Sprintf(", %d stale", staleCount)
	}
	if m.filterActive || m.filterInput.Value() != "" {
		left = fmt.Sprintf("%d/%d tunnels visible", visible, total)
	}

	return left
}

type DeleteTunnelMsg struct {
	Name string
}
