package main

import (
	"strings"
	"testing"
)

func TestURLCommandsExecute(t *testing.T) {
	t.Run("url add and update", func(t *testing.T) {
		fr := &fakeRunner{output: "ok"}
		setupTestRuntimeWithDB(t, fr)

		add := newURLAddCmd()
		add.SetArgs([]string{
			"--title", "task-a",
			"--notes", "n",
			"--when", "today",
			"--deadline", "",
			"--tags", "a,b",
			"--checklist-items", "one,two",
			"--list", "Inbox",
			"--reveal",
		})
		if err := add.Execute(); err != nil {
			t.Fatalf("url add failed: %v", err)
		}

		update := newURLUpdateCmd()
		update.SetArgs([]string{
			"--id", "abc",
			"--title", "task-b",
			"--append-notes", "x",
			"--append-checklist-items", "three,four",
			"--completed",
		})
		if err := update.Execute(); err != nil {
			t.Fatalf("url update failed: %v", err)
		}

		scripts := fr.allScripts()
		if len(scripts) < 2 {
			t.Fatalf("expected 2 scripts, got %d", len(scripts))
		}
		if !strings.Contains(scripts[0], "things:///add?") {
			t.Fatalf("unexpected add URL script: %s", scripts[0])
		}
		if !strings.Contains(scripts[1], "things:///update?") || !strings.Contains(scripts[1], "auth-token=token-test") {
			t.Fatalf("unexpected update URL script: %s", scripts[1])
		}
	})

	t.Run("url project commands", func(t *testing.T) {
		fr := &fakeRunner{output: "ok"}
		setupTestRuntimeWithDB(t, fr)

		addProject := newURLAddProjectCmd()
		addProject.SetArgs([]string{
			"--title", "p1",
			"--to-dos", "a,b",
			"--area", "Inbox",
			"--reveal",
		})
		if err := addProject.Execute(); err != nil {
			t.Fatalf("url add-project failed: %v", err)
		}

		updateProject := newURLUpdateProjectCmd()
		updateProject.SetArgs([]string{
			"--id", "pid",
			"--title", "p2",
			"--notes", "n",
			"--duplicate",
		})
		if err := updateProject.Execute(); err != nil {
			t.Fatalf("url update-project failed: %v", err)
		}

		scripts := fr.allScripts()
		if len(scripts) < 2 {
			t.Fatalf("expected 2 scripts, got %d", len(scripts))
		}
	})

	t.Run("url misc commands", func(t *testing.T) {
		fr := &fakeRunner{output: "ok"}
		setupTestRuntimeWithDB(t, fr)

		show := newURLShowCmd()
		show.SetArgs([]string{"--id", "today"})
		if err := show.Execute(); err != nil {
			t.Fatalf("url show failed: %v", err)
		}

		search := newURLSearchCmd()
		search.SetArgs([]string{"--query", "task"})
		if err := search.Execute(); err != nil {
			t.Fatalf("url search failed: %v", err)
		}

		version := newURLVersionCmd()
		if err := version.Execute(); err != nil {
			t.Fatalf("url version failed: %v", err)
		}

		addJSON := newURLAddJSONCmd()
		addJSON.SetArgs([]string{"--data", `{"items":[{"title":"x"}]}`, "--reveal"})
		if err := addJSON.Execute(); err != nil {
			t.Fatalf("url add-json failed: %v", err)
		}

		addJSONUpdate := newURLAddJSONCmd()
		addJSONUpdate.SetArgs([]string{"--data", `{"operation":"update","items":[]}`})
		if err := addJSONUpdate.Execute(); err != nil {
			t.Fatalf("url add-json update failed: %v", err)
		}

		scripts := strings.Join(fr.allScripts(), "\n")
		if !strings.Contains(scripts, "things:///show?") || !strings.Contains(scripts, "things:///search?") ||
			!strings.Contains(scripts, "things:///version") || !strings.Contains(scripts, "things:///add-json?") {
			t.Fatalf("unexpected URL scripts: %s", scripts)
		}
	})

	t.Run("url add-json update requires token", func(t *testing.T) {
		fr := &fakeRunner{output: "ok"}
		setupTestRuntimeWithDB(t, fr)
		config.authToken = ""
		cmd := newURLAddJSONCmd()
		cmd.SetArgs([]string{"--data", `{"operation":"update","items":[]}`})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "auth-token is required") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
