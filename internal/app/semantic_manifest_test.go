package app

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestParseSemanticManifest(t *testing.T) {
	raw := strings.Join([]string{
		"L\tlist-1\tInbox",
		"P\tproject-1\tProject A\topen",
		"T\ttask-1",
		"T\ttask-2",
	}, "\n")

	got, err := parseSemanticManifest(raw)
	if err != nil {
		t.Fatalf("parseSemanticManifest failed: %v", err)
	}
	if got.ListsCount != 1 || got.ProjectsCount != 1 || got.TasksCount != 2 {
		t.Fatalf("unexpected semantic counts: %#v", got)
	}
	if got.ListsHash == "" || got.ProjectsHash == "" || got.TasksHash == "" {
		t.Fatalf("expected semantic hashes, got %#v", got)
	}
	if len(got.TaskRefs) != 2 || got.TaskRefs[0] != "task-1" || got.TaskRefs[1] != "task-2" {
		t.Fatalf("expected task refs, got %#v", got.TaskRefs)
	}
}

func TestCompareSemanticManifests(t *testing.T) {
	base := backupSemanticManifest{
		ListsCount:    1,
		ListsHash:     "a",
		ProjectsCount: 2,
		ProjectsHash:  "b",
		TasksCount:    3,
		TasksHash:     "c",
	}
	if err := compareSemanticManifests(base, base); err != nil {
		t.Fatalf("expected identical semantic manifests to match: %v", err)
	}

	other := base
	other.TasksHash = "d"
	err := compareSemanticManifests(base, other)
	if err == nil || !strings.Contains(err.Error(), "task manifest mismatch") {
		t.Fatalf("expected task mismatch, got %v", err)
	}
}

func TestCompareSemanticManifestsAllowsCountOnlyActualProbe(t *testing.T) {
	expected := backupSemanticManifest{
		ListsCount:    1,
		ListsHash:     "a",
		ProjectsCount: 2,
		ProjectsHash:  "b",
		TasksCount:    3,
		TasksHash:     "c",
	}
	actual := backupSemanticManifest{
		ListsCount:    1,
		ProjectsCount: 2,
		TasksCount:    3,
	}
	if err := compareSemanticManifests(expected, actual); err != nil {
		t.Fatalf("expected count-only actual probe to pass, got %v", err)
	}
}

func TestCompareSemanticManifestsSummarizesTaskDiffs(t *testing.T) {
	base := backupSemanticManifest{
		TasksCount: 2,
		TasksHash:  "a",
		TaskRefs:   []string{"task-1", "task-2"},
	}
	other := backupSemanticManifest{
		TasksCount: 1,
		TasksHash:  "b",
		TaskRefs:   []string{"task-2"},
	}
	err := compareSemanticManifests(base, other)
	if err == nil || !strings.Contains(err.Error(), "missing=[task-1]") {
		t.Fatalf("expected task diff summary, got %v", err)
	}
}

func TestParseSemanticHealthManifest(t *testing.T) {
	raw := strings.Join([]string{
		"L\t4",
		"P\t2",
		"T\t9",
	}, "\n")
	got, err := parseSemanticHealthManifest(raw)
	if err != nil {
		t.Fatalf("parseSemanticHealthManifest failed: %v", err)
	}
	if got.ListsCount != 4 || got.ProjectsCount != 2 || got.TasksCount != 9 {
		t.Fatalf("unexpected semantic health manifest: %#v", got)
	}
	if got.ListsHash != "" || got.ProjectsHash != "" || got.TasksHash != "" {
		t.Fatalf("expected count-only semantic health manifest, got %#v", got)
	}
}

func TestSemanticManifestProbeSnapshotErrors(t *testing.T) {
	probe := newScriptSemanticManifestProbe(defaultBundleID, runnerFunc(func(context.Context, string) (string, error) {
		return "", errors.New("boom")
	}))
	if _, err := probe.Snapshot(context.Background()); err == nil || !strings.Contains(err.Error(), "run semantic manifest") {
		t.Fatalf("expected wrapped runner error, got %v", err)
	}

	badProbe := scriptSemanticManifestProbe{
		bundleID: defaultBundleID,
		runner: runnerFunc(func(context.Context, string) (string, error) {
			return "bad-row", nil
		}),
		script: scriptSemanticManifest,
		parse:  parseSemanticManifest,
	}
	if _, err := badProbe.Snapshot(context.Background()); err == nil || !strings.Contains(err.Error(), "invalid semantic manifest row") {
		t.Fatalf("expected parse error from snapshot, got %v", err)
	}
}

func TestParseSemanticManifestRejectsInvalidRows(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{name: "short row", raw: "only-one-field", want: "invalid semantic manifest row"},
		{name: "bad list row", raw: "L\ttoo-short", want: "invalid list semantic row"},
		{name: "bad project row", raw: "P\tproject\tname", want: "invalid project semantic row"},
		{name: "bad task row", raw: "T\ttask\textra", want: "invalid task semantic row"},
		{name: "unknown row kind", raw: "X\tvalue\tother", want: "unknown semantic manifest row kind"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseSemanticManifest(tc.raw)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q, got %v", tc.want, err)
			}
		})
	}
}

func TestCompareSemanticManifestsReportsListAndProjectMismatches(t *testing.T) {
	base := backupSemanticManifest{
		ListsCount:    1,
		ListsHash:     "list-a",
		ProjectsCount: 1,
		ProjectsHash:  "project-a",
		TasksCount:    1,
		TasksHash:     "task-a",
	}

	listMismatch := base
	listMismatch.ListsHash = "list-b"
	if err := compareSemanticManifests(base, listMismatch); err == nil || !strings.Contains(err.Error(), "list manifest mismatch") {
		t.Fatalf("expected list manifest mismatch, got %v", err)
	}

	projectMismatch := base
	projectMismatch.ProjectsCount = 2
	if err := compareSemanticManifests(base, projectMismatch); err == nil || !strings.Contains(err.Error(), "project manifest mismatch") {
		t.Fatalf("expected project manifest mismatch, got %v", err)
	}
}

func TestParseSemanticHealthManifestRejectsInvalidRows(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{name: "short row", raw: "L", want: "invalid semantic health row"},
		{name: "unknown kind", raw: "X\t3", want: "unknown semantic health row kind"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseSemanticHealthManifest(tc.raw)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q, got %v", tc.want, err)
			}
		})
	}
}

func TestParseSemanticCountAndRefsHelpers(t *testing.T) {
	if got := parseSemanticCount("12"); got != 12 {
		t.Fatalf("expected parseSemanticCount to parse digits, got %d", got)
	}
	if got := parseSemanticCount("12x"); got != 0 {
		t.Fatalf("expected invalid semantic count to return 0, got %d", got)
	}

	gotRefs := semanticRefs([]string{" task-2 ", "", "task-1"})
	if strings.Join(gotRefs, ",") != "task-2,task-1" {
		t.Fatalf("unexpected semanticRefs output: %#v", gotRefs)
	}

	summary := summarizeSemanticRefs([]string{"task-6", "task-1", "task-5", "task-2", "task-4", "task-3"})
	if !strings.Contains(summary, "task-1") || !strings.Contains(summary, "+1 more") {
		t.Fatalf("unexpected summarizeSemanticRefs output: %q", summary)
	}
}
