package tabs

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"

	"github.com/lululau/tuinnel/internal/tunnel"
	"github.com/lululau/tuinnel/internal/ui"
)

type SettingsModel struct {
	inputs []textinput.Model
	cursor int
	focus  int
	width  int
	height int
	Saved  bool
}

func NewSettingsModel(settings tunnel.Settings) SettingsModel {
	inputs := make([]textinput.Model, 3)

	inputs[0] = textinput.New()
	inputs[0].Placeholder = "ssh"
	inputs[0].CharLimit = 50
	inputs[0].SetValue(settings.SSHBin)

	inputs[1] = textinput.New()
	inputs[1].Placeholder = "/tmp/tuinnel"
	inputs[1].CharLimit = 100
	inputs[1].SetValue(settings.ControlDir)

	inputs[2] = textinput.New()
	inputs[2].Placeholder = "false"
	inputs[2].CharLimit = 5
	inputs[2].SetValue(fmt.Sprintf("%t", settings.KillOnExit))

	return SettingsModel{inputs: inputs, focus: -1}
}

var settingsKeys = struct {
	Up   key.Binding
	Down key.Binding
	Edit key.Binding
	Save key.Binding
	Quit key.Binding
}{
	Up:   key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "move up")),
	Down: key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "move down")),
	Edit: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "edit field")),
	Save: key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("ctrl+s", "save")),
	Quit: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
}

func (m *SettingsModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	for i := range m.inputs {
		m.inputs[i].SetWidth(w - 20)
	}
}

func (m *SettingsModel) Settings() tunnel.Settings {
	killOnExit := m.inputs[2].Value() == "true"
	return tunnel.Settings{
		SSHBin:     m.inputs[0].Value(),
		ControlDir: m.inputs[1].Value(),
		KillOnExit: killOnExit,
	}
}

func (m *SettingsModel) Update(msg tea.Msg) (SettingsModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, settingsKeys.Up):
			if m.focus >= 0 {
				m.inputs[m.focus].Blur()
				m.focus--
				if m.focus >= 0 {
					cmd = m.inputs[m.focus].Focus()
				}
			} else if m.cursor > 0 {
				m.cursor--
			}
			return *m, cmd
		case key.Matches(msg, settingsKeys.Down):
			if m.focus >= 0 {
				m.inputs[m.focus].Blur()
				m.focus++
				if m.focus < len(m.inputs) {
					cmd = m.inputs[m.focus].Focus()
				} else {
					m.focus = -1
				}
			} else if m.cursor < len(m.inputs)-1 {
				m.cursor++
			}
			return *m, cmd
		case key.Matches(msg, settingsKeys.Edit):
			m.focus = m.cursor
			cmd = m.inputs[m.focus].Focus()
			return *m, cmd
		case key.Matches(msg, settingsKeys.Save):
			m.Saved = true
			return *m, func() tea.Msg { return SettingsSavedMsg{} }
		case key.Matches(msg, settingsKeys.Quit):
			if m.focus >= 0 {
				m.inputs[m.focus].Blur()
				m.focus = -1
			}
			return *m, nil
		}
	}

	if m.focus >= 0 && m.focus < len(m.inputs) {
		m.inputs[m.focus], cmd = m.inputs[m.focus].Update(msg)
	}

	return *m, cmd
}

func (m SettingsModel) View() string {
	labels := []string{"SSH Binary", "Control Socket Dir", "Kill on Exit (true/false)"}

	var s strings.Builder
	s.WriteString(ui.StyleTitle.Render("Settings") + "\n\n")

	for i, label := range labels {
		cursor := " "
		if i == m.cursor && m.focus < 0 {
			cursor = ">"
		}
		style := ui.StyleFocused
		if m.focus == i {
			style = ui.StyleInput
		}
		s.WriteString(fmt.Sprintf("%s %s: %s\n", cursor, label, style.Render(m.inputs[i].View())))
	}

	return s.String()
}

type SettingsSavedMsg struct{}
