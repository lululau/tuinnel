package tabs

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ssh-tun-tui/internal/ui"
)

type TabID int

const (
	TabTunnels TabID = iota
	TabLogs
	TabSettings
	TabEditor
	TabCount
)

func (id TabID) String() string {
	switch id {
	case TabTunnels:
		return "Tunnels"
	case TabLogs:
		return "Logs"
	case TabSettings:
		return "Settings"
	case TabEditor:
		return "Editor"
	default:
		return "Unknown"
	}
}

func AllTabs() []TabID {
	return []TabID{TabTunnels, TabLogs, TabSettings, TabEditor}
}

type TabBar struct {
	active TabID
	width  int
}

func NewTabBar() TabBar {
	return TabBar{active: TabTunnels}
}

func (tb *TabBar) SetActive(id TabID) {
	tb.active = id
}

func (tb *TabBar) Active() TabID {
	return tb.active
}

func (tb *TabBar) SetWidth(w int) {
	tb.width = w
}

func (tb *TabBar) Update(msg tea.Msg) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		tb.width = msg.Width
	}
}

func (tb *TabBar) View() string {
	var parts []string
	for _, id := range AllTabs() {
		if id == tb.active {
			parts = append(parts, ui.StyleTabActive.Render(id.String()))
		} else {
			parts = append(parts, ui.StyleTabInactive.Render(id.String()))
		}
	}

	tabs := strings.Join(parts, " ")
	help := ui.StyleHelp.Render("?=Help")

	right := tb.width - lipgloss.Width(tabs) - 2
	if right < 0 {
		right = 0
	}

	return lipgloss.NewStyle().Width(tb.width).Render(
		fmt.Sprintf("%s%s%s", tabs, strings.Repeat(" ", right), help),
	)
}
