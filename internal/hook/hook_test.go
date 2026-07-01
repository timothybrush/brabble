package hook

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"brabble/internal/config"
	"brabble/internal/logging"
)

func TestShouldRunCooldown(t *testing.T) {
	cfg, _ := config.Default()
	cfg.Hooks = []config.HookConfig{{
		Command:     "/bin/echo",
		CooldownSec: 0.5,
	}}
	r := NewRunner(cfg, logging.NewTestLogger())
	r.SelectHook(&cfg.Hooks[0])

	if !r.ShouldRun() {
		t.Fatalf("first call should run")
	}
	if err := r.Run(context.Background(), Job{Text: "test", Timestamp: time.Now()}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if r.ShouldRun() {
		t.Fatalf("cooldown should block immediate subsequent run")
	}
	time.Sleep(time.Duration(cfg.Hook.CooldownSec*float64(time.Second)) + 20*time.Millisecond)
	if !r.ShouldRun() {
		t.Fatalf("should run after cooldown")
	}
}

func TestRunUsesPrefixAndEnv(t *testing.T) {
	cfg, _ := config.Default()
	cfg.Hooks = []config.HookConfig{{
		Command: "/bin/echo",
		Args:    []string{},
		Prefix:  "pref:",
	}}

	r := NewRunner(cfg, logging.NewTestLogger())
	r.SelectHook(&cfg.Hooks[0])
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := r.Run(ctx, Job{Text: "hello", Timestamp: time.Now()}); err != nil {
		t.Fatalf("run echo: %v", err)
	}
}

func TestRunDoesNotLogEnvironmentValues(t *testing.T) {
	var logs bytes.Buffer
	cfg, _ := config.Default()
	cfg.Hooks = []config.HookConfig{{
		Command: "/usr/bin/true",
		Env:     map[string]string{"BRABBLE_TEST_SECRET": "do-not-log-this"},
	}}
	r := NewRunner(cfg, &logging.Logger{Logger: slog.New(slog.NewTextHandler(&logs, nil))})
	r.SelectHook(&cfg.Hooks[0])

	if err := r.Run(context.Background(), Job{Text: "test", Timestamp: time.Now()}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if bytes.Contains(logs.Bytes(), []byte("do-not-log-this")) {
		t.Fatalf("secret environment value was logged: %s", logs.String())
	}
	if !bytes.Contains(logs.Bytes(), []byte("BRABBLE_TEST_SECRET")) {
		t.Fatalf("environment key missing from log: %s", logs.String())
	}
}

func TestSelectHookConfigFallsBackToSingleHook(t *testing.T) {
	cfg, _ := config.Default()
	cfg.Hook.Command = "/bin/echo"

	hk, index := SelectHookConfig(cfg, "make it so")
	if hk == nil || hk.Command != cfg.Hook.Command || index != 0 {
		t.Fatalf("single hook fallback failed: hook=%+v index=%d", hk, index)
	}
}
