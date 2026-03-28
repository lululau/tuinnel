# Modal Editor, Delete, and Filter Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove the Editor tab, replace with modal editor overlaid on Tunnels list, add delete with confirmation, add name filtering via `/`.

**Architecture:** Embed `EditorModel` inside `TunnelListModel`. The list tab manages three overlay states: editor modal, delete confirmation modal, and filter input. App model routes editor save messages to the embedded editor. Tab count drops from 4 to 3.

**Tech Stack:** Go, bubbletea v2, bubbles v2 (table, textinput), lipgloss v2

---

## File Structure

| File | Responsibility |
|------|---------------|
| `internal/ui/tabs/tablist.go` | Tab definitions (remove TabEditor, TabCount=3) |
| `internal/ui/tabs/tunnel_list.go` | Tunnel list + embedded editor modal + delete confirm + filter input |
| `internal/ui/tabs/editor.go` | Editor form logic (unchanged except View returns form content only) |
| `internal/ui/styles.go` | Add `StyleModal` shared border style |
| `internal/app/model.go` | Remove editorTab field, route through listTab, update help panel |

---

### Task 1: Remove TabEditor from tab list

**Files:**
- Modify: `internal/ui/tabs/tablist.go`

- [ ] **Step 1: Remove TabEditor constant and update TabCount**

In `tablist.go`, change the constants to:

```go
const (
	TabTunnels TabID = iota
	TabLogs
	TabSettings
	TabCount
)
```

Remove the `TabEditor` case from the `String()` method:

```go
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
```

Update `AllTabs()`:

```go
func AllTabs() []TabID {
	return []TabID{TabTunnels, TabLogs, TabSettings}
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: errors in `app/model.go` (references to `TabEditor`) — these are fixed in later tasks.

- [ ] **Step 3: Commit**

```bash
git add internal/ui/tabs/tablist.go
git commit -m "refactor: remove TabEditor from tab list"
```

---

### Task 2: Add StyleModal to styles

**Files:**
- Modify: `internal/ui/styles.go`

- [ ] **Step 1: Add StyleModal**

Add after `StyleHelpClose`:

```go
StyleModal = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorCyan).
		Padding(1, 3)
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: PASS (new style is unused but valid)

- [ ] **Step 3: Commit**

```bash
git add internal/ui/styles.go
git commit -m "style: add StyleModal for editor/delete confirm modals"
```

---

### Task 3: Refactor EditorModel.View for modal use

**Files:**
- Modify: `internal/ui/tabs/editor.go`

- [ ] **Step 1: Simplify View() to return form content only**

Replace the `View()` method to return just the form content (no full-screen wrapping). Remove the idle-mode "Tunnel Editor" title and the bottom help line referencing `a`/`ctrl+d` (those are now handled by the list tab):

```go
func (m EditorModel) View() string {
	labels := []string{"Name", "Type", "Local Port", "Remote Host", "Remote Port", "Login", "Group"}

	var title string
	switch m.mode {
	case editorAdd:
		title = ui.StyleTitle.Render("Add New Tunnel")
	case editorEdit:
		title = ui.StyleTitle.Render("Edit Tunnel")
	default:
		return ""
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
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/ui/tabs/editor.go
git commit -m "refactor: simplify EditorModel.View for modal rendering"
```

---

### Task 4: Add filter, delete confirm, and editor modal to TunnelListModel

This is the core task. `TunnelListModel` gains:
- `editor EditorModel` — embedded editor
- `filterInput textinput.Model` — filter text field
- `filterActive bool` — whether filter input is focused
- `deleteConfirm struct{ active bool; tunnelName string }` — delete confirmation state
- `allTunnels []tunnel.Tunnel` — full unfiltered list
- `filteredIndices []int` — indices into allTunnels that match filter

**Files:**
- Modify: `internal/ui/tabs/tunnel_list.go`

- [ ] **Step 1: Add new fields to TunnelListModel**

Replace the struct and imports:

```go
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
	table          table.Model
	editor         EditorModel
	filterInput    textinput.Model
	filterActive   bool
	deleteConfirm  deleteConfirm
	allTunnels     []tunnel.Tunnel
	filteredIndices []int
	width          int
	height         int
}
```

- [ ] **Step 2: Update NewTunnelListModel to initialize new fields**

Replace `NewTunnelListModel`:

```go
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
	fi.Width = 40

	return TunnelListModel{
		table:       t,
		editor:      NewEditorModel(),
		filterInput: fi,
	}
}
```

