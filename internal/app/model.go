package app

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/lipgloss/v2"

	"github.com/lululau/tuinnel/internal/tunnel"
	"github.com/lululau/tuinnel/internal/ui"
	"github.com/lululau/tuinnel/internal/ui/tabs"
)

var appKeys = struct {
	Tab1    key.Binding
	Tab2    key.Binding
	Tab3    key.Binding
	TabNext key.Binding
	TabPrev key.Binding
	Help    key.Binding
	Quit    key.Binding
}{
	Tab1:    key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "tunnels")),
	Tab2:    key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "logs")),
	Tab3:    key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "settings")),
	TabNext: key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next tab")),
	TabPrev: key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("⇧+tab", "prev tab")),
	Help:    key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}

type confirmQuit struct {
	active bool
}

type tunnelResultMsg struct {
	action string // "start", "stop", "restart"
	name   string
	err    error
}

type stopAllResultMsg struct {
	err error
}

type orphanScanMsg struct {
	lines []string // unmanaged SSH tunnel command lines
}

type Model struct {
	mgr         *tunnel.Manager
	config      *tunnel.Config
	configPath  string
	tabBar      tabs.TabBar
	listTab     tabs.TunnelListModel
	logTab      tabs.LogModel
	settingsTab tabs.SettingsModel
	width       int
	height      int
	confirm     confirmQuit
	showHelp    bool
	statusMsg   string
	quitting    bool
	spinner     spinner.Model
	loading     string   // tunnel name when loading, empty otherwise
	loadingAct  string   // "start", "stop", "restart"
	orphanLines []string // unmanaged SSH tunnel command lines, nil if dismissed or none
}

func NewModel(cfg *tunnel.Config, configPath string) Model {
	mgr := tunnel.NewManager(cfg)

	sp := spinner.New(spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(ui.ColorCyan)))

	m := Model{
		mgr:         mgr,
		config:      cfg,
		configPath:  configPath,
		tabBar:      tabs.NewTabBar(),
		listTab:     tabs.NewTunnelListModel(),
		logTab:      tabs.NewLogModel(),
		settingsTab: tabs.NewSettingsModel(cfg.Settings),
		spinner:     sp,
	}
	m.mgr.Refresh()
	m.syncTunnels()

	return m
}

