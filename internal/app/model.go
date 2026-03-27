package app

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/key"

	"github.com/ssh-tun-tui/internal/tunnel"
	"github.com/ssh-tun-tui/internal/ui"
	"github.com/ssh-tun-tui/internal/ui/tabs"
)

var appKeys = struct {
	Tab1    key.Binding
	Tab2    key.Binding
	Tab3    key.Binding
	Tab4    key.Binding
	TabNext key.Binding
	TabPrev key.Binding
	Help    key.Binding
	Quit    key.Binding
}{
	Tab1:    key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "tunnels")),
	Tab2:    key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "logs")),
	Tab3:    key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "settings")),
	Tab4:    key.NewBinding(key.WithKeys("4"), key.WithHelp("4", "editor")),
	TabNext: key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next tab")),
	TabPrev: key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("⇧+tab", "prev tab")),
	Help:    key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}

type confirmQuit struct {
	active bool
}

type Model struct {
	mgr         *tunnel.Manager
	config      *tunnel.Config
	configPath  string
	tabBar      tabs.TabBar
	listTab     tabs.TunnelListModel
	logTab      tabs.LogModel
	settingsTab tabs.SettingsModel
	editorTab   tabs.EditorModel
	width       int
	height      int
	confirm     confirmQuit
	statusMsg   string
	quitting    bool
}

func NewModel(cfg *tunnel.Config, configPath string) Model {
	mgr := tunnel.NewManager(cfg)
	mgr.Refresh()

	m := Model{
		mgr:         mgr,
		config:      cfg,
		configPath:  configPath,
		tabBar:      tabs.NewTabBar(),
		listTab:     tabs.NewTunnelListModel(),
		logTab:      tabs.NewLogModel(),
		settingsTab: tabs.NewSettingsModel(cfg.Settings),
		editorTab:   tabs.NewEditorModel(),
	}
	m.syncTunnels()
	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.tabBar.SetWidth(msg.Width)
		contentHeight := msg.Height - 4
		m.listTab.SetSize(msg.Width, contentHeight)
		m.logTab.SetSize(msg.Width, contentHeight)
		m.settingsTab.SetSize(msg.Width, contentHeight)
		m.editorTab.SetSize(msg.Width, contentHeight)
		return m, nil

	case tea.KeyPressMsg:
		if m.confirm.active {
			switch msg.String() {
			case "y":
				m.quitting = true
				if m.config.Settings.KillOnExit {
					_ = m.mgr.StopAll()
				}
				return m, tea.Quit
			case "n", "esc":
				m.confirm.active = false
				return m, nil
			}
			return m, nil
		}

		// Don't intercept global keys when editor is in edit mode
		if m.editorTab.IsEditing() && m.tabBar.Active() == tabs.TabEditor {
			return m.handleEditorMsg(msg)
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
		case key.Matches(msg, appKeys.Tab4):
			m.switchTab(tabs.TabEditor)
			return m, nil
		case key.Matches(msg, appKeys.TabNext):
			next := tabs.TabID((int(m.tabBar.Active()) + 1) % int(tabs.TabCount))
			m.switchTab(next)
			return m, nil
		case key.Matches(msg, appKeys.TabPrev):
			prev := tabs.TabID((int(m.tabBar.Active()) - 1 + int(tabs.TabCount)) % int(tabs.TabCount))
			m.switchTab(prev)
			return m, nil
		}

		switch m.tabBar.Active() {
		case tabs.TabTunnels:
			return m.handleTunnelListKeys(msg)
		case tabs.TabLogs:
			return m.handleLogKeys(msg)
		case tabs.TabSettings:
			return m.handleSettingsMsg(msg)
		case tabs.TabEditor:
			return m.handleEditorMsg(msg)
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
		t, err := m.editorTab.Tunnel()
		if err != nil {
			m.editorTab.Message = err.Error()
			return m, nil
		}
		if m.editorTab.EditIndex() < 0 {
			m.mgr.AddTunnel(t)
			m.statusMsg = fmt.Sprintf("Tunnel %q added", t.Name)
		} else {
			m.mgr.UpdateTunnel(m.editorTab.EditIndex(), t)
			m.statusMsg = fmt.Sprintf("Tunnel %q updated", t.Name)
		}
		if err := tunnel.SaveConfig(m.config, m.configPath); err != nil {
			m.statusMsg = fmt.Sprintf("Save error: %s", err)
		}
		m.editorTab.Cancel()
		m.syncTunnels()
		return m, nil
	}

	return m.updateActiveTab(msg)
}

func (m Model) View() tea.View {
	if m.quitting {
		return tea.NewView("Goodbye!\n")
	}

	content := m.activeTabView()

	status := m.statusMsg
	if status == "" {
		status = m.listTab.StatusText(m.mgr.Tunnels(), m.mgr.RunningCount())
	}
	statusBar := ui.StyleStatusBar.Render(status)

	if m.confirm.active {
		confirm := fmt.Sprintf(
			"\n  %d tunnels are running. Quit and %s them? [y/n]\n",
			m.mgr.RunningCount(),
			map[bool]string{true: "kill", false: "leave"}[m.config.Settings.KillOnExit],
		)
		content += ui.StyleError.Render(confirm)
	}

	v := tea.NewView(fmt.Sprintf("%s\n%s\n%s\n%s",
		ui.StyleTitle.Render(" ssh-tun-tui"),
		m.tabBar.View(),
		content,
		statusBar,
	))
	v.AltScreen = true
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
	case tabs.TabEditor:
		m.editorTab, cmd = m.editorTab.Update(msg)
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
	case tabs.TabEditor:
		return m.editorTab.View()
	default:
		return ""
	}
}

func (m Model) handleTunnelListKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "r", "enter":
		name := m.listTab.SelectedTunnelName()
		if name != "" {
			if err := m.mgr.Start(name); err != nil {
				m.statusMsg = fmt.Sprintf("Start error: %s", err)
			} else {
				m.statusMsg = fmt.Sprintf("Tunnel %q started", name)
			}
			m.syncTunnels()
		}
		return m, nil
	case "s":
		name := m.listTab.SelectedTunnelName()
		if name != "" {
			if err := m.mgr.Stop(name); err != nil {
				m.statusMsg = fmt.Sprintf("Stop error: %s", err)
			} else {
				m.statusMsg = fmt.Sprintf("Tunnel %q stopped", name)
			}
			m.syncTunnels()
		}
		return m, nil
	case "R":
		name := m.listTab.SelectedTunnelName()
		if name != "" {
			if err := m.mgr.Restart(name); err != nil {
				m.statusMsg = fmt.Sprintf("Restart error: %s", err)
			} else {
				m.statusMsg = fmt.Sprintf("Tunnel %q restarted", name)
			}
			m.syncTunnels()
		}
		return m, nil
	case "g":
		m.mgr.Refresh()
		m.syncTunnels()
		m.statusMsg = "Status refreshed"
		return m, nil
	case "e":
		name := m.listTab.SelectedTunnelName()
		idx := -1
		for i, t := range m.mgr.Tunnels() {
			if t.Name == name {
				idx = i
				break
			}
		}
		if idx >= 0 {
			cmd := m.editorTab.StartEdit(idx, m.mgr.Tunnels()[idx])
			m.switchTab(tabs.TabEditor)
			return m, cmd
		}
		return m, nil
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

func (m Model) handleEditorMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.editorTab, cmd = m.editorTab.Update(msg)
	return m, cmd
}
