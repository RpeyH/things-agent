package main

import (
	"context"
	"strings"
	"testing"
)

func TestPlanRestoreState(t *testing.T) {
	target := thingsStateSnapshot{
		SchemaVersion: 1,
		Areas:         []thingsStateArea{{ID: "area-1", Name: "Area A"}},
		Projects: []thingsStateProject{{
			ID:     "project-1",
			Name:   "Project A",
			AreaID: "area-1",
			Area:   "Area A",
			Notes:  "target notes",
			Tags:   []string{"tag-a"},
		}},
		Tasks: []thingsStateTask{{
			ID:        "task-1",
			Name:      "Task A",
			AreaID:    "area-1",
			Area:      "Area A",
			ProjectID: "project-1",
			Project:   "Project A",
			Due:       "2026-03-07 00:00:00",
			Deadline:  "2026-03-08 00:00:00",
			Notes:     "target task",
			Tags:      []string{"tag-a"},
		}},
	}
	current := thingsStateSnapshot{
		SchemaVersion: 1,
		Areas:         []thingsStateArea{{ID: "area-1", Name: "Area Renamed"}},
		Projects: []thingsStateProject{{
			ID:     "project-1",
			Name:   "Project A",
			AreaID: "area-1",
			Area:   "Area Renamed",
			Notes:  "current notes",
			Tags:   []string{"tag-b"},
		}},
	}

	report, err := planRestoreState("2026-03-07:10-10-10", target, current)
	if err != nil {
		t.Fatalf("planRestoreState failed: %v", err)
	}
	if report.TargetSummary.Tasks != 1 || report.CurrentSummary.Tasks != 0 {
		t.Fatalf("unexpected summaries: %#v", report)
	}
	kinds := []string{}
	for _, action := range report.Actions {
		kinds = append(kinds, action.Kind)
	}
	joined := strings.Join(kinds, ",")
	for _, expected := range []string{"rename-area", "move-project", "update-project-notes", "set-project-tags", "create-task"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("expected action %q in %#v", expected, report.Actions)
		}
	}
}

func TestPlanRestoreStateRejectsDuplicateLogicalKeys(t *testing.T) {
	_, err := planRestoreState("2026-03-07:10-10-10", thingsStateSnapshot{
		SchemaVersion: 1,
		Areas: []thingsStateArea{
			{ID: "area-1", Name: "Area A"},
			{ID: "area-2", Name: "Area A"},
		},
	}, thingsStateSnapshot{SchemaVersion: 1})
	if err == nil || !strings.Contains(err.Error(), "duplicate area logical key") {
		t.Fatalf("expected duplicate key error, got %v", err)
	}
}

func TestBuildRestoreStateReport(t *testing.T) {
	runner := runnerFunc(func(_ context.Context, script string) (string, error) {
		if strings.Contains(script, "state snapshot capture") {
			return "A\tarea-1\tArea Renamed", nil
		}
		return "", nil
	})
	tmp := setupTestRuntimeWithDB(t, runner)

	bm := newBackupManager(tmp)
	if _, err := bm.ensureBackupDir(); err != nil {
		t.Fatalf("ensureBackupDir failed: %v", err)
	}
	if err := bm.writeStateSnapshot("2026-03-07:10-10-10", thingsStateSnapshot{
		SchemaVersion: 1,
		Areas:         []thingsStateArea{{ID: "area-1", Name: "Area A"}},
	}); err != nil {
		t.Fatalf("writeStateSnapshot failed: %v", err)
	}

	cfg, err := resolveRuntimeConfig(context.Background())
	if err != nil {
		t.Fatalf("resolveRuntimeConfig failed: %v", err)
	}
	report, err := buildRestoreStateReport(context.Background(), cfg, "2026-03-07:10-10-10")
	if err != nil {
		t.Fatalf("buildRestoreStateReport failed: %v", err)
	}
	if len(report.Actions) != 1 || report.Actions[0].Kind != "rename-area" {
		t.Fatalf("unexpected restore state report: %#v", report)
	}
}
