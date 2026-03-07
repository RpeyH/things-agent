package main

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

type stateSnapshotSummary struct {
	Areas    int `json:"areas"`
	Projects int `json:"projects"`
	Tasks    int `json:"tasks"`
}

type stateRestoreAction struct {
	Kind       string            `json:"kind"`
	EntityType string            `json:"entity_type"`
	Match      string            `json:"match,omitempty"`
	TargetID   string            `json:"target_id,omitempty"`
	CurrentID  string            `json:"current_id,omitempty"`
	Name       string            `json:"name"`
	Parent     string            `json:"parent,omitempty"`
	Details    map[string]string `json:"details,omitempty"`
}

type stateRestoreReport struct {
	Timestamp      string               `json:"timestamp"`
	DryRun         bool                 `json:"dry_run"`
	TargetSummary  stateSnapshotSummary `json:"target_summary"`
	CurrentSummary stateSnapshotSummary `json:"current_summary"`
	Actions        []stateRestoreAction `json:"actions"`
	Warnings       []string             `json:"warnings,omitempty"`
}

type stateIndex struct {
	areasByID      map[string]thingsStateArea
	areasByLogical map[string]thingsStateArea

	projectsByID      map[string]thingsStateProject
	projectsByLogical map[string]thingsStateProject

	tasksByID      map[string]thingsStateTask
	tasksByLogical map[string]thingsStateTask
}

func newRestoreStateCmd() *cobra.Command {
	var timestamp string
	var dryRun bool
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "restore-state",
		Short: "Plan a surgical restore from a saved state snapshot",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !dryRun {
				return fmt.Errorf("restore-state currently requires --dry-run")
			}
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			report, err := buildRestoreStateReport(ctx, cfg, timestamp)
			if err != nil {
				return err
			}
			report.DryRun = true
			if jsonOutput {
				return writeJSON(report)
			}
			for _, action := range report.Actions {
				fmt.Printf("%s\t%s\t%s\n", action.Kind, action.EntityType, action.Name)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&timestamp, "timestamp", "", "Backup timestamp providing the saved state snapshot (YYYY-MM-DD:HH-MM-SS)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print the surgical restore plan without mutating Things")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output structured JSON")
	_ = cmd.MarkFlagRequired("timestamp")
	return cmd
}

func buildRestoreStateReport(ctx context.Context, cfg *runtimeConfig, timestamp string) (stateRestoreReport, error) {
	backups := newBackupManager(cfg.dataDir)
	target, err := backups.loadStateSnapshot(strings.TrimSpace(timestamp))
	if err != nil {
		return stateRestoreReport{}, fmt.Errorf("load state snapshot %s: %w", strings.TrimSpace(timestamp), err)
	}
	current, err := newScriptStateSnapshotter(cfg.bundleID, cfg.runner).Snapshot(ctx)
	if err != nil {
		return stateRestoreReport{}, err
	}
	return planRestoreState(strings.TrimSpace(timestamp), target, current)
}

