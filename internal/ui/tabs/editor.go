package tabs

import (
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"

	"github.com/lululau/tuinnel/internal/tunnel"
	"github.com/lululau/tuinnel/internal/ui"
)

type editorMode int

const (
	editorIdle editorMode = iota
	editorAdd
	editorEdit
)

type EditorModel struct {
	mode    editorMode
	editIdx int
	inputs  []textinput.Model
	cursor  int
	focus   int
	width   int
	height  int
	Message string
}

func NewEditorModel() EditorModel {
	inputs := make([]textinput.Model, 7)
	fields := []struct {
		label string
		limit int
	}{
		{"Name", 30}, {"Type (local/remote/dynamic)", 10},
		{"Local Port", 7}, {"Remote Host", 30},
		{"Remote Port", 7}, {"Login (user@host)", 50},
		{"Group", 20},
	}
	for i, f := range fields {
		inputs[i] = textinput.New()
		inputs[i].Placeholder = f.label
		inputs[i].CharLimit = f.limit
	}
	return EditorModel{inputs: inputs, mode: editorIdle, editIdx: -1, focus: -1}
}

var editorKeys = struct {
	Add    key.Binding
	Save   key.Binding
	Delete key.Binding
	Cancel key.Binding
}{
	Add:    key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add new")),
	Save:   key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("ctrl+s", "save")),
	Delete: key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("ctrl+d", "delete")),
	Cancel: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
}

func (m *EditorModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	inputWidth := w - 25
	if inputWidth > 28 {
		inputWidth = 28
	}
	for i := range m.inputs {
		m.inputs[i].SetWidth(inputWidth)
	}
}

func (m *EditorModel) StartAdd() tea.Cmd {
	m.mode = editorAdd
	m.editIdx = -1
	m.Message = ""
	for i := range m.inputs {
		m.inputs[i].SetValue("")
	}
	m.focus = 0
	return m.inputs[0].Focus()
}

func (m *EditorModel) StartEdit(idx int, t tunnel.Tunnel) tea.Cmd {
	m.mode = editorEdit
	m.editIdx = idx
	m.Message = ""
	m.inputs[0].SetValue(t.Name)
	m.inputs[1].SetValue(string(t.Type))
	m.inputs[2].SetValue(strconv.Itoa(t.LocalPort))
	m.inputs[3].SetValue(t.RemoteHost)
	m.inputs[4].SetValue(strconv.Itoa(t.RemotePort))
	m.inputs[5].SetValue(t.Login)
	m.inputs[6].SetValue(t.Group)
	m.focus = 0
	return m.inputs[0].Focus()
}

func (m *EditorModel) Cancel() {
	m.mode = editorIdle
	m.editIdx = -1
	m.Message = ""
	m.blurAll()
}

func (m *EditorModel) blurAll() {
	for i := range m.inputs {
		m.inputs[i].Blur()
	}
	m.focus = -1
}

func (m *EditorModel) Tunnel() (tunnel.Tunnel, error) {
	lp, _ := strconv.Atoi(m.inputs[2].Value())
	rp, _ := strconv.Atoi(m.inputs[4].Value())
	t := tunnel.Tunnel{
		Name:       m.inputs[0].Value(),
		Type:       tunnel.TunnelType(m.inputs[1].Value()),
		LocalPort:  lp,
		RemoteHost: m.inputs[3].Value(),
		RemotePort: rp,
		Login:      m.inputs[5].Value(),
		Group:      m.inputs[6].Value(),
	}
	return t, t.Validate()
}

func (m *EditorModel) IsEditing() bool {
	return m.mode != editorIdle
}

func (m *EditorModel) EditIndex() int {
	return m.editIdx
}

func (m *EditorModel) Update(msg tea.Msg) (EditorModel, tea.Cmd) {
	var cmd tea.Cmd

	if m.mode == editorIdle {
		switch msg := msg.(type) {
		case tea.KeyPressMsg:
			if key.Matches(msg, editorKeys.Add) {
				cmd = m.StartAdd()
				return *m, cmd
			}
		}
		return *m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, editorKeys.Cancel):
			m.Cancel()
			return *m, nil
		case key.Matches(msg, editorKeys.Save):
			return *m, func() tea.Msg { return EditorSaveMsg{} }
		case msg.String() == "tab":
			if m.focus >= 0 && m.focus < len(m.inputs)-1 {
				m.inputs[m.focus].Blur()
				m.focus++
				cmd = m.inputs[m.focus].Focus()
				return *m, cmd
			}
		case msg.String() == "shift+tab":
			if m.focus > 0 {
				m.inputs[m.focus].Blur()
				m.focus--
				cmd = m.inputs[m.focus].Focus()
				return *m, cmd
			}
		case msg.String() == "enter":
			if m.focus >= 0 && m.focus < len(m.inputs)-1 {
				m.inputs[m.focus].Blur()
				m.focus++
				cmd = m.inputs[m.focus].Focus()
				return *m, cmd
			} else if m.focus == len(m.inputs)-1 {
				return *m, func() tea.Msg { return EditorSaveMsg{} }
			}
		case msg.String() == "up":
			if m.focus > 0 {
				m.inputs[m.focus].Blur()
				m.focus--
				cmd = m.inputs[m.focus].Focus()
				return *m, cmd
			}
		}
	}

	if m.focus >= 0 && m.focus < len(m.inputs) {
		m.inputs[m.focus], cmd = m.inputs[m.focus].Update(msg)
	}

	return *m, cmd
}

func (m EditorModel) View() string {
	if m.mode == editorIdle {
		return ""
	}

	labels := []string{"Name", "Type", "Local Port", "Remote Host", "Remote Port", "Login", "Group"}

	var title string
	switch m.mode {
	case editorAdd:
		title = ui.StyleTitle.Render("Add New Tunnel")
	case editorEdit:
		title = ui.StyleTitle.Render("Edit Tunnel")
	}

	var s strings.Builder
	s.WriteString(title + "\n\n")

	for i, label := range labels {
		cursor := " "
		if i == m.focus {
			cursor = ">"
		}
		style := ui.StyleFocused
		if m.focus == i {
			style = ui.StyleInput
		}
		s.WriteString(fmt.Sprintf("%s %s: %s\n", cursor, label, style.Render(m.inputs[i].View())))
	}

	if m.Message != "" {
		s.WriteString("\n" + ui.StyleError.Render(m.Message))
	}

	s.WriteString("\n" + ui.StyleHelp.Render("ctrl+s: save • esc: cancel"))

	return s.String()
}

type EditorSaveMsg struct{}
