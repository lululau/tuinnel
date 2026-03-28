# SSH Tunnel TUI Manager — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a terminal-based SSH tunnel manager with tabbed TUI using Go, bubbletea v2, and bubbles v2.

**Architecture:** Single-process bubbletea app. Core layer (tunnel types, TOML config, SSH ControlMaster client, tunnel manager) provides the business logic. UI layer (styles, tab bar, 4 tab models) renders the TUI. App model wires them together.

**Tech Stack:** Go 1.23+, charmbracelet/bubbletea v2, charmbracelet/bubbles v2, charmbracelet/lipgloss v2, BurntSushi/toml

---

## Task 1: Project Scaffolding

**Files:**
- Create: `go.mod`
- Create: `internal/tunnel/tunnel.go`
- Create: `internal/tunnel/config.go`
- Create: `internal/tunnel/manager.go`
- Create: `internal/ssh/client.go`
- Create: `internal/ui/styles.go`
- Create: `internal/ui/tabs/tablist.go`
- Create: `internal/ui/tabs/tunnel_list.go`
- Create: `internal/ui/tabs/logs.go`
- Create: `internal/ui/tabs/settings.go`
- Create: `internal/ui/tabs/editor.go`
- Create: `internal/app/model.go`
- Create: `main.go`

**Step 1: Initialize Go module and create directory structure**

Run:
```bash
cd /Users/liuxiang/cascode/github.com/lululau/tuinnel
mkdir -p internal/tunnel internal/ssh internal/ui/tabs internal/app
go mod init github.com/lululau/tuinnel
```

**Step 2: Install dependencies**

Run:
```bash
go get charm.land/bubbletea/v2
go get charm.land/bubbles/v2
go get charm.land/lipgloss/v2
go get github.com/BurntSushi/toml@latest
```

**Step 3: Create placeholder files so the project compiles**

Create each file listed above with a minimal `package` declaration. For `main.go`:

```go
package main

func main() {}
```

For all other files, just the package declaration:

```go
package tunnel
```
```go
package ssh
```
```go
package ui
```
```go
package tabs
```
```go
package app
```

**Step 4: Verify it compiles**

Run: `go build ./...`
Expected: no errors

**Step 5: Commit**

```bash
git add -A
git commit -m "chore: scaffold project with Go module and directory structure"
```

---

## Task 2: Tunnel Types and Config Parsing

**Files:**
- Modify: `internal/tunnel/tunnel.go`
- Modify: `internal/tunnel/config.go`
- Create: `internal/tunnel/tunnel_test.go`
- Create: `internal/tunnel/config_test.go`
- Create: `testdata/config_valid.toml`
- Create: `testdata/config_empty.toml`

**Step 1: Write failing tests for Tunnel struct**

Create `internal/tunnel/tunnel_test.go`:

```go
package tunnel

import "testing"

func TestTunnelTypeSSHFlag(t *testing.T) {
	tests := []struct {
		ttype TunnelType
		want  string
	}{
		{TunnelLocal, "-L"},
		{TunnelRemote, "-R"},
		{TunnelDynamic, "-D"},
		{"unknown", "-L"},
	}
	for _, tt := range tests {
		if got := tt.ttype.SSHFlag(); got != tt.want {
			t.Errorf("TunnelType(%q).SSHFlag() = %q, want %q", tt.ttype, got, tt.want)
		}
	}
}

func TestTunnelForwardSpec(t *testing.T) {
	tests := []struct {
		name  string
		tunnel Tunnel
		want  string
	}{
		{
			"local forward",
			Tunnel{Type: TunnelLocal, LocalPort: 3307, RemoteHost: "localhost", RemotePort: 3306},
			"3307:localhost:3306",
		},
		{
			"remote forward",
			Tunnel{Type: TunnelRemote, LocalPort: 8080, RemoteHost: "localhost", RemotePort: 80},
			"80:localhost:8080",
		},
		{
			"dynamic forward",
			Tunnel{Type: TunnelDynamic, LocalPort: 1080},
			"1080",
		},
	}
	for _, tt := range tests {
		if got := tt.tunnel.ForwardSpec(); got != tt.want {
			t.Errorf("%s: ForwardSpec() = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestTunnelValidate(t *testing.T) {
	tests := []struct {
		name    string
		tunnel  Tunnel
		wantErr bool
	}{
		{"valid local", Tunnel{Name: "t", Type: TunnelLocal, LocalPort: 1, RemotePort: 1, Login: "u@h"}, false},
		{"valid dynamic", Tunnel{Name: "t", Type: TunnelDynamic, LocalPort: 1, Login: "u@h"}, false},
		{"missing name", Tunnel{Type: TunnelLocal, LocalPort: 1, Login: "u@h"}, true},
		{"missing login", Tunnel{Name: "t", Type: TunnelLocal, LocalPort: 1}, true},
		{"missing port local", Tunnel{Name: "t", Type: TunnelLocal, Login: "u@h"}, true},
		{"missing port dynamic", Tunnel{Name: "t", Type: TunnelDynamic, Login: "u@h"}, true},
	}
	for _, tt := range tests {
		err := tt.tunnel.Validate()
		if (err != nil) != tt.wantErr {
			t.Errorf("%s: Validate() error = %v, wantErr %v", tt.name, err, tt.wantErr)
		}
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/tunnel/ -run TestTunnel -v`
Expected: compilation errors (types not defined yet)

**Step 3: Implement Tunnel types**

Write `internal/tunnel/tunnel.go`:

