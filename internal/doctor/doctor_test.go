package doctor

import (
	"os"
	"path/filepath"
	"testing"

	"brabble/internal/config"
)

func TestCheckHookExecutablePath(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "hook.sh")
	if err := os.WriteFile(target, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	res := checkHookExecutable(target)
	if !res.Pass {
		t.Fatalf("expected pass, got %v", res.Detail)
	}
}

func TestCheckHookExecutableDirectory(t *testing.T) {
	dir := t.TempDir()
	res := checkHookExecutable(dir)
	if res.Pass {
		t.Fatalf("expected fail for directory")
	}
}

func TestCheckHookExecutableNotSet(t *testing.T) {
	res := checkHookExecutable("")
	if res.Pass {
		t.Fatalf("expected fail for empty")
	}
}

func TestRunChecksEveryPerWakeHook(t *testing.T) {
	dir := t.TempDir()
	cfg, err := config.Default()
	if err != nil {
		t.Fatalf("default: %v", err)
	}
	cfg.Paths.ConfigPath = filepath.Join(dir, "config.toml")
	cfg.ASR.ModelPath = filepath.Join(dir, "model.bin")
	cfg.Hooks = []config.HookConfig{{Command: "/usr/bin/true"}, {Command: filepath.Join(dir, "missing")}}
	if err := os.WriteFile(cfg.Paths.ConfigPath, nil, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.WriteFile(cfg.ASR.ModelPath, nil, 0o600); err != nil {
		t.Fatalf("write model: %v", err)
	}

	results := Run(cfg)
	var sawFirst, sawSecond bool
	for _, result := range results {
		switch result.Name {
		case "hooks[0].command":
			sawFirst = result.Pass
		case "hooks[1].command":
			sawSecond = !result.Pass
		}
	}
	if !sawFirst || !sawSecond {
		t.Fatalf("multi-hook doctor results missing: %+v", results)
	}
}
