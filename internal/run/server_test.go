package run

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"brabble/internal/config"
	"brabble/internal/hook"
	"brabble/internal/logging"
)

func TestSelectHookConfigMatchesWakeTokens(t *testing.T) {
	cfg, _ := config.Default()
	cfg.Hooks = []config.HookConfig{
		{Wake: []string{"alpha"}, Command: "/bin/echo"},
		{Wake: []string{"clawd", "claude", "cloud"}, Command: "/bin/echo"},
	}

	if hk, _ := hook.SelectHookConfig(cfg, "Claude can you hear me"); hk == nil {
		t.Fatalf("expected hook match for Claude")
	}
	if hk, _ := hook.SelectHookConfig(cfg, "hello alpha"); hk == nil {
		t.Fatalf("expected hook match for alpha")
	}
	if hk, _ := hook.SelectHookConfig(cfg, "no wake here"); hk != &cfg.Hooks[0] {
		t.Fatalf("expected fallback to first hook")
	}
}

func TestWakeMatchesAliases(t *testing.T) {
	if !wakeMatches("hi Claude", "clawd", []string{"claude", "cloud"}) {
		t.Fatalf("expected alias match")
	}
	if wakeMatches("hi there", "clawd", []string{"claude"}) {
		t.Fatalf("expected no match")
	}
}

func TestControlLoopStopsOnCancellation(t *testing.T) {
	socketFile, err := os.CreateTemp("/tmp", "brabble-*.sock")
	if err != nil {
		t.Fatalf("create socket path: %v", err)
	}
	socketPath := socketFile.Name()
	if err := socketFile.Close(); err != nil {
		t.Fatalf("close socket path: %v", err)
	}
	if err := os.Remove(socketPath); err != nil {
		t.Fatalf("remove socket placeholder: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(socketPath) })

	cfg := &config.Config{}
	cfg.Paths.SocketPath = socketPath
	srv := &Server{cfg: cfg, logger: logging.NewTestLogger()}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		srv.controlLoop(ctx)
		close(done)
	}()

	deadline := time.Now().Add(time.Second)
	for {
		if info, err := os.Stat(cfg.Paths.SocketPath); err == nil {
			if got := info.Mode().Perm(); got != 0o600 {
				t.Fatalf("control socket mode = %o, want 600", got)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("control socket was not created")
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("control loop did not stop")
	}
	if _, err := os.Stat(cfg.Paths.SocketPath); !os.IsNotExist(err) {
		t.Fatalf("control socket still exists: %v", err)
	}
}

func TestHandleConnLogsMalformedRequest(t *testing.T) {
	var logs bytes.Buffer
	logger := &logging.Logger{Logger: slog.New(slog.NewTextHandler(&logs, nil))}
	srv := &Server{logger: logger}
	serverConn, clientConn := net.Pipe()
	done := make(chan struct{})
	go func() {
		srv.handleConn(context.Background(), serverConn)
		close(done)
	}()

	if _, err := io.WriteString(clientConn, "{not-json}\n"); err != nil {
		t.Fatalf("write request: %v", err)
	}
	if err := clientConn.Close(); err != nil {
		t.Fatalf("close client: %v", err)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("connection handler did not stop")
	}
	if !bytes.Contains(logs.Bytes(), []byte("control unmarshal")) {
		t.Fatalf("missing control error log: %s", logs.String())
	}
}

func TestRecordTranscriptLogsOpenFailure(t *testing.T) {
	var logs bytes.Buffer
	logger := &logging.Logger{Logger: slog.New(slog.NewTextHandler(&logs, nil))}
	cfg := &config.Config{}
	cfg.Transcripts.Enabled = true
	cfg.UI.StatusTail = 10
	cfg.Paths.TranscriptPath = t.TempDir()
	srv := &Server{cfg: cfg, logger: logger}

	srv.recordTranscript("release proof")

	if !bytes.Contains(logs.Bytes(), []byte("open transcript")) {
		t.Fatalf("missing transcript error log: %s", logs.String())
	}
}

func TestRecordTranscriptRestrictsPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "transcripts.log")
	if err := os.WriteFile(path, []byte("old transcript\n"), 0o644); err != nil {
		t.Fatalf("write existing transcript: %v", err)
	}
	cfg := &config.Config{}
	cfg.Transcripts.Enabled = true
	cfg.UI.StatusTail = 10
	cfg.Paths.TranscriptPath = path
	srv := &Server{cfg: cfg, logger: logging.NewTestLogger()}

	srv.recordTranscript("private transcript")

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat transcript: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("transcript mode = %o, want 600", got)
	}
}
