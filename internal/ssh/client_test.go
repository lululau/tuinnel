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
	c := Client{ControlDir: "/tmp/ssh-tun-tui"}
	if got := c.SocketPath("dev-db"); got != "/tmp/ssh-tun-tui/dev-db" {
		t.Errorf("SocketPath() = %q, want %q", got, "/tmp/ssh-tun-tui/dev-db")
	}
}