- [ ] **Step 3: Update SetSize to also resize editor**

Replace `SetSize`:

```go
func (m *TunnelListModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.table.SetWidth(w)
	m.table.SetHeight(h - 4)
	m.editor.SetSize(w, h)
	m.filterInput.Width = w - 4
}
```

- [ ] **Step 4: Add filter methods and update UpdateTunnels**

Replace `UpdateTunnels` and add filter helpers:

```go
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

func (m *TunnelListModel) FilteredCount() int {
	return len(m.filteredIndices)
}
```

- [ ] **Step 5: Add SelectedTunnelIndex that respects filter**

Add after `SelectedTunnelName`:

```go
func (m *TunnelListModel) SelectedTunnelIndex() int {
	row := m.table.SelectedRow()
	if row == nil {
		return -1
	}
	tableRow := m.table.Cursor()
	if tableRow < 0 || tableRow >= len(m.filteredIndices) {
		return -1
	}
	return m.filteredIndices[tableRow]
}
```

- [ ] **Step 6: Add modal state accessors**

```go
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
```

- [ ] **Step 7: Update Update to handle modal/filter states**

Replace the `Update` method:

```go
func (m *TunnelListModel) Update(msg tea.Msg) (TunnelListModel, tea.Cmd) {
	var cmd tea.Cmd

	// Editor modal takes priority over everything
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
		// Only pass non-filter keys to table when filter is active
		if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
			switch keyMsg.String() {
			case "up", "down", "j", "k", "r", "s", "R", "g", "e", "d", "a":
				return *m, nil // block list keys while filter input is focused
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
```

- [ ] **Step 8: Update View to render modals and filter**

Replace `View()`:

```go
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
```

- [ ] **Step 9: Update StatusText to show filter and actions**

Replace `StatusText`:

```go
func (m *TunnelListModel) StatusText(tunnels []tunnel.Tunnel, runningCount int) string {
	total := len(tunnels)
	visible := m.FilteredCount()

	left := fmt.Sprintf("%d/%d tunnels running", runningCount, total)
	if m.filterActive || m.filterInput.Value() != "" {
		left = fmt.Sprintf("%d/%d tunnels visible", visible, total)
	}

	actions := "a: add  e: edit  d: delete  /: filter"
	if m.filterActive || m.filterInput.Value() != "" {
		actions += "  esc: clear"
	}

	return fmt.Sprintf("%s  │  %s", left, actions)
}
```

- [ ] **Step 10: Add DeleteTunnelMsg**

Add at the bottom of the file:

```go
type DeleteTunnelMsg struct {
	Name string
}
```

- [ ] **Step 11: Add missing import**

Add `"charm.land/bubbles/v2/textinput"` to imports (already included in Step 1).

- [ ] **Step 12: Verify build**

Run: `go build ./...`
Expected: errors in `app/model.go` (references to old editorTab) — fixed in Task 5.

- [ ] **Step 13: Commit**

```bash
git add internal/ui/tabs/tunnel_list.go
git commit -m "feat: add modal editor, delete confirm, and filter to TunnelListModel"
```

---

### Task 5: Update app model to use embedded editor and new features

**Files:**
- Modify: `internal/app/model.go`

- [ ] **Step 1: Remove editorTab field and Tab4 key from Model**

Remove `editorTab tabs.EditorModel` from the `Model` struct.
Remove `Tab4` from `appKeys`.

The struct becomes:

```go
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
}
```

The keys become:

```go
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
```

- [ ] **Step 2: Update NewModel**

Remove `editorTab: tabs.NewEditorModel()` from the initialization:

```go
func NewModel(cfg *tunnel.Config, configPath string) Model {
	mgr := tunnel.NewManager(cfg)

	m := Model{
		mgr:         mgr,
		config:      cfg,
		configPath:  configPath,
		tabBar:      tabs.NewTabBar(),
		listTab:     tabs.NewTunnelListModel(),
		logTab:      tabs.NewLogModel(),
		settingsTab: tabs.NewSettingsModel(cfg.Settings),
	}
	m.syncTunnels()
	return m
}
```

- [ ] **Step 3: Update WindowSizeMsg handler**

Remove `m.editorTab.SetSize(msg.Width, contentHeight)`:

```go
case tea.WindowSizeMsg:
	m.width = msg.Width
	m.height = msg.Height
	m.tabBar.SetWidth(msg.Width)
	contentHeight := msg.Height - 4
	m.listTab.SetSize(msg.Width, contentHeight)
	m.logTab.SetSize(msg.Width, contentHeight)
	m.settingsTab.SetSize(msg.Width, contentHeight)
	return m, nil
```

