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
	Stale   bool `toml:"-"` // socket or process exists without the other
}

func (t Tunnel) ForwardSpec() string {
	switch t.Type {
	case TunnelDynamic:
		return fmt.Sprintf("%d", t.LocalPort)
	case TunnelRemote:
		return fmt.Sprintf("%d:%s:%d", t.RemotePort, t.remoteHost(), t.LocalPort)
	default:
		return fmt.Sprintf("%d:%s:%d", t.LocalPort, t.remoteHost(), t.RemotePort)
	}
}

func (t Tunnel) remoteHost() string {
	if t.RemoteHost == "" {
		return "localhost"
	}
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