func planRestoreState(timestamp string, target, current thingsStateSnapshot) (stateRestoreReport, error) {
	targetIndex, err := newStateIndex(target)
	if err != nil {
		return stateRestoreReport{}, fmt.Errorf("target snapshot: %w", err)
	}
	currentIndex, err := newStateIndex(current)
	if err != nil {
		return stateRestoreReport{}, fmt.Errorf("current snapshot: %w", err)
	}

	report := stateRestoreReport{
		Timestamp:      timestamp,
		TargetSummary:  summarizeStateSnapshot(target),
		CurrentSummary: summarizeStateSnapshot(current),
	}

	for _, area := range target.Areas {
		currentArea, match := matchArea(area, currentIndex)
		switch match {
		case "":
			report.Actions = append(report.Actions, stateRestoreAction{
				Kind:       "create-area",
				EntityType: "area",
				TargetID:   area.ID,
				Name:       area.Name,
			})
		default:
			if strings.TrimSpace(currentArea.Name) != strings.TrimSpace(area.Name) {
				report.Actions = append(report.Actions, stateRestoreAction{
					Kind:       "rename-area",
					EntityType: "area",
					Match:      match,
					TargetID:   area.ID,
					CurrentID:  currentArea.ID,
					Name:       area.Name,
					Details: map[string]string{
						"from": currentArea.Name,
						"to":   area.Name,
					},
				})
			}
		}
	}

	for _, project := range target.Projects {
		currentProject, match := matchProject(project, currentIndex)
		switch match {
		case "":
			report.Actions = append(report.Actions, stateRestoreAction{
				Kind:       "create-project",
				EntityType: "project",
				TargetID:   project.ID,
				Name:       project.Name,
				Parent:     project.Area,
			})
		default:
			if strings.TrimSpace(currentProject.Area) != strings.TrimSpace(project.Area) {
				report.Actions = append(report.Actions, stateRestoreAction{
					Kind:       "move-project",
					EntityType: "project",
					Match:      match,
					TargetID:   project.ID,
					CurrentID:  currentProject.ID,
					Name:       project.Name,
					Parent:     project.Area,
				})
			}
			if strings.TrimSpace(currentProject.Name) != strings.TrimSpace(project.Name) {
				report.Actions = append(report.Actions, stateRestoreAction{
					Kind:       "rename-project",
					EntityType: "project",
					Match:      match,
					TargetID:   project.ID,
					CurrentID:  currentProject.ID,
					Name:       project.Name,
					Parent:     project.Area,
					Details: map[string]string{
						"from": currentProject.Name,
						"to":   project.Name,
					},
				})
			}
			if strings.TrimSpace(currentProject.Notes) != strings.TrimSpace(project.Notes) {
				report.Actions = append(report.Actions, stateRestoreAction{
					Kind:       "update-project-notes",
					EntityType: "project",
					Match:      match,
					TargetID:   project.ID,
					CurrentID:  currentProject.ID,
					Name:       project.Name,
					Parent:     project.Area,
				})
			}
			if !sameStringSet(currentProject.Tags, project.Tags) {
				report.Actions = append(report.Actions, stateRestoreAction{
					Kind:       "set-project-tags",
					EntityType: "project",
					Match:      match,
					TargetID:   project.ID,
					CurrentID:  currentProject.ID,
					Name:       project.Name,
					Parent:     project.Area,
					Details: map[string]string{
						"tags": strings.Join(normalizeStringSet(project.Tags), ", "),
					},
				})
			}
		}
	}

	for _, task := range target.Tasks {
		if !taskHasSupportedRestoreContainer(task) {
			report.Warnings = append(report.Warnings, fmt.Sprintf("task %q has no supported area/project container and was skipped", task.Name))
			continue
		}
		currentTask, match := matchTask(task, currentIndex)
		switch match {
		case "":
			parent := task.Area
			if strings.TrimSpace(task.Project) != "" {
				parent = task.Project
			}
			report.Actions = append(report.Actions, stateRestoreAction{
				Kind:       "create-task",
				EntityType: "task",
				TargetID:   task.ID,
				Name:       task.Name,
				Parent:     parent,
			})
		default:
			targetParent := task.Area
			currentParent := currentTask.Area
			if strings.TrimSpace(task.Project) != "" {
				targetParent = task.Project
			}
			if strings.TrimSpace(currentTask.Project) != "" {
				currentParent = currentTask.Project
			}
			if strings.TrimSpace(currentParent) != strings.TrimSpace(targetParent) {
				report.Actions = append(report.Actions, stateRestoreAction{
					Kind:       "move-task",
					EntityType: "task",
					Match:      match,
					TargetID:   task.ID,
					CurrentID:  currentTask.ID,
					Name:       task.Name,
					Parent:     targetParent,
				})
			}
			if strings.TrimSpace(currentTask.Name) != strings.TrimSpace(task.Name) {
				report.Actions = append(report.Actions, stateRestoreAction{
					Kind:       "rename-task",
					EntityType: "task",
					Match:      match,
					TargetID:   task.ID,
					CurrentID:  currentTask.ID,
					Name:       task.Name,
					Parent:     targetParent,
					Details: map[string]string{
						"from": currentTask.Name,
						"to":   task.Name,
					},
				})
			}
			if strings.TrimSpace(currentTask.Notes) != strings.TrimSpace(task.Notes) {
				report.Actions = append(report.Actions, stateRestoreAction{
					Kind:       "update-task-notes",
					EntityType: "task",
					Match:      match,
					TargetID:   task.ID,
					CurrentID:  currentTask.ID,
					Name:       task.Name,
					Parent:     targetParent,
				})
			}
			if !sameStringSet(currentTask.Tags, task.Tags) {
				report.Actions = append(report.Actions, stateRestoreAction{
					Kind:       "set-task-tags",
					EntityType: "task",
					Match:      match,
					TargetID:   task.ID,
					CurrentID:  currentTask.ID,
					Name:       task.Name,
					Parent:     targetParent,
					Details: map[string]string{
						"tags": strings.Join(normalizeStringSet(task.Tags), ", "),
					},
				})
			}
			if strings.TrimSpace(currentTask.Due) != strings.TrimSpace(task.Due) || strings.TrimSpace(currentTask.Deadline) != strings.TrimSpace(task.Deadline) {
				report.Actions = append(report.Actions, stateRestoreAction{
					Kind:       "set-task-dates",
					EntityType: "task",
					Match:      match,
					TargetID:   task.ID,
					CurrentID:  currentTask.ID,
					Name:       task.Name,
					Parent:     targetParent,
					Details: map[string]string{
						"due":      task.Due,
						"deadline": task.Deadline,
					},
				})
			}
		}
	}

	sort.SliceStable(report.Actions, func(i, j int) bool {
		return stateSortKey(report.Actions[i].EntityType, report.Actions[i].Kind, report.Actions[i].Parent, report.Actions[i].Name, report.Actions[i].TargetID) <
			stateSortKey(report.Actions[j].EntityType, report.Actions[j].Kind, report.Actions[j].Parent, report.Actions[j].Name, report.Actions[j].TargetID)
	})
	sort.Strings(report.Warnings)
	_ = targetIndex
	return report, nil
}

