package tunnel

import "testing"

func TestManagerSocketPath(t *testing.T) {
	m := NewManager(&Config{
		Settings: Settings{SSHBin: "ssh", ControlDir: "/tmp/test"},
		Tunnels:  []Tunnel{{Name: "t1", Login: "u@h", Type: TunnelLocal, LocalPort: 1, RemotePort: 1}},
	})
	if got := m.Client().SocketPath("t1"); got != "/tmp/test/t1" {
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
	rb.Add("line4")

	lines := rb.Lines()
	if len(lines) != 3 {
		t.Fatalf("len(lines) = %d, want 3", len(lines))
	}
	if lines[0] != "line2" {
		t.Errorf("lines[0] = %q, want %q", lines[0], "line2")
	}
}
