# Modal Editor, Delete, and Filter Design

## Overview

Remove the Editor tab and replace it with a modal editor overlaid on the Tunnels list tab. Add tunnel deletion with confirmation, and name-based filtering via `/`.

## Tab Changes

Remove `TabEditor`. Tabs become 3: Tunnels, Logs, Settings.

- `tablist.go`: remove `TabEditor` constant, `TabCount` = 3
- `app/model.go`: remove `editorTab` field, embed `EditorModel` inside `TunnelListModel`
- Remove global key `4` and all `TabEditor` routing logic

## Modal Editor

Embed `EditorModel` in `TunnelListModel`. When user presses `a` (add) or `e` (edit), the editor form appears as a centered modal over the tunnel list.

### State

```
List normal → press a/e → modal opens → save/cancel → back to list
```

### Rendering

- Use `lipgloss.Place` to center the editor form over the full terminal area
- Round border style consistent with help overlay (`StyleHelpOverlay` border)
- Editor form shows all 7 fields: Name, Type, Local Port, Remote Host, Remote Port, Login, Group

### Key Interception

- When modal is open, all keypresses route to `EditorModel.Update()`
- List does not respond to any keys while modal is open
- `esc` closes modal (discard), `ctrl+s` saves, `tab` moves to next field, `enter` moves to next field (save on last)

### Message Flow

- Save sends `EditorSaveMsg` (unchanged from current implementation)
- App model handles `EditorSaveMsg`: validate, add/update tunnel, save config, close modal, sync list

## Delete

Press `d` on selected tunnel to delete.

### Flow

- If tunnel is not running: delete immediately, save config, sync list
- If tunnel is running: show confirmation modal "Tunnel 'xxx' is running. Delete anyway? (y/n)"
  - `y`: stop tunnel, delete, save config, sync list
  - `n` or `esc`: cancel, return to list

### Confirmation Rendering

- Centered modal with round border (same style as help/editor)
- Two lines: message + y/n prompt

## Filter

Press `/` to enter filter mode. A text input appears below the table for typing a filter query.

### Flow

```
List normal → press / → filter input appears below table → type to filter → esc to clear and exit
```

### Behavior

- Filter matches tunnel names by case-insensitive substring
- Filtering is real-time as user types
- Status bar shows visible/total count: `"2/5 tunnels visible"`
- Actions (`e`, `d`, `r`, `s`) operate on visible (filtered) tunnels
- Filter input has its own cursor; list keybindings (`j`/`k`, `enter`, `r`, `s`, `R`, `g`, `e`, `d`) are disabled while filter input is focused
- `enter` in filter input blurs it (exit filter mode but keep filter active)
- `esc` clears filter and exits filter mode
- While filter is active but input is blurred, list keybindings work normally on the filtered view

### Rendering

- Filter input rendered below the table, above the status bar
- Uses existing `StyleInput` for the input field
- Label: `/` prefix before the input

## Status Bar

Tunnels tab status bar dynamically shows available actions:

```
Normal:     "3/5 tunnels running  │  a: add  e: edit  d: delete  /: filter"
Filtering:  "2/5 tunnels visible  │  a: add  e: edit  d: delete  /: filter  esc: clear"
```

## Help Panel Update

Help panel "Tunnel List" section updates to:

```
Tunnel List
  ↑/↓          Move selection
  enter/r      Start tunnel
  s            Stop tunnel
  R            Restart tunnel
  e            Edit tunnel
  d            Delete tunnel
  a            Add tunnel
  /            Filter by name
  g            Refresh status
```

Remove "Editor" section from help panel entirely.

## Files Changed

| File | Change |
|------|--------|
| `internal/ui/tabs/tablist.go` | Remove `TabEditor`, `TabCount` = 3 |
| `internal/ui/tabs/tunnel_list.go` | Embed `EditorModel`, add filter state, add delete confirm state, update `View()` for modal/filter rendering |
| `internal/ui/tabs/editor.go` | Remove tab-specific `View()` wrapping, make `View()` return form content only (no full-screen layout) |
| `internal/app/model.go` | Remove `editorTab` field, route editor/delete/filter through `listTab`, update tab count references, update help panel |
| `internal/ui/styles.go` | Add `StyleModal` for modal border style (shared by editor, delete confirm) |
