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
		name   string
		tunnel Tunnel
		want   string
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