func (m Model) Init() tea.Cmd {
	return m.scanOrphanTunnelsCmd()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.tabBar.SetWidth(msg.Width)
		contentHeight := msg.Height - 6
		m.listTab.SetSize(msg.Width, contentHeight)
		m.logTab.SetSize(msg.Width, contentHeight)
		m.settingsTab.SetSize(msg.Width, contentHeight)
		return m, nil

	case tea.KeyPressMsg:
		// Ignore keys while loading
		if m.loading != "" {
			return m, nil
		}

		// Dismiss warning with x
		if msg.String() == "x" && m.orphanLines != nil {
			m.orphanLines = nil
			return m, nil
		}

		if m.showHelp {
			switch msg.String() {
			case "?", "esc", "q":
				m.showHelp = false
				return m, nil
			}
			return m, nil
		}

		if m.confirm.active {
			switch msg.String() {
			case "k":
				m.loading = "all tunnels"
				m.loadingAct = "stop"
				m.confirm.active = false
				return m, tea.Batch(m.spinner.Tick, m.stopAllCmd())
			case "l":
				m.quitting = true
				return m, tea.Quit
			case "n", "esc":
				m.confirm.active = false
				return m, nil
			}
			return m, nil
		}

		// Don't intercept global keys when settings has focused input
		if m.settingsTab.Saved && m.tabBar.Active() == tabs.TabSettings {
			m.settingsTab.Saved = false
		}

		switch {
		case key.Matches(msg, appKeys.Quit):
			if m.mgr.HasRunning() {
				m.confirm.active = true
				return m, nil
			}
			m.quitting = true
			return m, tea.Quit
		case key.Matches(msg, appKeys.Tab1):
			m.switchTab(tabs.TabTunnels)
			return m, nil
		case key.Matches(msg, appKeys.Tab2):
			m.switchTab(tabs.TabLogs)
			return m, nil
		case key.Matches(msg, appKeys.Tab3):
			m.switchTab(tabs.TabSettings)
			return m, nil
		case key.Matches(msg, appKeys.TabNext):
			if m.listTab.IsEditorActive() {
				break
			}
			next := tabs.TabID((int(m.tabBar.Active()) + 1) % int(tabs.TabCount))
			m.switchTab(next)
			return m, nil
		case key.Matches(msg, appKeys.TabPrev):
			if m.listTab.IsEditorActive() {
				break
			}
			prev := tabs.TabID((int(m.tabBar.Active()) - 1 + int(tabs.TabCount)) % int(tabs.TabCount))
			m.switchTab(prev)
			return m, nil
		case key.Matches(msg, appKeys.Help):
			m.showHelp = !m.showHelp
			return m, nil
		}

		switch m.tabBar.Active() {
		case tabs.TabTunnels:
			return m.handleTunnelListKeys(msg)
		case tabs.TabLogs:
			return m.handleLogKeys(msg)
		case tabs.TabSettings:
			return m.handleSettingsMsg(msg)
		}

	case tabs.SettingsSavedMsg:
		m.config.Settings = m.settingsTab.Settings()
		if err := tunnel.SaveConfig(m.config, m.configPath); err != nil {
			m.statusMsg = fmt.Sprintf("Save error: %s", err)
		} else {
			m.statusMsg = "Settings saved"
		}
		return m, nil

	case tabs.EditorSaveMsg:
		t, err := m.listTab.Editor().Tunnel()
		if err != nil {
			m.listTab.Editor().Message = err.Error()
			return m, nil
		}
		if m.listTab.Editor().EditIndex() < 0 {
			m.mgr.AddTunnel(t)
			m.statusMsg = fmt.Sprintf("Tunnel %q added", t.Name)
		} else {
			m.mgr.UpdateTunnel(m.listTab.Editor().EditIndex(), t)
			m.statusMsg = fmt.Sprintf("Tunnel %q updated", t.Name)
		}
		if err := tunnel.SaveConfig(m.config, m.configPath); err != nil {
			m.statusMsg = fmt.Sprintf("Save error: %s", err)
		}
		m.listTab.Editor().Cancel()
		m.syncTunnels()
		return m, nil

	case tabs.DeleteTunnelMsg:
		name := msg.Name
		idx := -1
		for i, t := range m.mgr.Tunnels() {
			if t.Name == name {
				idx = i
				break
			}
		}
		if idx >= 0 {
			m.mgr.RemoveTunnel(idx)
			m.statusMsg = fmt.Sprintf("Tunnel %q deleted", name)
			if err := tunnel.SaveConfig(m.config, m.configPath); err != nil {
				m.statusMsg = fmt.Sprintf("Save error: %s", err)
			}
			m.syncTunnels()
		}
		return m, nil

	case spinner.TickMsg:
		if m.loading != "" {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tunnelResultMsg:
		m.loading = ""
		m.loadingAct = ""
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("%s error (%s): %s", msg.action, msg.name, msg.err)
		} else {
			m.statusMsg = fmt.Sprintf("Tunnel %q %sed", msg.name, msg.action)
		}
		m.syncTunnels()
		return m, nil

	case stopAllResultMsg:
		m.loading = ""
		m.loadingAct = ""
		m.syncTunnels()
		m.quitting = true
		return m, tea.Quit

	case orphanScanMsg:
		if len(msg.lines) > 0 {
			m.orphanLines = msg.lines
		}
		return m, nil
	}

	return m.updateActiveTab(msg)
}

