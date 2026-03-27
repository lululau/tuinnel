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