```go
package tunnel

import (
	"fmt"
	"strings"
)

type TunnelType string

const (
	TunnelLocal   TunnelType = "local"
	TunnelRemote  TunnelType = "remote"
	TunnelDynamic TunnelType = "dynamic"
)

func (t TunnelType) SSHFlag() string {
	switch t {
	case TunnelRemote:
		return "-R"
	case TunnelDynamic:
		return "-D"
	default:
		return "-L"
	}
}

func (t TunnelType) Display() string {
	switch t {
	case TunnelRemote:
		return "Remote"
	case TunnelDynamic:
		return "Dynamic"
	default:
		return "Local"
	}
}

type Tunnel struct {
	Name       string     `toml:"name"`
	Type       TunnelType `toml:"type"`
	LocalPort  int        `toml:"local_port"`
	RemoteHost string     `toml:"remote_host"`
	RemotePort int        `toml:"remote_port"`
	Login      string     `toml:"login"`
	Group      string     `toml:"group"`

	Running bool `toml:"-"`
	Error   bool `toml:"-"`
}

func (t Tunnel) ForwardSpec() string {
	switch t.Type {
	case TunnelDynamic:
		return fmt.Sprintf("%d", t.LocalPort)
	case TunnelRemote:
		return fmt.Sprintf("%d:%s:%d", t.RemotePort, t.remoteHost(), t.LocalPort)
	default: // Local
		return fmt.Sprintf("%d:%s:%d", t.LocalPort, t.remoteHost(), t.RemotePort)
	}
}

func (t Tunnel) remoteHost() string {
	if t.RemoteHost == "" {
		return "localhost"
	}
	// IPv6 addresses need brackets
	if strings.Contains(t.RemoteHost, ":") {
		return "[" + t.RemoteHost + "]"
	}
	return t.RemoteHost
}

func (t Tunnel) Validate() error {
	if t.Name == "" {
		return fmt.Errorf("tunnel name is required")
	}
	if t.Login == "" {
		return fmt.Errorf("tunnel %q: login is required", t.Name)
	}
	if t.Type == TunnelDynamic {
		if t.LocalPort == 0 {
			return fmt.Errorf("tunnel %q: local_port is required for dynamic tunnels", t.Name)
		}
		return nil
	}
	if t.LocalPort == 0 {
		return fmt.Errorf("tunnel %q: local_port is required", t.Name)
	}
	if t.RemotePort == 0 {
		return fmt.Errorf("tunnel %q: remote_port is required", t.Name)
	}
	return nil
}
```

**Step 4: Run Tunnel tests to verify they pass**

Run: `go test ./internal/tunnel/ -run TestTunnel -v`
Expected: all PASS

**Step 5: Write failing tests for config parsing**

Create `testdata/config_valid.toml`:

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
name = "socks-proxy"
type = "dynamic"
local_port = 1080
login = "user@jump-host"
```

Create `testdata/config_empty.toml`:

```toml
[settings]
```

Create `internal/tunnel/config_test.go`:

```go
package tunnel

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "config_valid.toml")
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.Settings.SSHBin != "ssh" {
		t.Errorf("SSHBin = %q, want %q", cfg.Settings.SSHBin, "ssh")
	}
	if len(cfg.Tunnels) != 2 {
		t.Fatalf("len(Tunnels) = %d, want 2", len(cfg.Tunnels))
	}
	if cfg.Tunnels[0].Name != "dev-db" {
		t.Errorf("Tunnels[0].Name = %q", cfg.Tunnels[0].Name)
	}
	if cfg.Tunnels[1].Type != TunnelDynamic {
		t.Errorf("Tunnels[1].Type = %q", cfg.Tunnels[1].Type)
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "config_empty.toml")
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.Settings.SSHBin != "ssh" {
		t.Errorf("SSHBin default = %q, want %q", cfg.Settings.SSHBin, "ssh")
	}
	if cfg.Settings.ControlDir != "/tmp/tuinnel" {
		t.Errorf("ControlDir default = %q", cfg.Settings.ControlDir)
	}
	if len(cfg.Tunnels) != 0 {
		t.Errorf("len(Tunnels) = %d, want 0", len(cfg.Tunnels))
	}
}

func TestSaveConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	cfg := &Config{
		Settings: Settings{SSHBin: "ssh", ControlDir: "/tmp/test", KillOnExit: true},
		Tunnels: []Tunnel{
			{Name: "test", Type: TunnelLocal, LocalPort: 1234, RemotePort: 5678, Login: "u@h"},
		},
	}

	if err := SaveConfig(cfg, path); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	loaded, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig() after save error = %v", err)
	}
	if loaded.Tunnels[0].Name != "test" {
		t.Errorf("roundtrip: Name = %q", loaded.Tunnels[0].Name)
	}
	if !loaded.Settings.KillOnExit {
		t.Error("roundtrip: KillOnExit = false, want true")
	}
}

func TestConfigPath(t *testing.T) {
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".config", "tuinnel", "config.toml")
	if got := DefaultConfigPath(); got != want {
		t.Errorf("DefaultConfigPath() = %q, want %q", got, want)
	}
}
```

**Step 6: Run config tests to verify they fail**

Run: `go test ./internal/tunnel/ -run TestConfig -v`
Expected: compilation errors

**Step 7: Implement config parsing**

Write `internal/tunnel/config.go`:

```go
package tunnel

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Settings struct {
	SSHBin      string `toml:"ssh_bin"`
	ControlDir  string `toml:"control_dir"`
	KillOnExit  bool   `toml:"kill_on_exit"`
}

type Config struct {
	Settings Settings `toml:"settings"`
	Tunnels  []Tunnel `toml:"tunnels"`
}

func DefaultConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "tuinnel", "config.toml")
}

func DefaultSettings() Settings {
	return Settings{
		SSHBin:     "ssh",
		ControlDir: "/tmp/tuinnel",
	}
}

func LoadConfig(path string) (*Config, error) {
	var cfg Config
	cfg.Settings = DefaultSettings()

	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}

	for i, t := range cfg.Tunnels {
		if t.Type == "" {
			cfg.Tunnels[i].Type = TunnelLocal
		}
	}

	return &cfg, nil
}

func SaveConfig(cfg *Config, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create config file: %w", err)
	}
	defer f.Close()

	enc := toml.NewEncoder(f)
	if err := enc.Encode(cfg); err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	return nil
}
```

**Step 8: Run all tunnel tests**

Run: `go test ./internal/tunnel/ -v`
Expected: all PASS

**Step 9: Commit**

```bash
git add internal/tunnel/ testdata/
git commit -m "feat: implement tunnel types and TOML config parsing"
```

---

## Task 3: SSH Client Wrapper

**Files:**
- Modify: `internal/ssh/client.go`
- Create: `internal/ssh/client_test.go`

**Step 1: Write failing tests for SSH client**

Create `internal/ssh/client_test.go`:

```go
package ssh

import "testing"

func TestBuildStartArgs(t *testing.T) {
	c := Client{Bin: "ssh"}
	args := c.BuildStartArgs("/tmp/ctrl/test", "-L", "3307:localhost:3306", "deploy@host")
	expected := []string{"-M", "-f", "-N", "-T", "-S", "/tmp/ctrl/test", "-L", "3307:localhost:3306", "deploy@host"}
	if len(args) != len(expected) {
		t.Fatalf("len(args) = %d, want %d", len(args), len(expected))
	}
	for i, a := range args {
		if a != expected[i] {
			t.Errorf("args[%d] = %q, want %q", i, a, expected[i])
		}
	}
}

