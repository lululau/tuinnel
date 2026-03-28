package tabs

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/lululau/tuinnel/internal/ui"
)

type TabID int

const (
	TabTunnels TabID = iota
	TabLogs
	TabSettings
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
	default:
		return "Unknown"
	}
}

func AllTabs() []TabID {
	return []TabID{TabTunnels, TabLogs, TabSettings}
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

	right := tb.width - lipgloss.Width(tabs) - lipgloss.Width(help) - 2
	if right < 0 {
		right = 0
	}

	row := fmt.Sprintf("%s%s%s", tabs, strings.Repeat(" ", right), help)

	// Full-width separator line below the tab bar
	sep := ui.StyleTabSep.Render(strings.Repeat("─", tb.width))

	return lipgloss.NewStyle().Width(tb.width).Render(
		fmt.Sprintf("%s\n%s", row, sep),
	)
}
