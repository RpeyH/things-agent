package main

import (
	"os"
	"strings"
	"testing"
)

func TestDocsSyncGate(t *testing.T) {
	t.Helper()

	agents := mustReadDocFile(t, "AGENTS.md")
	readme := mustReadDocFile(t, "README.md")

	required := []string{
		"things-agent url json",
		"things-agent restore list",
		"things-agent restore verify",
		"add-task --area",
		"add-task --project",
		"edit-task (--name <name> | --id <id>)",
		"list-subtasks (--task <name> | --task-id <id>)",
	}
	for _, needle := range required {
		if !strings.Contains(agents, needle) && !strings.Contains(readme, needle) {
			t.Fatalf("docs sync gate missing required command surface %q", needle)
		}
	}

	agentsRequired := []string{
		"show-task (--name <name> | --id <id>)",
		"add-subtask (--task <name> | --task-id <id>) --name <name>",
	}
	for _, needle := range agentsRequired {
		if !strings.Contains(agents, needle) {
			t.Fatalf("AGENTS.md missing %q", needle)
		}
	}

	readmeRequired := []string{
		"things-agent show-task --id",
		"things-agent complete-task --id",
		"things-agent add-subtask --task-id",
	}
	for _, needle := range readmeRequired {
		if !strings.Contains(readme, needle) {
			t.Fatalf("README.md missing %q", needle)
		}
	}

	forbidden := []string{
		"things-agent url add-json",
		"restore --file",
		"add-task --name \"Say hello\" --notes \"Message\" --list",
		"add-project --name <name> [--list <area>]",
	}
	for _, needle := range forbidden {
		if strings.Contains(agents, needle) || strings.Contains(readme, needle) {
			t.Fatalf("docs sync gate found forbidden legacy surface %q", needle)
		}
	}
}

func mustReadDocFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}
