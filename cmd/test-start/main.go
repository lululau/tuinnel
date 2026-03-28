package main

import (
	"fmt"
	"os"

	"github.com/lululau/tuinnel/internal/tunnel"
)

func main() {
	configPath := tunnel.DefaultConfigPath()
	cfg, err := tunnel.LoadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Load config error: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Config loaded: %d tunnels\n", len(cfg.Tunnels))
	fmt.Printf("Settings: SSHBin=%q ControlDir=%q\n", cfg.Settings.SSHBin, cfg.Settings.ControlDir)

	mgr := tunnel.NewManager(cfg)

	name := "lx-sd-16-clawdbot"
	fmt.Printf("\n--- Starting tunnel %q ---\n", name)

	err = mgr.Start(name)
	if err != nil {
		fmt.Printf("START ERROR: %s\n", err)
	} else {
		fmt.Printf("START OK\n")
	}

	// Check tunnel state
	tunnels := mgr.Tunnels()
	for _, t := range tunnels {
		if t.Name == name {
			fmt.Printf("Tunnel state: Running=%v Error=%v\n", t.Running, t.Error)
		}
	}

	// Check log
	log := mgr.Log(name)
	lines := log.Lines()
	for _, line := range lines {
		fmt.Printf("LOG: %s\n", line)
	}

	// Check socket
	socket := mgr.Client().SocketPath(name)
	if _, err := os.Stat(socket); err != nil {
		fmt.Printf("Socket NOT found: %s\n", socket)
	} else {
		fmt.Printf("Socket found: %s\n", socket)
	}

	// Check via SSH
	if mgr.Client().Check(socket, "lx") {
		fmt.Println("SSH Check: OK")
	} else {
		fmt.Println("SSH Check: FAILED")
	}

	// Cleanup
	fmt.Println("\n--- Stopping tunnel ---")
	_ = mgr.Stop(name)
}
