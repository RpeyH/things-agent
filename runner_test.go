package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNewRunner(t *testing.T) {
	r := newRunner(defaultBundleID)
	if r == nil {
		t.Fatal("expected runner")
	}
}

func TestRunnerEnsureReachableAndRunErrorPaths(t *testing.T) {
	r := newRunner("com.invalid.bundle")
	if err := r.ensureReachable(context.Background()); err == nil {
		t.Fatal("expected ensureReachable error with invalid bundle")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := r.run(ctx, `return "ok"`)
	if err == nil {
		t.Fatal("expected run error with canceled context")
	}
}

func TestRunnerEnsureReachableAndRunSuccessWithFakeOsaScript(t *testing.T) {
	tmp := t.TempDir()
	fake := filepath.Join(tmp, "osascript")
	script := "#!/bin/sh\necho runner-ok\n"
	if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake osascript failed: %v", err)
	}
	t.Setenv("PATH", tmp+":"+os.Getenv("PATH"))

	r := newRunner(defaultBundleID)
	if err := r.ensureReachable(context.Background()); err != nil {
		t.Fatalf("ensureReachable should succeed with fake osascript: %v", err)
	}
	out, err := r.run(context.Background(), `return "ok"`)
	if err != nil {
		t.Fatalf("run should succeed with fake osascript: %v", err)
	}
	if out != "runner-ok" {
		t.Fatalf("unexpected output: %q", out)
	}
}
