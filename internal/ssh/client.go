package ssh

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

type Client struct {
	Bin        string
	ControlDir string
}

func (c Client) SocketPath(name string) string {
	return fmt.Sprintf("%s/%s", c.ControlDir, name)
}

func (c Client) BuildStartArgs(socket, flag, forward, login string) []string {
	return []string{
		"-M", "-f", "-N", "-T",
		"-o", "ConnectTimeout=5",
		"-o", "ServerAliveInterval=15",
		"-o", "ServerAliveCountMax=3",
		"-S", socket, flag, forward, login,
	}
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
	// Create a new session so SSH survives TUI exit
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ssh start: %s (exit: %v)", string(output), err)
	}
	return nil
}

func (c Client) Stop(socket, login string) error {
	args := c.BuildStopArgs(socket, login)
	cmd := exec.Command(c.Bin, args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ssh stop: %s (exit: %v)", string(output), err)
	}
	return nil
}

// KillBySocket finds and kills the SSH master process for the given socket path.
// It matches by command-line arguments since the socket may already be removed.
func (c Client) KillBySocket(socket string) error {
	// Find SSH process by matching the socket path in its command line
	out, err := exec.Command("pgrep", "-f", socket).Output()
	if err == nil && len(out) > 0 {
		pids := strings.Fields(strings.TrimSpace(string(out)))
		for _, pidStr := range pids {
			pid, _ := strconv.Atoi(pidStr)
			if pid > 0 {
				syscall.Kill(pid, syscall.SIGTERM)
			}
		}
	}
	os.Remove(socket)
	return nil
}

func (c Client) Check(socket, login string) bool {
	args := c.BuildCheckArgs(socket, login)
	cmd := exec.Command(c.Bin, args...)
	return cmd.Run() == nil
}

// HasProcess checks whether an SSH process with the given socket path exists.
func (c Client) HasProcess(socket string) bool {
	out, err := exec.Command("pgrep", "-f", socket).Output()
	return err == nil && len(strings.TrimSpace(string(out))) > 0
}

// RemoveSocket removes the control socket file.
func (c Client) RemoveSocket(socket string) error {
	return os.Remove(socket)
}