func TestBuildStopArgs(t *testing.T) {
	c := Client{Bin: "ssh"}
	args := c.BuildStopArgs("/tmp/ctrl/test", "deploy@host")
	expected := []string{"-S", "/tmp/ctrl/test", "-O", "exit", "deploy@host"}
	if len(args) != len(expected) {
		t.Fatalf("len(args) = %d, want %d", len(args), len(expected))
	}
	for i, a := range args {
		if a != expected[i] {
			t.Errorf("args[%d] = %q, want %q", i, a, expected[i])
		}
	}
}

func TestBuildCheckArgs(t *testing.T) {
	c := Client{Bin: "ssh"}
	args := c.BuildCheckArgs("/tmp/ctrl/test", "deploy@host")
	expected := []string{"-S", "/tmp/ctrl/test", "-O", "check", "deploy@host"}
	for i, a := range args {
		if a != expected[i] {
			t.Errorf("args[%d] = %q, want %q", i, a, expected[i])
		}
	}
}

func TestControlSocketPath(t *testing.T) {
	c := Client{ControlDir: "/tmp/tuinnel"}
	if got := c.SocketPath("dev-db"); got != "/tmp/tuinnel/dev-db" {
		t.Errorf("SocketPath() = %q, want %q", got, "/tmp/tuinnel/dev-db")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/ssh/ -v`
Expected: compilation errors

**Step 3: Implement SSH client**

Write `internal/ssh/client.go`:

```go
package ssh

import (
	"fmt"
	"os/exec"
)

type Client struct {
	Bin        string
	ControlDir string
}

func (c Client) SocketPath(name string) string {
	return fmt.Sprintf("%s/%s", c.ControlDir, name)
}

func (c Client) BuildStartArgs(socket, flag, forward, login string) []string {
	return []string{"-M", "-f", "-N", "-T", "-S", socket, flag, forward, login}
}

func (c Client) BuildStopArgs(socket, login string) []string {
	return []string{"-S", socket, "-O", "exit", login}
}

func (c Client) BuildCheckArgs(socket, login string) []string {
	return []string{"-S", socket, "-O", "check", login}
}

func (c Client) Start(socket, flag, forward, login string) error {
	args := c.BuildStartArgs(socket, flag, forward, login)
	cmd := exec.Command(c.Bin, args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ssh start: %s (exit: %v)", string(output), err)
	}
	return nil
}

func (c Client) Stop(socket, login string) error {
	args := c.BuildStopArgs(socket, login)
	cmd := exec.Command(c.Bin, args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ssh stop: %w", err)
	}
	return nil
}

func (c Client) Check(socket, login string) bool {
	args := c.BuildCheckArgs(socket, login)
	cmd := exec.Command(c.Bin, args...)
	return cmd.Run() == nil
}
```

**Step 4: Run SSH tests**

Run: `go test ./internal/ssh/ -v`
Expected: all PASS

**Step 5: Commit**

```bash
git add internal/ssh/
git commit -m "feat: implement SSH ControlMaster client wrapper"
```

---

## Task 4: Tunnel Manager

**Files:**
- Modify: `internal/tunnel/manager.go`
- Create: `internal/tunnel/manager_test.go`

**Step 1: Write failing tests for tunnel manager**

Create `internal/tunnel/manager_test.go`:

```go
package tunnel

import "testing"

func TestManagerSocketPath(t *testing.T) {
	m := NewManager(&Config{
		Settings: Settings{SSHBin: "ssh", ControlDir: "/tmp/test"},
		Tunnels: []Tunnel{{Name: "t1", Login: "u@h", Type: TunnelLocal, LocalPort: 1, RemotePort: 1}},
	})
	if got := m.client.SocketPath("t1"); got != "/tmp/test/t1" {
		t.Errorf("socket path = %q", got)
	}
}

func TestManagerGroups(t *testing.T) {
	m := NewManager(&Config{
		Settings: Settings{SSHBin: "ssh", ControlDir: "/tmp/test"},
		Tunnels: []Tunnel{
			{Name: "t1", Group: "dev", Login: "u@h", Type: TunnelLocal, LocalPort: 1, RemotePort: 1},
			{Name: "t2", Group: "prod", Login: "u@h", Type: TunnelLocal, LocalPort: 2, RemotePort: 2},
			{Name: "t3", Group: "dev", Login: "u@h", Type: TunnelLocal, LocalPort: 3, RemotePort: 3},
		},
	})
	groups := m.Groups()
	if len(groups) != 2 {
		t.Fatalf("Groups() = %v, want 2 groups", groups)
	}

	devTunnels := m.TunnelsByGroup("dev")
	if len(devTunnels) != 2 {
		t.Errorf("TunnelsByGroup(dev) = %d, want 2", len(devTunnels))
	}
}

func TestManagerRunningCount(t *testing.T) {
	m := NewManager(&Config{
		Settings: Settings{SSHBin: "ssh", ControlDir: "/tmp/test"},
		Tunnels: []Tunnel{
			{Name: "t1", Login: "u@h", Type: TunnelLocal, LocalPort: 1, RemotePort: 1},
			{Name: "t2", Login: "u@h", Type: TunnelLocal, LocalPort: 2, RemotePort: 2},
		},
	})
	if m.RunningCount() != 0 {
		t.Errorf("RunningCount() = %d, want 0", m.RunningCount())
	}
	m.tunnels[0].Running = true
	if m.RunningCount() != 1 {
		t.Errorf("RunningCount() = %d, want 1", m.RunningCount())
	}
}

func TestRingBuffer(t *testing.T) {
	rb := NewRingBuffer(3)
	rb.Add("line1")
	rb.Add("line2")
	rb.Add("line3")
	rb.Add("line4") // evicts line1

	lines := rb.Lines()
	if len(lines) != 3 {
		t.Fatalf("len(lines) = %d, want 3", len(lines))
	}
	if lines[0] != "line2" {
		t.Errorf("lines[0] = %q, want %q", lines[0], "line2")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/tunnel/ -run TestManager -v`
Expected: compilation errors

**Step 3: Implement tunnel manager and ring buffer**

Write `internal/tunnel/manager.go`:

```go
package tunnel

import (
	"fmt"
	"os"
	"sync"

	"github.com/lululau/tuinnel/internal/ssh"
)

type RingBuffer struct {
	mu    sync.Mutex
	lines []string
	cap   int
}

func NewRingBuffer(cap int) *RingBuffer {
	return &RingBuffer{cap: cap}
}

func (rb *RingBuffer) Add(line string) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	if len(rb.lines) >= rb.cap {
		rb.lines = rb.lines[1:]
	}
	rb.lines = append(rb.lines, line)
}

func (rb *RingBuffer) Lines() []string {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	out := make([]string, len(rb.lines))
	copy(out, rb.lines)
	return out
}

func (rb *RingBuffer) Clear() {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.lines = nil
}

type Manager struct {
	config  *Config
	client  ssh.Client
	tunnels []Tunnel
	logs    map[string]*RingBuffer
}

func NewManager(cfg *Config) *Manager {
	tunnels := make([]Tunnel, len(cfg.Tunnels))
	copy(tunnels, cfg.Tunnels)
	return &Manager{
		config:  cfg,
		client:  ssh.Client{Bin: cfg.Settings.SSHBin, ControlDir: cfg.Settings.ControlDir},
		tunnels: tunnels,
		logs:    make(map[string]*RingBuffer),
	}
}

func (m *Manager) Tunnels() []Tunnel {
	return m.tunnels
}

func (m *Manager) Log(name string) *RingBuffer {
	if rb, ok := m.logs[name]; ok {
		return rb
	}
	rb := NewRingBuffer(1000)
	m.logs[name] = rb
	return rb
}

func (m *Manager) Groups() []string {
	seen := map[string]bool{}
	for _, t := range m.tunnels {
		if t.Group != "" {
			seen[t.Group] = true
		}
	}
	groups := make([]string, 0, len(seen))
	for g := range seen {
		groups = append(groups, g)
	}
	return groups
}

func (m *Manager) TunnelsByGroup(group string) []Tunnel {
	var out []Tunnel
	for _, t := range m.tunnels {
		if t.Group == group {
			out = append(out, t)
		}
	}
	return out
}

func (m *Manager) RunningCount() int {
	count := 0
	for _, t := range m.tunnels {
		if t.Running {
			count++
		}
	}
	return count
}

func (m *Manager) HasRunning() bool {
	return m.RunningCount() > 0
}

func (m *Manager) Start(name string) error {
	idx := m.findTunnel(name)
	if idx < 0 {
		return fmt.Errorf("tunnel %q not found", name)
	}
	t := &m.tunnels[idx]

	socket := m.client.SocketPath(t.Name)
	if m.client.Check(socket, t.Login) {
		t.Running = true
		t.Error = false
		return nil
	}

	if err := os.MkdirAll(m.config.Settings.ControlDir, 0755); err != nil {
		return fmt.Errorf("create control dir: %w", err)
	}

	forward := t.ForwardSpec()
	if err := m.client.Start(socket, t.Type.SSHFlag(), forward, t.Login); err != nil {
		t.Error = true
		t.Running = false
		m.Log(name).Add(fmt.Sprintf("[ERROR] %s", err))
		return err
	}

	t.Running = true
	t.Error = false
	m.Log(name).Add(fmt.Sprintf("[STARTED] %s %s %s", t.Type.SSHFlag(), forward, t.Login))
	return nil
}

func (m *Manager) Stop(name string) error {
	idx := m.findTunnel(name)
	if idx < 0 {
		return fmt.Errorf("tunnel %q not found", name)
	}
	t := &m.tunnels[idx]

	socket := m.client.SocketPath(t.Name)
	if err := m.client.Stop(socket, t.Login); err != nil {
		t.Error = true
		m.Log(name).Add(fmt.Sprintf("[ERROR] stop: %s", err))
		return err
	}

	t.Running = false
	t.Error = false
	m.Log(name).Add("[STOPPED]")
	return nil
}

func (m *Manager) Restart(name string) error {
	if err := m.Stop(name); err != nil {
		// ignore stop errors (tunnel may not be running)
	}
	return m.Start(name)
}

func (m *Manager) StartGroup(group string) error {
	for _, t := range m.TunnelsByGroup(group) {
		if err := m.Start(t.Name); err != nil {
			return fmt.Errorf("group %q: %w", group, err)
		}
	}
	return nil
}

func (m *Manager) StopGroup(group string) error {
	for _, t := range m.TunnelsByGroup(group) {
		if err := m.Stop(t.Name); err != nil {
			return fmt.Errorf("group %q: %w", group, err)
		}
	}
	return nil
}

func (m *Manager) StopAll() error {
	for _, t := range m.tunnels {
		if t.Running {
			_ = m.Stop(t.Name)
		}
	}
	return nil
}

func (m *Manager) Refresh() {
	for i, t := range m.tunnels {
		socket := m.client.SocketPath(t.Name)
		m.tunnels[i].Running = m.client.Check(socket, t.Login)
		m.tunnels[i].Error = false
	}
}

func (m *Manager) AddTunnel(t Tunnel) {
	t.Running = false
	t.Error = false
	m.tunnels = append(m.tunnels, t)
	m.config.Tunnels = m.tunnels
}

func (m *Manager) UpdateTunnel(index int, t Tunnel) {
	t.Running = m.tunnels[index].Running
	t.Error = m.tunnels[index].Error
	m.tunnels[index] = t
	m.config.Tunnels = m.tunnels
}

func (m *Manager) RemoveTunnel(index int) {
	_ = m.Stop(m.tunnels[index].Name)
	m.tunnels = append(m.tunnels[:index], m.tunnels[index+1:]...)
	m.config.Tunnels = m.tunnels
}

func (m *Manager) findTunnel(name string) int {
	for i, t := range m.tunnels {
		if t.Name == name {
			return i
		}
	}
	return -1
}
```

**Step 4: Run all tunnel tests**

Run: `go test ./internal/tunnel/ -v`
Expected: all PASS

**Step 5: Commit**

```bash
git add internal/tunnel/manager.go internal/tunnel/manager_test.go
git commit -m "feat: implement tunnel manager with lifecycle, groups, and ring buffer"
```

---

## Task 5: UI Styles

**Files:**
- Modify: `internal/ui/styles.go`

**Step 1: Implement global lipgloss styles**

Write `internal/ui/styles.go`:

```go
package ui

import "charm.land/lipgloss/v2"

var (
	ColorGreen  = lipgloss.Color("#04B575")
	ColorRed    = lipgloss.Color("#FF5F57")
ColorYellow  = lipgloss.Color("#F1C40F")
ColorGray    = lipgloss.Color("#6C7086")
ColorCyan    = lipgloss.Color("#00D7FF")
ColorDim     = lipgloss.Color("#45475A")
ColorText    = lipgloss.Color("#CDD6F4")
ColorSubtext = lipgloss.Color("#A6ADC8")
ColorBg      = lipgloss.Color("#1E1E2E")
ColorSurface = lipgloss.Color("#313244")

	StyleTitle = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorCyan)

	StyleRunning = lipgloss.NewStyle().
		Foreground(ColorGreen)

	StyleStopped = lipgloss.NewStyle().
		Foreground(ColorGray)

	StyleError = lipgloss.NewStyle().
		Foreground(ColorRed)

	StyleTabActive = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorCyan).
		Padding(0, 1).
		Border(lipgloss.BottomBorder(), true, ColorCyan)

	StyleTabInactive = lipgloss.NewStyle().
		Foreground(ColorSubtext).
		Padding(0, 1)

	StyleStatusBar = lipgloss.NewStyle().
		Foreground(ColorSubtext).
		Padding(0, 1)

	StyleHelp = lipgloss.NewStyle().
		Foreground(ColorGray)

	StyleFocused = lipgloss.NewStyle().
		Foreground(ColorText)

	StyleInput = lipgloss.NewStyle().
		Foreground(ColorText).
		Background(ColorSurface)
)
```

**Step 2: Verify it compiles**

Run: `go build ./internal/ui/`
Expected: no errors

**Step 3: Commit**

```bash
git add internal/ui/styles.go
git commit -m "feat: add global lipgloss styles with Catppuccin Mocha palette"
```

---

## Task 6: Tab Bar Component

**Files:**
- Modify: `internal/ui/tabs/tablist.go`

**Step 1: Implement tab bar**

Write `internal/ui/tabs/tablist.go`:

```go
package tabs

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/lululau/tuinnel/internal/ui"
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
```

Note: this file imports `lipgloss` which needs `"charm.land/lipgloss/v2"`. Add the import.

**Step 2: Verify it compiles**

Run: `go build ./internal/ui/tabs/`
Expected: no errors

**Step 3: Commit**

```bash
git add internal/ui/tabs/tablist.go
git commit -m "feat: implement tab bar component"
```

---

## Task 7: Tunnel List Tab

**Files:**
- Modify: `internal/ui/tabs/tunnel_list.go`

**Step 1: Implement tunnel list tab**

Write `internal/ui/tabs/tunnel_list.go`:

```go
package tabs

import (
	"fmt"
	"strconv"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"

	"github.com/lululau/tuinnel/internal/tunnel"
	"github.com/lululau/tuinnel/internal/ui"
)

type tunnelListModel struct {
	table  table.Model
	width  int
	height int
}

func newTunnelListModel() tunnelListModel {
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

	return tunnelListModel{table: t}
}

func (m *tunnelListModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.table = m.table.WithHeight(h - 4) // leave room for header/footer
}

func (m *tunnelListModel) UpdateTunnels(tunnels []tunnel.Tunnel) {
	var rows []table.Row
	for _, t := range tunnels {
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

func (m *tunnelListModel) SelectedTunnelName() string {
	row := m.table.SelectedRow()
	if row == nil {
		return ""
	}
	return row[1] // Name column
}

func (m *tunnelListModel) Update(msg tea.Msg) (tunnelListModel, tea.Cmd) {
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return *m, cmd
}

func (m tunnelListModel) View() string {
	return ui.StyleFocused.Render(m.table.View())
}

// statusText returns a status bar string for the tunnel list.
func (m *tunnelListModel) statusText(tunnels []tunnel.Tunnel, runningCount int) string {
	total := len(tunnels)
	return fmt.Sprintf("%d/%d tunnels running", runningCount, total)
}
```

**Step 2: Verify it compiles**

Run: `go build ./internal/ui/tabs/`
Expected: no errors

**Step 3: Commit**

```bash
git add internal/ui/tabs/tunnel_list.go
git commit -m "feat: implement tunnel list tab with table view"
```

---

## Task 8: Log Panel Tab

**Files:**
- Modify: `internal/ui/tabs/logs.go`

**Step 1: Implement log panel tab**

Write `internal/ui/tabs/logs.go`:

```go
package tabs

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"

	"github.com/lululau/tuinnel/internal/tunnel"
	"github.com/lululau/tuinnel/internal/ui"
)

type logModel struct {
	tunnels  []tunnel.Tunnel
	cursor   int
	viewport viewport.Model
	width    int
	height   int
}

func newLogModel() logModel {
	vp := viewport.New(40, 10)
	vp.SetContent("Select a tunnel to view logs.")
	return logModel{viewport: vp}
}

func (m *logModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.viewport.Width = w/2 - 2
	m.viewport.Height = h - 4
}

func (m *logModel) UpdateTunnels(tunnels []tunnel.Tunnel) {
	m.tunnels = tunnels
	if m.cursor >= len(tunnels) {
		m.cursor = max(0, len(tunnels)-1)
	}
}

func (m *logModel) UpdateLogs(name string, lines []string) {
	content := strings.Join(lines, "\n")
	if content == "" {
		content = "No logs yet."
	}
	m.viewport.SetContent(content)
	m.viewport.GotoBottom()
}

func (m *logModel) SelectedTunnelName() string {
	if len(m.tunnels) == 0 {
		return ""
	}
	return m.tunnels[m.cursor].Name
}

func (m *logModel) Update(msg tea.Msg) (logModel, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			return *m, nil
		case "down", "j":
			if m.cursor < len(m.tunnels)-1 {
				m.cursor++
			}
			return *m, nil
		}
	}
	m.viewport, cmd = m.viewport.Update(msg)
	return *m, cmd
}

