package main

import (
	"strings"
	"testing"
)

func TestOpenCloseCommands(t *testing.T) {
	fr := &fakeRunner{output: "ok"}
	setupTestRuntime(t, t.TempDir(), fr)

	openStdout, err := captureStdout(t, func() error {
		cmd := newOpenCmd()
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	if strings.TrimSpace(openStdout) != "ok" {
		t.Fatalf("expected open stdout ok, got %q", openStdout)
	}

	closeStdout, err := captureStdout(t, func() error {
		cmd := newCloseCmd()
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("close failed: %v", err)
	}
	if strings.TrimSpace(closeStdout) != "ok" {
		t.Fatalf("expected close stdout ok, got %q", closeStdout)
	}

	scripts := strings.Join(fr.allScripts(), "\n")
	if !strings.Contains(scripts, "activate") {
		t.Fatalf("expected activate script, got %s", scripts)
	}
	if !strings.Contains(scripts, "quit") {
		t.Fatalf("expected quit script, got %s", scripts)
	}
}
