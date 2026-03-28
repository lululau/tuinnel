package tunnel

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Settings struct {
	SSHBin     string `toml:"ssh_bin"`
	ControlDir string `toml:"control_dir"`
	KillOnExit bool   `toml:"kill_on_exit"`
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