func (m logModel) View() string {
	// Left panel: tunnel list
	var tunnelList strings.Builder
	for i, t := range m.tunnels {
		cursor := " "
		if i == m.cursor {
			cursor = ">"
		}
		state := "○"
		if t.Running {
			state = "●"
		}
		tunnelList.WriteString(fmt.Sprintf("%s %s %s\n", cursor, state, t.Name))
	}

	left := lipgloss.NewStyle().
		Width(m.width/2 - 1).
		Height(m.height - 4).
		Render(tunnelList.String())

	right := ui.StyleFocused.Render(m.viewport.View())

	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}
```

**Step 2: Verify it compiles**

Run: `go build ./internal/ui/tabs/`
Expected: no errors

**Step 3: Commit**

```bash
git add internal/ui/tabs/logs.go
git commit -m "feat: implement log panel tab with split view"
```

---

## Task 9: Settings Tab

**Files:**
- Modify: `internal/ui/tabs/settings.go`

**Step 1: Implement settings tab**

Write `internal/ui/tabs/settings.go`:

```go
package tabs

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"

	"github.com/lululau/tuinnel/internal/tunnel"
	"github.com/lululau/tuinnel/internal/ui"
)

type settingsModel struct {
	inputs  []textinput.Model
	cursor  int
	focus   int // which input is focused: -1 = none
	width   int
	height  int
	saved   bool
}