func (m Model) View() tea.View {
	if m.quitting {
		return tea.NewView("Goodbye!\n")
	}

	content := lipgloss.NewStyle().Height(m.height - 6).Render(m.activeTabView())

	status := m.statusMsg
	if m.loading != "" {
		status = m.spinner.View() + fmt.Sprintf(" %s %q...", actionLabel(m.loadingAct), m.loading)
	} else if status == "" {
		// Show stale details if selected tunnel is stale
		if m.tabBar.Active() == tabs.TabTunnels && !m.listTab.IsEditorActive() && !m.listTab.IsDeleteConfirmActive() && !m.listTab.IsFilterActive() {
			if idx := m.listTab.SelectedTunnelIndex(); idx >= 0 && idx < len(m.mgr.Tunnels()) {
				t := m.mgr.Tunnels()[idx]
				if t.Stale {
					socket := m.mgr.Client().SocketPath(t.Name)
					status = ui.StyleWarning.Render(
						fmt.Sprintf("Stale: socket %s exists but SSH process not found  │  c: cleanup", socket),
					)
				}
			}
		}
		if status == "" {
			status = m.listTab.StatusText(m.mgr.Tunnels(), m.mgr.RunningCount())
			// Append tab-specific actions
			switch m.tabBar.Active() {
			case tabs.TabTunnels:
				actions := "  │  a: add  e: edit  d: delete  /: filter"
				staleCount := 0
				for _, t := range m.mgr.Tunnels() {
					if t.Stale {
						staleCount++
					}
				}
				if staleCount > 0 {
					actions += "  c: cleanup"
				}
				if m.listTab.IsFilterActive() {
					actions += "  esc: clear"
				}
				status += actions
			case tabs.TabSettings:
				status += "  │  enter: edit  ctrl+s: save  esc: back"
			}
		}
	}
	statusBar := ui.StyleStatusBar.Render(status)
	shortcutBar := ui.StyleHelp.Render(m.activeTabShortcuts())

	// Warning bar for orphan tunnels (between content and status bar)
	warningBar := ""
	if len(m.orphanLines) > 0 {
		var wb strings.Builder
		wb.WriteString(fmt.Sprintf("⚠ %d unmanaged SSH tunnel(s):", len(m.orphanLines)))
		for _, line := range m.orphanLines {
			wb.WriteString("\n  ")
			wb.WriteString(formatOrphanLine(line))
		}
		wb.WriteString("\n  Press x to dismiss")
		warningBar = ui.StyleWarningPadded.Render(wb.String())
	}

	if m.confirm.active {
		confirm := fmt.Sprintf(
			"\n  %d tunnels are running. (k)ill and quit / (l)eave running and quit / (n)cancel\n",
			m.mgr.RunningCount(),
		)
		content += ui.StyleError.Render(confirm)
	}

	parts := []string{
		ui.StyleTitle.Render(" Tuinnel"),
		"",
		m.tabBar.View(),
		content,
	}
	if warningBar != "" {
		parts = append(parts, warningBar)
	}
	parts = append(parts, statusBar, shortcutBar)

	v := tea.NewView(strings.Join(parts, "\n"))
	v.AltScreen = true

	if m.showHelp {
		v.Content = m.helpOverlay()
	}

	return v
}

func (m *Model) switchTab(id tabs.TabID) {
	m.tabBar.SetActive(id)
	if id == tabs.TabLogs {
		name := m.logTab.SelectedTunnelName()
		if name == "" && len(m.mgr.Tunnels()) > 0 {
			name = m.mgr.Tunnels()[0].Name
		}
		if name != "" {
			m.logTab.UpdateLogs(name, m.mgr.Log(name).Lines())
		}
	}
}

func (m *Model) syncTunnels() {
	tunnels := m.mgr.Tunnels()
	m.listTab.UpdateTunnels(tunnels)
	m.logTab.UpdateTunnels(tunnels)
}

func (m Model) updateActiveTab(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.tabBar.Active() {
	case tabs.TabTunnels:
		m.listTab, cmd = m.listTab.Update(msg)
	case tabs.TabLogs:
		m.logTab, cmd = m.logTab.Update(msg)
	case tabs.TabSettings:
		m.settingsTab, cmd = m.settingsTab.Update(msg)
	}
	m.tabBar.Update(msg)
	return m, cmd
}

func (m Model) activeTabView() string {
	switch m.tabBar.Active() {
	case tabs.TabTunnels:
		return m.listTab.View()
	case tabs.TabLogs:
		return m.logTab.View()
	case tabs.TabSettings:
		return m.settingsTab.View()
	default:
		return ""
	}
}

func (m Model) activeTabShortcuts() string {
	shortcuts := ""
	switch m.tabBar.Active() {
	case tabs.TabTunnels:
		shortcuts = "↑/↓: move  enter/r: start  s: stop  R: restart  g: refresh  c: cleanup  1/2/3: tabs  ?: help  q: quit"
	case tabs.TabLogs:
		shortcuts = "↑/↓/j/k: select/scroll  1/2/3: tabs  ?: help  q: quit"
	case tabs.TabSettings:
		shortcuts = "1/2/3: tabs  ?: help  q: quit"
	}
	if m.orphanLines != nil {
		shortcuts += "  x: dismiss warning"
	}
	return shortcuts
}

