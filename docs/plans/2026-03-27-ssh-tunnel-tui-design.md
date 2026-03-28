# SSH Tunnel TUI Manager — Design Document

## Overview

A terminal-based SSH tunnel manager built with Go, bubbletea, and bubbles. Provides a tabbed TUI interface for managing SSH tunnels with configuration stored in TOML.

## Architecture

Single-process bubbletea application. One Go binary manages both the TUI and SSH child processes.

```
main → App Model → Tab Models → SSH Manager
```

### Project Structure

```
tuinnel/
├── main.go                    # Entry point, bubbletea init
├── go.mod / go.sum
├── internal/
│   ├── app/
│   │   └── model.go           # Top-level App Model, tab switching
│   ├── tunnel/
│   │   ├── tunnel.go          # Tunnel struct definition
│   │   ├── manager.go         # SSH tunnel lifecycle (start/stop/check)
│   │   └── config.go          # TOML config read/write
│   ├── ui/
│   │   ├── tabs/
│   │   │   ├── tablist.go     # Tab bar component
│   │   │   ├── tunnel_list.go # Tab 1: tunnel list
│   │   │   ├── logs.go        # Tab 2: log panel
│   │   │   ├── settings.go    # Tab 3: settings
│   │   │   └── editor.go      # Tab 4: tunnel editor
│   │   └── styles.go          # Global lipgloss theme
│   └── ssh/
│       └── client.go          # SSH command wrapper (ControlMaster ops)
```

### Dependencies

- `charmbracelet/bubbletea` — TUI framework
- `charmbracelet/bubbles` — prebuilt components (table, viewport, textinput, textarea)
- `charmbracelet/lipgloss` — styling
- `BurntSushi/toml` — TOML parsing

## Data Model

### Tunnel Struct

```go
type TunnelType string

const (
    TunnelLocal   TunnelType = "local"    // -L
    TunnelRemote  TunnelType = "remote"   // -R
    TunnelDynamic TunnelType = "dynamic"  // -D
)

type Tunnel struct {
    Name       string     `toml:"name"`
    Type       TunnelType `toml:"type"`
    LocalPort  int        `toml:"local_port"`
    RemoteHost string     `toml:"remote_host"`
    RemotePort int        `toml:"remote_port"`
    Login      string     `toml:"login"`
    Group      string     `toml:"group"`

    // Runtime state (not persisted)
    Running    bool       `toml:"-"`
    PID        int        `toml:"-"`
}
```

### TOML Configuration

Path: `~/.config/tuinnel/config.toml`

```toml
[settings]
ssh_bin = "ssh"
control_dir = "/tmp/tuinnel"
kill_on_exit = false

[[tunnels]]
name = "dev-db"
type = "local"
local_port = 3307
remote_host = "localhost"
remote_port = 3306
login = "deploy@db-server"
group = "dev"

[[tunnels]]
name = "staging-api"
type = "local"
local_port = 8080
remote_host = "localhost"
remote_port = 80
login = "deploy@api-staging"
group = "staging"

[[tunnels]]
name = "socks-proxy"
type = "dynamic"
local_port = 1080
login = "user@jump-host"
```

## TUI Layout

### Overall Structure

```
┌─ tuinnel ────────────────────────────────────┐
│ [Tunnels] [Logs] [Settings] [Editor]    ?=Help   │
├──────────────────────────────────────────────────┤
│                                                   │
│              (Active Tab Content)                 │
│                                                   │
├──────────────────────────────────────────────────┤
│ 1/4 tunnels running │ group: dev │ j/k: move     │
└───────────────────────────────────────────────────┘
```

### Tab 1: Tunnel List (Default)

Table view with columns: State, Name, Type, LPort, RHost, RPort, Login.

- Green bullet = running, gray bullet = stopped
- Shortcuts: `Enter`/`r` start, `k` stop, `R` restart, `g` refresh, `e` edit in editor tab, `/` filter/search

### Tab 2: Log Panel

Split view: compact tunnel list on left, selected tunnel's SSH stdout/stderr on right (scrollable viewport).

### Tab 3: Settings

Form for global settings: SSH binary path, control socket directory, exit behavior. Save writes to config.toml.

### Tab 4: Tunnel Editor

Form for add/edit/delete tunnel configurations. Fields: Name, Type, LocalPort, RemoteHost, RemotePort, Login, Group. Saves to config.toml and refreshes tunnel list.

### Global Shortcuts

- `1-4` or `Tab`/`Shift+Tab` — switch tabs
- `q` — quit (confirm if tunnels running)
- `?` — help dialog

## SSH Management

### ControlMaster Operations

Three core operations via SSH ControlMaster:

1. **Start**: `ssh -M -f -N -T -S {socket} -L/-R/-D {forward} {login}`
2. **Stop**: `ssh -S {socket} -O exit {login}`
3. **Check**: `ssh -S {socket} -O check {login}` → bool

Control socket path: `{control_dir}/{tunnel_name}`

### Manager (`internal/tunnel/manager.go`)

- Maintains `map[string]*exec.Cmd` for SSH process tracking
- Start: check socket existence, create dir, exec SSH, capture stdout/stderr to log buffer
- Stop: send `-O exit`, wait for exit, clean socket
- Check: iterate all tunnels on startup and manual refresh, sync running state
- Group ops: `StartGroup(group)`, `StopGroup(group)`, `StopAll()`

### Log Collection

- Each SSH process stdout/stderr captured via `os.Pipe()`
- Goroutine reads line-by-line into ring buffer (1000 lines)
- Log panel displays via viewport

### Error Handling

- SSH connection failure: status bar error message, tunnel marked "error" (red)
- Port conflict: check local_port availability before start
- Config validation: verify TOML on startup, clear messages for missing fields
- Exit confirmation: `q` shows confirm dialog when tunnels are running

## Supported Tunnel Types

| Type | Flag | Forward Spec |
|------|------|-------------|
| Local | `-L` | `local_port:remote_host:remote_port` |
| Remote | `-R` | `remote_port:remote_host:local_port` |
| Dynamic | `-D` | `local_port` (SOCKS proxy) |