- [ ] **Step 4: Remove editorTab global key interception**

Remove this block from the KeyPressMsg handler:

```go
// Don't intercept global keys when editor is in edit mode
if m.editorTab.IsEditing() && m.tabBar.Active() == tabs.TabEditor {
	return m.handleEditorMsg(msg)
}
```

- [ ] **Step 5: Remove Tab4 case from switch**

Remove from the global key switch:

```go
case key.Matches(msg, appKeys.Tab4):
	m.switchTab(tabs.TabEditor)
	return m, nil
```

- [ ] **Step 6: Remove TabEditor case from tab routing**

In the `switch m.tabBar.Active()` block, remove:

```go
case tabs.TabEditor:
	return m.handleEditorMsg(msg)
```

- [ ] **Step 7: Update EditorSaveMsg handler to use listTab.editor**

Replace the `EditorSaveMsg` case:

```go
case tabs.EditorSaveMsg:
	t, err := m.listTab.editor.Tunnel()
	if err != nil {
		m.listTab.editor.Message = err.Error()
		return m, nil
	}
	if m.listTab.editor.EditIndex() < 0 {
		m.mgr.AddTunnel(t)
		m.statusMsg = fmt.Sprintf("Tunnel %q added", t.Name)
	} else {
		m.mgr.UpdateTunnel(m.listTab.editor.EditIndex(), t)
		m.statusMsg = fmt.Sprintf("Tunnel %q updated", t.Name)
	}
	if err := tunnel.SaveConfig(m.config, m.configPath); err != nil {
		m.statusMsg = fmt.Sprintf("Save error: %s", err)
	}
	m.listTab.editor.Cancel()
	m.syncTunnels()
	return m, nil
```

- [ ] **Step 8: Add DeleteTunnelMsg handler**

Add after the `EditorSaveMsg` case:

```go
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
```

- [ ] **Step 9: Update handleTunnelListKeys for a, e, d, /**

Replace `handleTunnelListKeys`:

```go
func (m Model) handleTunnelListKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Block list keys when editor modal, delete confirm, or filter input is active
	if m.listTab.IsEditorActive() || m.listTab.IsDeleteConfirmActive() || m.listTab.IsFilterActive() {
		return m.updateActiveTab(msg)
	}

	switch msg.String() {
	case "a":
		cmd := m.listTab.editor.StartAdd()
		return m, cmd
	case "e":
		idx := m.listTab.SelectedTunnelIndex()
		if idx >= 0 {
			t := m.mgr.Tunnels()[idx]
			cmd := m.listTab.editor.StartEdit(idx, t)
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
		m.listTab.filterActive = true
		cmd := m.listTab.filterInput.Focus()
		return m, cmd
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
	}
	return m.updateActiveTab(msg)
}
```

- [ ] **Step 10: Remove handleEditorMsg method**

Delete the `handleEditorMsg` method entirely.

- [ ] **Step 11: Remove TabEditor from updateActiveTab and activeTabView**

In `updateActiveTab`, remove the `case tabs.TabEditor` branch.
In `activeTabView`, remove the `case tabs.TabEditor` branch.

- [ ] **Step 12: Update help panel**

Replace the `helpOverlay` method's sections:

```go
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
```

- [ ] **Step 13: Remove unused imports**

Ensure `strings` and `lipgloss` are still used (they are — by `helpOverlay`). Remove any unused imports.

- [ ] **Step 14: Verify build**

Run: `go build ./...`
Expected: PASS

- [ ] **Step 15: Commit**

```bash
git add internal/app/model.go
git commit -m "feat: wire modal editor, delete confirm, and filter into app model"
```

---

### Task 6: Run tests and verify

- [ ] **Step 1: Run all tests**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 2: Build and manual smoke test**

Run: `go build -o tuinnel . && ./tuinnel`

Verify:
- 3 tabs (Tunnels, Logs, Settings)
- Press `a` → modal editor appears centered
- Press `e` on a tunnel → modal editor with fields populated
- Press `d` on a running tunnel → delete confirmation modal
- Press `d` on a stopped tunnel → immediate delete
- Press `/` → filter input appears below table
- Type in filter → table updates in real-time
- Press `esc` in filter → clears and exits
- Status bar shows actions hints
- Press `?` → help shows updated shortcuts (no Editor section)