func summarizeStateSnapshot(snapshot thingsStateSnapshot) stateSnapshotSummary {
	return stateSnapshotSummary{
		Areas:    len(snapshot.Areas),
		Projects: len(snapshot.Projects),
		Tasks:    len(snapshot.Tasks),
	}
}

func newStateIndex(snapshot thingsStateSnapshot) (stateIndex, error) {
	index := stateIndex{
		areasByID:         make(map[string]thingsStateArea, len(snapshot.Areas)),
		areasByLogical:    make(map[string]thingsStateArea, len(snapshot.Areas)),
		projectsByID:      make(map[string]thingsStateProject, len(snapshot.Projects)),
		projectsByLogical: make(map[string]thingsStateProject, len(snapshot.Projects)),
		tasksByID:         make(map[string]thingsStateTask, len(snapshot.Tasks)),
		tasksByLogical:    make(map[string]thingsStateTask, len(snapshot.Tasks)),
	}
	for _, area := range snapshot.Areas {
		if area.ID != "" {
			index.areasByID[area.ID] = area
		}
		key := areaLogicalKey(area)
		if _, exists := index.areasByLogical[key]; exists {
			return stateIndex{}, fmt.Errorf("duplicate area logical key %q", key)
		}
		index.areasByLogical[key] = area
	}
	for _, project := range snapshot.Projects {
		if project.ID != "" {
			index.projectsByID[project.ID] = project
		}
		key := projectLogicalKey(project)
		if _, exists := index.projectsByLogical[key]; exists {
			return stateIndex{}, fmt.Errorf("duplicate project logical key %q", key)
		}
		index.projectsByLogical[key] = project
	}
	for _, task := range snapshot.Tasks {
		if task.ID != "" {
			index.tasksByID[task.ID] = task
		}
		key := taskLogicalKey(task)
		if key == "" {
			continue
		}
		if _, exists := index.tasksByLogical[key]; exists {
			return stateIndex{}, fmt.Errorf("duplicate task logical key %q", key)
		}
		index.tasksByLogical[key] = task
	}
	return index, nil
}

func areaLogicalKey(area thingsStateArea) string {
	return normalizeStateValue(area.Name)
}

func projectLogicalKey(project thingsStateProject) string {
	return normalizeStateValue(project.Area) + "|" + normalizeStateValue(project.Name)
}

func taskLogicalKey(task thingsStateTask) string {
	if !taskHasSupportedRestoreContainer(task) {
		return ""
	}
	parentType := "area"
	parentName := task.Area
	if strings.TrimSpace(task.Project) != "" {
		parentType = "project"
		parentName = task.Project
	}
	return parentType + "|" + normalizeStateValue(parentName) + "|" + normalizeStateValue(task.Name)
}

func taskHasSupportedRestoreContainer(task thingsStateTask) bool {
	return strings.TrimSpace(task.Project) != "" || strings.TrimSpace(task.Area) != ""
}

func normalizeStateValue(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func matchArea(target thingsStateArea, current stateIndex) (thingsStateArea, string) {
	if target.ID != "" {
		if area, ok := current.areasByID[target.ID]; ok {
			return area, "id"
		}
	}
	if area, ok := current.areasByLogical[areaLogicalKey(target)]; ok {
		return area, "logical"
	}
	return thingsStateArea{}, ""
}

func matchProject(target thingsStateProject, current stateIndex) (thingsStateProject, string) {
	if target.ID != "" {
		if project, ok := current.projectsByID[target.ID]; ok {
			return project, "id"
		}
	}
	if project, ok := current.projectsByLogical[projectLogicalKey(target)]; ok {
		return project, "logical"
	}
	return thingsStateProject{}, ""
}

func matchTask(target thingsStateTask, current stateIndex) (thingsStateTask, string) {
	if target.ID != "" {
		if task, ok := current.tasksByID[target.ID]; ok {
			return task, "id"
		}
	}
	if key := taskLogicalKey(target); key != "" {
		if task, ok := current.tasksByLogical[key]; ok {
			return task, "logical"
		}
	}
	return thingsStateTask{}, ""
}

func sameStringSet(left, right []string) bool {
	left = normalizeStringSet(left)
	right = normalizeStringSet(right)
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func normalizeStringSet(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, trimmed)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i]) < strings.ToLower(out[j])
	})
	return out
}
