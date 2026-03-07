package main

import (
	"context"
	"fmt"
	"strings"
)

type scriptSemanticSnapshotter struct {
	bundleID string
	runner   scriptRunner
}

func newScriptSemanticSnapshotter(bundleID string, runner scriptRunner) scriptSemanticSnapshotter {
	return scriptSemanticSnapshotter{
		bundleID: bundleID,
		runner:   runner,
	}
}

func (s scriptSemanticSnapshotter) Snapshot(ctx context.Context) (backupSemanticSnapshot, error) {
	out, err := s.runner.run(ctx, scriptSemanticSnapshot(s.bundleID))
	if err != nil {
		return backupSemanticSnapshot{}, fmt.Errorf("run semantic snapshot: %w", err)
	}
	return parseSemanticSnapshot(out)
}

func parseSemanticSnapshot(raw string) (backupSemanticSnapshot, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return backupSemanticSnapshot{}, nil
	}

	lines := strings.Split(raw, "\n")
	lists := make([]string, 0, len(lines))
	projects := make([]string, 0, len(lines))
	tasks := make([]string, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 2 {
			return backupSemanticSnapshot{}, fmt.Errorf("invalid semantic snapshot row %q", line)
		}
		kind := fields[0]
		switch kind {
		case "L":
			if len(fields) != 3 {
				return backupSemanticSnapshot{}, fmt.Errorf("invalid list semantic row %q", line)
			}
			payload := strings.Join(fields[1:], "\t")
			lists = append(lists, payload)
		case "P":
			if len(fields) != 4 {
				return backupSemanticSnapshot{}, fmt.Errorf("invalid project semantic row %q", line)
			}
			payload := strings.Join(fields[1:], "\t")
			projects = append(projects, payload)
		case "T":
			if len(fields) != 2 {
				return backupSemanticSnapshot{}, fmt.Errorf("invalid task semantic row %q", line)
			}
			tasks = append(tasks, fields[1])
		default:
			return backupSemanticSnapshot{}, fmt.Errorf("unknown semantic snapshot row kind %q", kind)
		}
	}

	return backupSemanticSnapshot{
		ListsCount:    len(lists),
		ListsHash:     hashSemanticLines(lists),
		ProjectsCount: len(projects),
		ProjectsHash:  hashSemanticLines(projects),
		TasksCount:    parseSemanticTaskCount(tasks),
		TasksHash:     "",
	}, nil
}

func scriptSemanticSnapshot(bundleID string) string {
	return fmt.Sprintf(`tell application id "%s"
  set outLines to {}
  repeat with l in every list
    set end of outLines to ("L" & tab & (id of l as string) & tab & (name of l))
  end repeat
  repeat with p in every project
    set end of outLines to ("P" & tab & (id of p as string) & tab & (name of p) & tab & (status of p as string))
  end repeat
  set end of outLines to ("T" & tab & ((count of to dos) as string))
  set AppleScript's text item delimiters to linefeed
  return outLines as text
end tell`, bundleID)
}

func compareSemanticSnapshots(expected, actual backupSemanticSnapshot) error {
	switch {
	case expected.ListsCount != actual.ListsCount || expected.ListsHash != actual.ListsHash:
		return fmt.Errorf("list snapshot mismatch: expected count=%d hash=%s got count=%d hash=%s", expected.ListsCount, expected.ListsHash, actual.ListsCount, actual.ListsHash)
	case expected.ProjectsCount != actual.ProjectsCount || expected.ProjectsHash != actual.ProjectsHash:
		return fmt.Errorf("project snapshot mismatch: expected count=%d hash=%s got count=%d hash=%s", expected.ProjectsCount, expected.ProjectsHash, actual.ProjectsCount, actual.ProjectsHash)
	case expected.TasksCount != actual.TasksCount:
		return fmt.Errorf("task snapshot mismatch: expected count=%d hash=%s got count=%d hash=%s", expected.TasksCount, expected.TasksHash, actual.TasksCount, actual.TasksHash)
	case expected.TasksHash != "" && expected.TasksHash != actual.TasksHash:
		return fmt.Errorf("task snapshot mismatch: expected count=%d hash=%s got count=%d hash=%s", expected.TasksCount, expected.TasksHash, actual.TasksCount, actual.TasksHash)
	default:
		return nil
	}
}

func parseSemanticTaskCount(values []string) int {
	if len(values) == 0 {
		return 0
	}
	count := 0
	for _, ch := range strings.TrimSpace(values[0]) {
		if ch < '0' || ch > '9' {
			return 0
		}
		count = count*10 + int(ch-'0')
	}
	return count
}
