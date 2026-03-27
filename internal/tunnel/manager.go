package tunnel

import (
	"fmt"
	"os"
	"sync"

	"github.com/ssh-tun-tui/internal/ssh"
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

func (m *Manager) Client() ssh.Client {
	return m.client
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
	_ = m.Stop(name)
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
