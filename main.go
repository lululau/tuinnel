package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/ssh-tun-tui/internal/app"
	"github.com/ssh-tun-tui/internal/tunnel"
)

func main() {
	configPath := tunnel.DefaultConfigPath()

	if p := os.Getenv("SSH_TUN_TUI_CONFIG"); p != "" {
		configPath = p
	}

	cfg, err := loadOrCreateConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %s\n", err)
		os.Exit(1)
	}

	m := app.NewModel(cfg, configPath)

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

func loadOrCreateConfig(path string) (*tunnel.Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := tunnel.SaveConfig(&tunnel.Config{
			Settings: tunnel.DefaultSettings(),
		}, path); err != nil {
			return nil, fmt.Errorf("create default config: %w", err)
		}
		fmt.Printf("Created default config at %s\n", path)
	}
	return tunnel.LoadConfig(path)
}