func newSettingsModel(settings tunnel.Settings) settingsModel {
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

	return settingsModel{inputs: inputs, focus: -1}
}

type settingsKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Edit   key.Binding
	Save   key.Binding
	Quit   key.Binding
}

var settingsKeys = settingsKeyMap{
	Up:   key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "move up")),
	Down: key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "move down")),
	Edit: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "edit field")),
	Save: key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("ctrl+s", "save")),
	Quit: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
}

func (m *settingsModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	for i := range m.inputs {
		m.inputs[i].Width = w - 20
	}
}

func (m *settingsModel) Settings() tunnel.Settings {
	killOnExit := false
	if m.inputs[2].Value() == "true" {
		killOnExit = true
	}
	return tunnel.Settings{
		SSHBin:     m.inputs[0].Value(),
		ControlDir: m.inputs[1].Value(),
		KillOnExit: killOnExit,
	}
}

func (m *settingsModel) Update(msg tea.Msg) (settingsModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, settingsKeys.Up):
			if m.focus >= 0 {
				m.inputs[m.focus].Blur()
				m.focus--
				if m.focus >= 0 {
					m.inputs[m.focus].Focus()
				}
			} else if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, settingsKeys.Down):
			if m.focus >= 0 {
				m.inputs[m.focus].Blur()
				m.focus++
				if m.focus < len(m.inputs) {
					m.inputs[m.focus].Focus()
				} else {
					m.focus = -1
				}
			} else if m.cursor < len(m.inputs)-1 {
				m.cursor++
			}
		case key.Matches(msg, settingsKeys.Edit):
			m.focus = m.cursor
			m.inputs[m.focus].Focus()
			return *m, textinput.Blink
		case key.Matches(msg, settingsKeys.Save):
			m.saved = true
			return *m, func() tea.Msg { return settingsSavedMsg{} }
		case key.Matches(msg, settingsKeys.Quit):
			if m.focus >= 0 {
				m.inputs[m.focus].Blur()
				m.focus = -1
			}
		}
	}

	// Update focused input
	if m.focus >= 0 && m.focus < len(m.inputs) {
		m.inputs[m.focus], cmd = m.inputs[m.focus].Update(msg)
	}

	return *m, cmd
}

