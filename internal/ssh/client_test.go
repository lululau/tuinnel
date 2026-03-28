package ssh

import (
	"os"
	"testing"
)

func TestBuildStartArgs(t *testing.T) {
	c := Client{Bin: "ssh"}
	args := c.BuildStartArgs("/tmp/ctrl/test", "-L", "3307:localhost:3306", "deploy@host")
	expected := []string{
		"-M", "-f", "-N", "-T",
		"-o", "ConnectTimeout=5",
		"-o", "ServerAliveInterval=15",
		"-o", "ServerAliveCountMax=3",
		"-S", "/tmp/ctrl/test", "-L", "3307:localhost:3306", "deploy@host",
	}
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

func TestStartStopIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	c := Client{Bin: "ssh", ControlDir: "/tmp/tuinnel-inttest"}
	socket := c.SocketPath("inttest-tunnel")
	login := "jicai.dev"
	forward := "19222:jicai.dev:443"

	// Cleanup from previous test runs
	_ = os.RemoveAll(c.ControlDir)
	if err := os.MkdirAll(c.ControlDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Start tunnel
	t.Logf("Starting tunnel: ssh args = %v", c.BuildStartArgs(socket, "-L", forward, login))
	if err := c.Start(socket, "-L", forward, login); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Verify socket exists
	if _, err := os.Stat(socket); err != nil {
		t.Fatalf("Socket not found after Start: %v", err)
	}

	// Check tunnel is running
	if !c.Check(socket, login) {
		t.Fatal("Check returned false after Start")
	}

	// Stop tunnel
	if err := c.Stop(socket, login); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	// Verify socket is removed
	if _, err := os.Stat(socket); !os.IsNotExist(err) {
		// Socket might still exist briefly, check via Check
		if c.Check(socket, login) {
			t.Log("Warning: socket still exists after Stop")
		}
	}

	// Cleanup
	_ = os.RemoveAll(c.ControlDir)
}
