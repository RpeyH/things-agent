package main

import (
	"strings"
	"testing"
)

func TestParseSemanticSnapshot(t *testing.T) {
	raw := strings.Join([]string{
		"L\tlist-1\tInbox",
		"P\tproject-1\tProject A\topen",
		"T\t42",
	}, "\n")

	got, err := parseSemanticSnapshot(raw)
	if err != nil {
		t.Fatalf("parseSemanticSnapshot failed: %v", err)
	}
	if got.ListsCount != 1 || got.ProjectsCount != 1 || got.TasksCount != 42 {
		t.Fatalf("unexpected semantic counts: %#v", got)
	}
	if got.ListsHash == "" || got.ProjectsHash == "" {
		t.Fatalf("expected semantic hashes, got %#v", got)
	}
	if got.TasksHash != "" {
		t.Fatalf("expected task hash to stay empty in count-only mode, got %#v", got)
	}
}

func TestCompareSemanticSnapshots(t *testing.T) {
	base := backupSemanticSnapshot{
		ListsCount:    1,
		ListsHash:     "a",
		ProjectsCount: 2,
		ProjectsHash:  "b",
		TasksCount:    3,
		TasksHash:     "c",
	}
	if err := compareSemanticSnapshots(base, base); err != nil {
		t.Fatalf("expected identical semantic snapshots to match: %v", err)
	}

	other := base
	other.TasksHash = "d"
	err := compareSemanticSnapshots(base, other)
	if err == nil || !strings.Contains(err.Error(), "task snapshot mismatch") {
		t.Fatalf("expected task mismatch, got %v", err)
	}
}