func (m settingsModel) View() string {
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

	s.WriteString("\n" + ui.StyleHelp.Render("enter: edit • ctrl+s: save • esc: back"))

	return s.String()
}

type settingsSavedMsg struct{}
```

**Step 2: Verify it compiles**

Run: `go build ./internal/ui/tabs/`
Expected: no errors (may need `strings` import)

**Step 3: Commit**

```bash
git add internal/ui/tabs/settings.go
git commit -m "feat: implement settings tab with text input forms"
```

---

## Task 10: Tunnel Editor Tab

**Files:**
- Modify: `internal/ui/tabs/editor.go`

**Step 1: Implement tunnel editor tab**

Write `internal/ui/tabs/editor.go`:

```go
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

type editorModel struct {
	mode    editorMode
	editIdx int // index being edited (-1 for new)
	inputs  []textinput.Model
	cursor  int
	focus   int
	width   int
	height  int
	message string
}

func newEditorModel() editorModel {
	inputs := make([]textinput.Model, 7)
	fields := []struct {
		label string
		width int
	}{
		{"Name", 20}, {"Type (local/remote/dynamic)", 25},
		{"Local Port", 7}, {"Remote Host", 20},
		{"Remote Port", 7}, {"Login (user@host)", 30},
		{"Group", 15},
	}
	for i, f := range fields {
		inputs[i] = textinput.New()
		inputs[i].Placeholder = f.label
		inputs[i].CharLimit = f.width
	}
	return editorModel{inputs: inputs, mode: editorIdle, editIdx: -1, focus: -1}
}

type editorKeyMap struct {
	Add    key.Binding
	Save   key.Binding
	Delete key.Binding
	Cancel key.Binding
}

var editorKeys = editorKeyMap{
	Add:    key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add new")),
	Save:   key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("ctrl+s", "save")),
	Delete: key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("ctrl+d", "delete")),
	Cancel: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
}

func (m *editorModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	for i := range m.inputs {
		m.inputs[i].Width = w - 25
	}
}

func (m *editorModel) StartAdd() {
	m.mode = editorAdd
	m.editIdx = -1
	m.message = ""
	for i := range m.inputs {
		m.inputs[i].SetValue("")
	}
	m.focus = 0
	m.inputs[0].Focus()
}

func (m *editorModel) StartEdit(idx int, t tunnel.Tunnel) {
	m.mode = editorEdit
	m.editIdx = idx
	m.message = ""
	m.inputs[0].SetValue(t.Name)
	m.inputs[1].SetValue(string(t.Type))
	m.inputs[2].SetValue(strconv.Itoa(t.LocalPort))
	m.inputs[3].SetValue(t.RemoteHost)
	m.inputs[4].SetValue(strconv.Itoa(t.RemotePort))
	m.inputs[5].SetValue(t.Login)
	m.inputs[6].SetValue(t.Group)
	m.focus = 0
	m.inputs[0].Focus()
}

func (m *editorModel) Cancel() {
	m.mode = editorIdle
	m.editIdx = -1
	m.message = ""
	m.blurAll()
}

func (m *editorModel) blurAll() {
	for i := range m.inputs {
		m.inputs[i].Blur()
	}
	m.focus = -1
}

