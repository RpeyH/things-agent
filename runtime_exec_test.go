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

func TestBackupIfNeededIsNoOp(t *testing.T) {
	ctx := context.Background()

	cfg := &runtimeConfig{dataDir: t.TempDir()}
	if err := backupIfNeeded(ctx, cfg); err != nil {
		t.Fatalf("backupIfNeeded should be a no-op: %v", err)
	}
}

func TestBackupIfDestructiveBranches(t *testing.T) {
	ctx := context.Background()

	cfgErr := &runtimeConfig{dataDir: t.TempDir()}
	if err := backupIfDestructive(ctx, cfgErr); err == nil {
		t.Fatal("backupIfDestructive should fail without backupable db files")
	}

	fr := &fakeRunner{}
	tmp := setupTestRuntimeWithDB(t, fr)
	cfgOK := &runtimeConfig{dataDir: tmp}
	if err := backupIfDestructive(ctx, cfgOK); err != nil {
		t.Fatalf("backupIfDestructive should succeed with db files: %v", err)
	}
}
