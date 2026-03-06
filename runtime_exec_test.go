package main

import (
	"context"
	"errors"
	"testing"
)

func TestRunResultBranches(t *testing.T) {
	cfg := &runtimeConfig{
		runner: &fakeRunner{output: ""},
	}
	if err := runResult(context.Background(), cfg, "script"); err != nil {
		t.Fatalf("runResult with empty output should succeed: %v", err)
	}

	cfgErr := &runtimeConfig{
		runner: &fakeRunner{err: errors.New("boom")},
	}
	if err := runResult(context.Background(), cfgErr, "script"); err == nil {
		t.Fatal("runResult should return runner error")
	}
}

func TestBackupIfNeededBranches(t *testing.T) {
	ctx := context.Background()

	cfgErr := &runtimeConfig{dataDir: t.TempDir()}
	if err := backupIfNeeded(ctx, cfgErr); err == nil {
		t.Fatal("backupIfNeeded should fail without backupable db files")
	}

	fr := &fakeRunner{}
	tmp := setupTestRuntimeWithDB(t, fr)
	cfgOK := &runtimeConfig{dataDir: tmp}
	if err := backupIfNeeded(ctx, cfgOK); err != nil {
		t.Fatalf("backupIfNeeded should succeed with db files: %v", err)
	}
}
