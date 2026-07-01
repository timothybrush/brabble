package control

import (
	"path/filepath"
	"testing"

	"brabble/internal/config"
)

func TestTestHookUsesSingleHookConfiguration(t *testing.T) {
	dir := t.TempDir()
	cfg, err := config.Default()
	if err != nil {
		t.Fatalf("default config: %v", err)
	}
	cfg.Hook.Command = "/usr/bin/true"
	cfg.Paths.StateDir = dir
	cfg.Paths.LogPath = filepath.Join(dir, "brabble.log")
	cfg.Paths.TranscriptPath = filepath.Join(dir, "transcripts.log")
	cfg.Paths.SocketPath = filepath.Join(dir, "brabble.sock")
	cfg.Paths.PidPath = filepath.Join(dir, "brabble.pid")
	configPath := filepath.Join(dir, "config.toml")
	if err := config.Save(cfg, configPath); err != nil {
		t.Fatalf("save config: %v", err)
	}

	cmd := NewTestHookCmd(&configPath)
	cmd.SetArgs([]string{"behavior probe"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("test-hook: %v", err)
	}
}
