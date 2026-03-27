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
	if cfg.Settings.ControlDir != "/tmp/ssh-tun-tui" {
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
	want := filepath.Join(home, ".config", "ssh-tun-tui", "config.toml")
	if got := DefaultConfigPath(); got != want {
		t.Errorf("DefaultConfigPath() = %q, want %q", got, want)
	}
}