func (m *editorModel) Tunnel() (tunnel.Tunnel, error) {
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

func (m *editorModel) IsEditing() bool {
	return m.mode != editorIdle
}

func (m *editorModel) EditIndex() int {
	return m.editIdx
}

func (m *editorModel) Update(msg tea.Msg) (editorModel, tea.Cmd) {
	var cmd tea.Cmd

	if m.mode == editorIdle {
		switch msg := msg.(type) {
		case tea.KeyPressMsg:
			if key.Matches(msg, editorKeys.Add) {
				m.StartAdd()
				return *m, textinput.Blink
			}
		}
		return *m, nil
	}

	// Editing mode
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, editorKeys.Cancel):
			m.Cancel()
			return *m, nil
		case key.Matches(msg, editorKeys.Save):
			return *m, func() tea.Msg { return editorSaveMsg{} }
		case msg.String() == "enter":
			if m.focus >= 0 && m.focus < len(m.inputs)-1 {
				m.inputs[m.focus].Blur()
				m.focus++
				m.inputs[m.focus].Focus()
				return *m, textinput.Blink
			} else if m.focus == len(m.inputs)-1 {
				// Last field, save
				return *m, func() tea.Msg { return editorSaveMsg{} }
			}
		case msg.String() == "up":
			if m.focus > 0 {
				m.inputs[m.focus].Blur()
				m.focus--
				m.inputs[m.focus].Focus()
				return *m, textinput.Blink
			}
		}
	}

	if m.focus >= 0 && m.focus < len(m.inputs) {
		m.inputs[m.focus], cmd = m.inputs[m.focus].Update(msg)
	}

	return *m, cmd
}

func (m editorModel) View() string {
	labels := []string{"Name", "Type", "Local Port", "Remote Host", "Remote Port", "Login", "Group"}

	var title string
	switch m.mode {
	case editorIdle:
		title = ui.StyleTitle.Render("Tunnel Editor")
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

	if m.message != "" {
		s.WriteString("\n" + ui.StyleError.Render(m.message))
	}

	s.WriteString("\n" + ui.StyleHelp.Render("a: add new • ctrl+s: save • ctrl+d: delete • esc: cancel"))

	return s.String()
}

type editorSaveMsg struct{}
```

**Step 2: Verify it compiles**

Run: `go build ./internal/ui/tabs/`
Expected: no errors

**Step 3: Commit**

```bash
git add internal/ui/tabs/editor.go
git commit -m "feat: implement tunnel editor tab with add/edit/delete forms"
```

---

## Task 11: App Model — Wire Everything Together

**Files:**
- Modify: `internal/app/model.go`

**Step 1: Implement the app model**

Write `internal/app/model.go`:

```go
package app

import (
	"fmt"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/key"

	"github.com/lululau/tuinnel/internal/tunnel"
	"github.com/lululau/tuinnel/internal/ui"
	"github.com/lululau/tuinnel/internal/ui/tabs"
)

type appKeyMap struct {
	Tab1   key.Binding
	Tab2   key.Binding
	Tab3   key.Binding
	Tab4   key.Binding
	TabNext key.Binding
	TabPrev key.Binding
	Help   key.Binding
	Quit   key.Binding
}

var appKeys = appKeyMap{
	Tab1:   key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "tunnels")),
	Tab2:   key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "logs")),
	Tab3:   key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "settings")),
	Tab4:   key.NewBinding(key.WithKeys("4"), key.WithHelp("4", "editor")),
	TabNext: key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next tab")),
	TabPrev: key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("⇧+tab", "prev tab")),
	Help:   key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	Quit:   key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}

type confirmQuit struct {
	active bool
}

type Model struct {
	mgr       *tunnel.Manager
	config    *tunnel.Config
	configPath string
	tabBar    tabs.TabBar
	listTab   tabs.tunnelListModel
	logTab    tabs.logModel
	settingsTab tabs.settingsModel
	editorTab tabs.editorModel
	width     int
	height    int
	confirm   confirmQuit
	statusMsg string
	quitting  bool
}

func NewModel(cfg *tunnel.Config, configPath string) Model {
	mgr := tunnel.NewManager(cfg)
	mgr.Refresh()

	m := Model{
		mgr:        mgr,
		config:     cfg,
		configPath: configPath,
		tabBar:     tabs.NewTabBar(),
		listTab:    newTunnelListModel(),
		logTab:     newLogModel(),
		settingsTab: newSettingsModel(cfg.Settings),
		editorTab:  newEditorModel(),
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
		m.listTab.SetSize(msg.Width, msg.Height-4)
		m.logTab.SetSize(msg.Width, msg.Height-4)
		m.settingsTab.SetSize(msg.Width, msg.Height-4)
		m.editorTab.SetSize(msg.Width, msg.Height-4)
		return m, nil

	case tea.KeyPressMsg:
		// Confirm quit dialog
		if m.confirm.active {
			switch msg.String() {
			case "y", "yes":
				m.quitting = true
				if m.config.Settings.KillOnExit {
					_ = m.mgr.StopAll()
				}
				return m, tea.Quit
			case "n", "no", "esc":
				m.confirm.active = false
				return m, nil
			}
			return m, nil
		}

		// Global keys
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

		// Tab-specific keys
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

	case tabs.settingsSavedMsg:
		m.config.Settings = m.settingsTab.Settings()
		if err := tunnel.SaveConfig(m.config, m.configPath); err != nil {
			m.statusMsg = fmt.Sprintf("Save error: %s", err)
		} else {
			m.statusMsg = "Settings saved"
		}
		return m, nil

	case tabs.editorSaveMsg:
		t, err := m.editorTab.Tunnel()
		if err != nil {
			m.editorTab.message = err.Error()
			return m, nil
		}
		if m.editorTab.EditIndex() < 0 {
			// Add new
			m.mgr.AddTunnel(t)
			m.statusMsg = fmt.Sprintf("Tunnel %q added", t.Name)
		} else {
			// Update existing
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

	// Pass to active tab
	return m.updateActiveTab(msg)
}

func (m Model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	content := m.activeTabView()

	// Status bar
	status := m.statusMsg
	if status == "" {
		status = m.listTab.statusText(m.mgr.Tunnels(), m.mgr.RunningCount())
	}
	statusBar := ui.StyleStatusBar.Render(status)

	// Confirm quit overlay
	if m.confirm.active {
		confirm := fmt.Sprintf(
			"\n  %d tunnels are running. Quit and %s them? [y/n]\n",
			m.mgr.RunningCount(),
			map[bool]string{true: "kill", false: "leave"}[m.config.Settings.KillOnExit],
		)
		content += ui.StyleError.Render(confirm)
	}

	return fmt.Sprintf("%s\n%s\n%s\n%s",
		ui.StyleTitle.Render(" tuinnel"),
		m.tabBar.View(),
		content,
		statusBar,
	)
}

func (m *Model) switchTab(id tabs.TabID) {
	m.tabBar.SetActive(id)
	// Update log tab with current logs when switching to it
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

func (m Model) handleTunnelListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
	case "k":
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
			m.editorTab.StartEdit(idx, m.mgr.Tunnels()[idx])
			m.switchTab(tabs.TabEditor)
		}
		return m, nil
	}
	return m.updateActiveTab(msg)
}

