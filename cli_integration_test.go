//go:build integration

package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestIntegrationTagsSearchUsesMockRunner(t *testing.T) {
	fr := &fakeRunner{output: "work, urgent"}
	setupTestRuntime(t, t.TempDir(), fr)

	root := newRootCmd()
	root.SetArgs([]string{"tags", "search", "--query", "wo"})
	if err := root.Execute(); err != nil {
		t.Fatalf("root execute failed: %v", err)
	}

	scripts := fr.allScripts()
	if len(scripts) != 1 {
		t.Fatalf("expected one runner call, got %d", len(scripts))
	}
	if !strings.Contains(scripts[0], "every tag whose name contains") {
		t.Fatalf("unexpected script content: %s", scripts[0])
	}
}

func TestIntegrationAddTaskUsesMockRunnerWithExplicitArea(t *testing.T) {
	fr := &fakeRunner{output: "task-id-1"}
	setupTestRuntimeWithDB(t, fr)

	stdout, err := captureStdout(t, func() error {
		root := newRootCmd()
		root.SetArgs([]string{"add-task", "--name", "integration-task", "--area", "Inbox"})
		return root.Execute()
	})
	if err != nil {
		t.Fatalf("root execute failed: %v", err)
	}
	if !strings.Contains(stdout, "task-id-1") {
		t.Fatalf("expected created task id on stdout, got %q", stdout)
	}

	scripts := fr.allScripts()
	if len(scripts) == 0 {
		t.Fatal("expected mocked runner to be called")
	}
	if !strings.Contains(scripts[0], `set targetList to first list whose name is "Inbox"`) {
		t.Fatalf("unexpected script content: %s", scripts[0])
	}
}

func TestIntegrationRestoreDryRunReturnsStructuredJournal(t *testing.T) {
	tmp := setupTestRuntimeWithDB(t, &fakeRunner{})
	writeLiveDBSet(t, tmp, "live")

	bm := newBackupManager(tmp)
	created, err := bm.Create(context.Background())
	if err != nil {
		t.Fatalf("seed snapshot: %v", err)
	}
	targetTS := inferTimestamp(created[0])

	fr := &fakeRunner{runFn: func(script string) (string, error) {
		switch {
		case strings.Contains(script, "return running"):
			return "false", nil
		default:
			return "ok", nil
		}
	}}
	setupTestRuntime(t, tmp, fr)

	stdout, err := captureStdout(t, func() error {
		root := newRootCmd()
		root.SetArgs([]string{"restore", "--timestamp", targetTS, "--dry-run", "--json"})
		return root.Execute()
	})
	if err != nil {
		t.Fatalf("restore dry-run failed: %v", err)
	}

	var journal map[string]any
	if err := json.Unmarshal([]byte(stdout), &journal); err != nil {
		t.Fatalf("decode restore journal: %v\nstdout=%q", err, stdout)
	}
	if journal["outcome"] != "dry-run" {
		t.Fatalf("expected dry-run outcome, got %#v", journal["outcome"])
	}
	if journal["timestamp"] != targetTS {
		t.Fatalf("expected timestamp %q, got %#v", targetTS, journal["timestamp"])
	}
	preflight, ok := journal["preflight"].(map[string]any)
	if !ok || preflight["ok"] != true {
		t.Fatalf("expected successful preflight report, got %#v", journal["preflight"])
	}
}