func (m Model) handleTunnelListKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.listTab.IsEditorActive() || m.listTab.IsDeleteConfirmActive() || m.listTab.IsFilterActive() {
		return m.updateActiveTab(msg)
	}

	switch msg.String() {
	case "a":
		cmd := m.listTab.Editor().StartAdd()
		return m, cmd
	case "e":
		idx := m.listTab.SelectedTunnelIndex()
		if idx >= 0 {
			t := m.mgr.Tunnels()[idx]
			cmd := m.listTab.Editor().StartEdit(idx, t)
			return m, cmd
		}
		return m, nil
	case "d":
		idx := m.listTab.SelectedTunnelIndex()
		if idx >= 0 {
			t := m.mgr.Tunnels()[idx]
			if t.Running {
				m.listTab.ShowDeleteConfirm(t.Name)
			} else {
				m.mgr.RemoveTunnel(idx)
				m.statusMsg = fmt.Sprintf("Tunnel %q deleted", t.Name)
				if err := tunnel.SaveConfig(m.config, m.configPath); err != nil {
					m.statusMsg = fmt.Sprintf("Save error: %s", err)
				}
				m.syncTunnels()
			}
		}
		return m, nil
	case "/":
		cmd := m.listTab.ActivateFilter()
		return m, cmd
	case "r", "enter":
		name := m.listTab.SelectedTunnelName()
		if name != "" {
			m.loading = name
			m.loadingAct = "start"
			return m, tea.Batch(m.spinner.Tick, m.startTunnelCmd(name))
		}
		return m, nil
	case "s":
		name := m.listTab.SelectedTunnelName()
		if name != "" {
			m.loading = name
			m.loadingAct = "stop"
			return m, tea.Batch(m.spinner.Tick, m.stopTunnelCmd(name))
		}
		return m, nil
	case "R":
		name := m.listTab.SelectedTunnelName()
		if name != "" {
			m.loading = name
			m.loadingAct = "restart"
			return m, tea.Batch(m.spinner.Tick, m.restartTunnelCmd(name))
		}
		return m, nil
	case "g":
		m.mgr.Refresh()
		m.syncTunnels()
		m.statusMsg = "Status refreshed"
		return m, nil
	case "c":
		return m.cleanupStale()
	}
	return m.updateActiveTab(msg)
}

func (m Model) handleLogKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k", "down", "j":
		var cmd tea.Cmd
		m.logTab, cmd = m.logTab.Update(msg)
		name := m.logTab.SelectedTunnelName()
		if name != "" {
			m.logTab.UpdateLogs(name, m.mgr.Log(name).Lines())
		}
		return m, cmd
	}
	return m.updateActiveTab(msg)
}

func (m Model) handleSettingsMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.settingsTab, cmd = m.settingsTab.Update(msg)
	return m, cmd
}

func (m Model) cleanupStale() (tea.Model, tea.Cmd) {
	cleaned, failed := m.mgr.CleanupStale()
	m.syncTunnels()
	if cleaned > 0 && failed > 0 {
		m.statusMsg = fmt.Sprintf("Cleaned %d stale tunnel(s), %d failed", cleaned, failed)
	} else if cleaned > 0 {
		m.statusMsg = fmt.Sprintf("Cleaned %d stale tunnel(s)", cleaned)
	} else if failed > 0 {
		m.statusMsg = fmt.Sprintf("Failed to clean %d stale tunnel(s)", failed)
	} else {
		m.statusMsg = "No stale tunnels found"
	}
	return m, nil
}