func (m Model) handleLogKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k", "down", "j":
		// After moving cursor, update log content
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
```

**Note:** The app model uses `tea.KeyMsg` in `handleTunnelListKeys` — this should be `tea.KeyPressMsg` for bubbletea v2. Also, `lipgloss.Width()` is used in `tablist.go` and needs to be imported as `"charm.land/lipgloss/v2"`.

**Step 2: Verify it compiles**

Run: `go build ./...`
Expected: may have import/compilation errors — fix them iteratively until clean

**Step 3: Commit**

```bash
git add internal/app/
git commit -m "feat: implement app model wiring all tabs and tunnel management"
```

---

## Task 12: Main Entry Point

**Files:**
- Modify: `main.go`

**Step 1: Implement main.go**

Write `main.go`:

```go
package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/lululau/tuinnel/internal/app"
	"github.com/lululau/tuinnel/internal/tunnel"
)

func main() {
	configPath := tunnel.DefaultConfigPath()

	// Allow override via env var
	if p := os.Getenv("SSH_TUN_TUI_CONFIG"); p != "" {
		configPath = p
	}

	// Load config, create default if missing
	cfg, err := loadOrCreateConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %s\n", err)
		os.Exit(1)
	}

	m := app.NewModel(cfg, configPath)

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

func loadOrCreateConfig(path string) (*tunnel.Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := tunnel.SaveConfig(&tunnel.Config{
			Settings: tunnel.DefaultSettings(),
		}, path); err != nil {
			return nil, fmt.Errorf("create default config: %w", err)
		}
		fmt.Printf("Created default config at %s\n", path)
	}
	return tunnel.LoadConfig(path)
}
```

**Step 2: Build and verify**

Run: `go build -o tuinnel .`
Expected: binary created, no errors

**Step 3: Quick smoke test**

Run: `./tuinnel`
Expected: TUI opens with empty tunnel list, Tab bar visible, can navigate tabs with 1-4, q to quit

Press `q` to exit.

**Step 4: Commit**

```bash
git add main.go
git commit -m "feat: implement main entry point with config auto-creation"
```

---

## Task 13: Integration and Compilation Fixes

**Files:**
- Any files with compilation errors

**Step 1: Full build check**

Run: `go build ./...`
Expected: clean build

**Step 2: Run all tests**

Run: `go test ./... -v`
Expected: all PASS

**Step 3: Fix any compilation or test issues**

Address any type mismatches between bubbletea v2 API and the code. Common fixes:
- `tea.KeyMsg` → `tea.KeyPressMsg`
- Missing imports for `lipgloss`, `strings`
- `tea.Model` interface: `Init()` returns `tea.Cmd`, `Update()` returns `(tea.Model, tea.Cmd)`, `View()` returns `tea.View` or `string`

**Step 4: Final smoke test**

Run: `go build -o tuinnel . && ./tuinnel`
Verify:
- Tab bar shows 4 tabs
- Tab 1 (Tunnels) shows empty table
- Tab 2 (Logs) shows "Select a tunnel to view logs"
- Tab 3 (Settings) shows 3 editable fields
- Tab 4 (Editor) shows "a: add new" prompt
- `1-4` keys switch tabs
- `q` quits

**Step 5: Commit**

```bash
git add -A
git commit -m "fix: resolve compilation issues and verify integration"
```

---

## Task 14: Add Sample Config and Final Polish

**Files:**
- Create: `examples/config.toml`

**Step 1: Create example config**

Create `examples/config.toml`:

```toml
# SSH Tunnel TUI Manager Configuration
# Path: ~/.config/tuinnel/config.toml

[settings]
ssh_bin = "ssh"                        # Path to SSH binary
control_dir = "/tmp/tuinnel"       # Control socket directory
kill_on_exit = false                    # Kill tunnels on exit

# Local port forwarding: local_port:remote_host:remote_port
[[tunnels]]
name = "dev-db"
type = "local"
local_port = 3307
remote_host = "localhost"
remote_port = 3306
login = "deploy@db-server"
group = "dev"

# Remote port forwarding: remote_port:remote_host:local_port
[[tunnels]]
name = "remote-api"
type = "remote"
local_port = 8080
remote_host = "localhost"
remote_port = 80
login = "deploy@api-server"
group = "staging"

# Dynamic (SOCKS) proxy
[[tunnels]]
name = "socks-proxy"
type = "dynamic"
local_port = 1080
login = "user@jump-host"
```

**Step 2: Run full test suite**

Run: `go test ./... -v`
Expected: all PASS

**Step 3: Commit**

```bash
git add examples/
git commit -m "docs: add example configuration file"
```

---

## Summary

| Task | Description | Key Files |
|------|------------|-----------|
| 1 | Project scaffolding | go.mod, directory structure |
| 2 | Tunnel types + config parsing | `internal/tunnel/tunnel.go`, `config.go` |
| 3 | SSH client wrapper | `internal/ssh/client.go` |
| 4 | Tunnel manager + ring buffer | `internal/tunnel/manager.go` |
| 5 | UI styles (Catppuccin Mocha) | `internal/ui/styles.go` |
| 6 | Tab bar component | `internal/ui/tabs/tablist.go` |
| 7 | Tunnel list tab | `internal/ui/tabs/tunnel_list.go` |
| 8 | Log panel tab | `internal/ui/tabs/logs.go` |
| 9 | Settings tab | `internal/ui/tabs/settings.go` |
| 10 | Tunnel editor tab | `internal/ui/tabs/editor.go` |
| 11 | App model (wiring) | `internal/app/model.go` |
| 12 | Main entry point | `main.go` |
| 13 | Integration fixes | Various |
| 14 | Example config + polish | `examples/config.toml` |
