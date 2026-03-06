package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestCmdBasicHelpers(t *testing.T) {
	params := map[string]string{}
	setIfNotEmpty(params, "a", "x")
	setIfNotEmpty(params, "b", "  ")
	if params["a"] != "x" {
		t.Fatalf("setIfNotEmpty should set non-empty value")
	}
	if _, ok := params["b"]; ok {
		t.Fatal("setIfNotEmpty should skip empty value")
	}

	cmd := &cobra.Command{Use: "x"}
	cmd.Flags().String("name", "", "")
	cmd.Flags().Bool("done", false, "")
	cmd.Flags().Bool("done-false", true, "")
	if err := cmd.Flags().Set("name", "alpha"); err != nil {
		t.Fatalf("set name flag failed: %v", err)
	}
	if err := cmd.Flags().Set("done", "true"); err != nil {
		t.Fatalf("set done flag failed: %v", err)
	}

	setIfChanged(cmd, params, "name", "alpha")
	setBoolIfChanged(cmd, params, "done", true)
	if err := cmd.Flags().Set("done-false", "false"); err != nil {
		t.Fatalf("set done-false flag failed: %v", err)
	}
	setBoolIfChanged(cmd, params, "done-false", false)
	if params["name"] != "alpha" || params["done"] != "true" {
		t.Fatalf("unexpected params after setIfChanged/setBoolIfChanged: %#v", params)
	}
	if params["done-false"] != "false" {
		t.Fatalf("expected done-false=false param, got %#v", params)
	}
}

func TestBasicReadCommands(t *testing.T) {
	fr := &fakeRunner{output: "ok"}
	setupTestRuntime(t, t.TempDir(), fr)

	lists := newListsCmd()
	if err := lists.Execute(); err != nil {
		t.Fatalf("lists failed: %v", err)
	}
	projects := newProjectsCmd()
	if err := projects.Execute(); err != nil {
		t.Fatalf("projects failed: %v", err)
	}
	tasks := newTasksCmd()
	tasks.SetArgs([]string{"--list", "Inbox", "--query", "x"})
	if err := tasks.Execute(); err != nil {
		t.Fatalf("tasks failed: %v", err)
	}
	search := newSearchCmd()
	search.SetArgs([]string{"--query", "x", "--list", "Inbox"})
	if err := search.Execute(); err != nil {
		t.Fatalf("search failed: %v", err)
	}
}

func TestBackupRestoreSessionCommands(t *testing.T) {
	fr := &fakeRunner{}
	tmp := setupTestRuntimeWithDB(t, fr)

	backup := newBackupCmd()
	if err := backup.Execute(); err != nil {
		t.Fatalf("backup failed: %v", err)
	}

	session := newSessionStartCmd()
	if err := session.Execute(); err != nil {
		t.Fatalf("session-start failed: %v", err)
	}

	entries, err := os.ReadDir(filepath.Join(tmp, backupDirName))
	if err != nil || len(entries) == 0 {
		t.Fatalf("expected backup files, err=%v count=%d", err, len(entries))
	}

	restore := newRestoreCmd()
	if err := restore.Execute(); err != nil {
		t.Fatalf("restore latest failed: %v", err)
	}

	restoreByFile := newRestoreCmd()
	restoreByFile.SetArgs([]string{"--file", filepath.Join(tmp, backupDirName, entries[0].Name())})
	err = restoreByFile.Execute()
	if err == nil || !strings.Contains(err.Error(), "--unsafe-legacy-restore") {
		t.Fatalf("expected unsafe legacy restore guard, got: %v", err)
	}

	restoreMissing := newRestoreCmd()
	restoreMissing.SetArgs([]string{"--timestamp", "missing-ts"})
	if err := restoreMissing.Execute(); err == nil {
		t.Fatal("expected restore error for missing timestamp/file")
	}

	restoreByTimestamp := newRestoreCmd()
	restoreByTimestamp.SetArgs([]string{"--timestamp", inferTimestamp(entries[0].Name())})
	if err := restoreByTimestamp.Execute(); err != nil {
		t.Fatalf("restore by timestamp failed: %v", err)
	}

	restoreByUnsafeFile := newRestoreCmd()
	restoreByUnsafeFile.SetArgs([]string{"--file", filepath.Join(tmp, backupDirName, entries[0].Name()), "--unsafe-legacy-restore"})
	if err := restoreByUnsafeFile.Execute(); err != nil && !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("restore by unsafe file unexpected error: %v", err)
	}
}

func TestRestoreLatestWithoutBackupReturnsError(t *testing.T) {
	fr := &fakeRunner{}
	tmp := t.TempDir()
	setupTestRuntime(t, tmp, fr)
	restore := newRestoreCmd()
	if err := restore.Execute(); err == nil {
		t.Fatal("expected restore latest error when no backups exist")
	}
}

func TestBackupCommandsFailWithoutDBFiles(t *testing.T) {
	fr := &fakeRunner{}
	setupTestRuntime(t, t.TempDir(), fr)

	backup := newBackupCmd()
	if err := backup.Execute(); err == nil {
		t.Fatal("expected backup failure without sqlite files")
	}

	session := newSessionStartCmd()
	if err := session.Execute(); err == nil {
		t.Fatal("expected session-start failure without sqlite files")
	}
}