func (m Model) helpOverlay() string {
	sections := []struct {
		title string
		binds [][2]string
	}{
		{
			title: "Global",
			binds: [][2]string{
				{"1/2/3", "Switch tabs"},
				{"tab/⇧+tab", "Next/prev tab"},
				{"?", "Toggle this help"},
				{"q/ctrl+c", "Quit"},
			},
		},
		{
			title: "Tunnel List",
			binds: [][2]string{
				{"↑/↓", "Move selection"},
				{"enter/r", "Start tunnel"},
				{"s", "Stop tunnel"},
				{"R", "Restart tunnel"},
				{"e", "Edit tunnel"},
				{"d", "Delete tunnel"},
				{"a", "Add tunnel"},
				{"/", "Filter by name"},
				{"g", "Refresh status"},
			{"c", "Cleanup stale tunnels"},
			},
		},
		{
			title: "Logs",
			binds: [][2]string{
				{"↑/↓/j/k", "Select tunnel / scroll"},
			},
		},
	}

	var body strings.Builder
	for _, s := range sections {
		body.WriteString(ui.StyleHelpSection.Render(s.title))
		body.WriteString("\n")
		for _, b := range s.binds {
			body.WriteString(ui.StyleHelpKey.Render(b[0]))
			body.WriteString(ui.StyleHelpDesc.Render(b[1]))
			body.WriteString("\n")
		}
	}
	body.WriteString(ui.StyleHelpClose.Render("Press ? or esc to close"))

	panel := ui.StyleHelpOverlay.Render(
		ui.StyleHelpTitle.Render("Keyboard Shortcuts") + "\n" + body.String(),
	)

	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		panel,
	)
}

func actionLabel(act string) string {
	switch act {
	case "start":
		return "Starting"
	case "stop":
		return "Stopping"
	case "restart":
		return "Restarting"
	default:
		return strings.ToUpper(act[:1]) + act[1:] + "ing"
	}
}

// formatOrphanLine extracts the key info from an SSH command line.
// Shows: forward-spec  login  [socket:path]
func formatOrphanLine(line string) string {
	fields := strings.Fields(line)
	var forward, login, socket string
	for i := 0; i < len(fields); i++ {
		f := fields[i]
		// Handle both "-L spec" and "-Lspec" forms
		if strings.HasPrefix(f, "-L") || strings.HasPrefix(f, "-R") || strings.HasPrefix(f, "-D") {
			if len(f) > 2 {
				forward = f
			} else if i+1 < len(fields) {
				forward = f + " " + fields[i+1]
			}
		} else if strings.HasPrefix(f, "-S") {
			if len(f) > 2 {
				socket = f[2:]
			} else if i+1 < len(fields) {
				socket = fields[i+1]
			}
		}
	}
	// Last field is typically user@host or a Host alias
	if len(fields) > 0 {
		login = fields[len(fields)-1]
	}
	parts := []string{}
	if forward != "" {
		parts = append(parts, forward)
	}
	if login != "" {
		parts = append(parts, login)
	}
	if socket != "" {
		parts = append(parts, "socket:"+socket)
	}
	if len(parts) == 0 {
		return line
	}
	return strings.Join(parts, "  ")
}

func (m Model) startTunnelCmd(name string) tea.Cmd {
	return func() tea.Msg {
		return tunnelResultMsg{action: "start", name: name, err: m.mgr.Start(name)}
	}
}

func (m Model) stopTunnelCmd(name string) tea.Cmd {
	return func() tea.Msg {
		return tunnelResultMsg{action: "stop", name: name, err: m.mgr.Stop(name)}
	}
}

func (m Model) restartTunnelCmd(name string) tea.Cmd {
	return func() tea.Msg {
		return tunnelResultMsg{action: "restart", name: name, err: m.mgr.Restart(name)}
	}
}

func (m Model) stopAllCmd() tea.Cmd {
	return func() tea.Msg {
		return stopAllResultMsg{err: m.mgr.StopAll()}
	}
}

// scanOrphanTunnelsCmd scans for SSH tunnel processes not managed by this program.
func (m Model) scanOrphanTunnelsCmd() tea.Cmd {
	controlDir := m.config.Settings.ControlDir
	return func() tea.Msg {
		out, err := exec.Command("ps", "-eo", "args=").Output()
		if err != nil {
			return orphanScanMsg{}
		}
		// Match SSH tunnel processes: ssh with -L, -R, or -D flags
		re := regexp.MustCompile(`\bssh\s+.*(?:-[LRD]\s+\S+)`)
		var orphans []string
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if !re.MatchString(line) {
				continue
			}
			// Skip if the process uses our control directory socket
			if strings.Contains(line, controlDir) {
				continue
			}
			orphans = append(orphans, line)
		}
		return orphanScanMsg{lines: orphans}
	}
}
