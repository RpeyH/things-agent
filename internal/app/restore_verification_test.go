package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerificationErrorBranches(t *testing.T) {
	t.Run("rejects incomplete snapshot", func(t *testing.T) {
		err := verificationError(restoreVerificationReport{})
		if err == nil || !strings.Contains(err.Error(), "snapshot is incomplete") {
			t.Fatalf("expected incomplete snapshot error, got %v", err)
		}
	})

	t.Run("returns first file error", func(t *testing.T) {
		err := verificationError(restoreVerificationReport{
			Complete: true,
			Files: []restoreVerifiedFile{
				{Name: "main.sqlite", Match: false, Error: "boom"},
			},
		})
		if err == nil || !strings.Contains(err.Error(), "verification failed for main.sqlite: boom") {
			t.Fatalf("expected file verification error, got %v", err)
		}
	})

	t.Run("reports mismatch without explicit file error", func(t *testing.T) {
		err := verificationError(restoreVerificationReport{
			Complete: true,
			Files: []restoreVerifiedFile{
				{Name: "main.sqlite", Match: false},
			},
		})
		if err == nil || !strings.Contains(err.Error(), "live file mismatch for main.sqlite") {
			t.Fatalf("expected live file mismatch error, got %v", err)
		}
	})
}

func TestFilesEqualBranches(t *testing.T) {
	dir := t.TempDir()
	left := filepath.Join(dir, "left")
	right := filepath.Join(dir, "right")

	if err := os.WriteFile(left, []byte("same"), 0o644); err != nil {
		t.Fatalf("write left failed: %v", err)
	}
	if err := os.WriteFile(right, []byte("same"), 0o644); err != nil {
		t.Fatalf("write right failed: %v", err)
	}

	match, err := filesEqual(left, right)
	if err != nil || !match {
		t.Fatalf("expected equal files, got match=%v err=%v", match, err)
	}

	if err := os.WriteFile(right, []byte("diff"), 0o644); err != nil {
		t.Fatalf("rewrite right failed: %v", err)
	}
	match, err = filesEqual(left, right)
	if err != nil || match {
		t.Fatalf("expected content mismatch, got match=%v err=%v", match, err)
	}

	if err := os.WriteFile(right, []byte("different-size"), 0o644); err != nil {
		t.Fatalf("rewrite right size failed: %v", err)
	}
	match, err = filesEqual(left, right)
	if err != nil || match {
		t.Fatalf("expected size mismatch, got match=%v err=%v", match, err)
	}

	match, err = filesEqual(left, filepath.Join(dir, "missing"))
	if err == nil || match {
		t.Fatalf("expected missing file error, got match=%v err=%v", match, err)
	}
}

func TestBytesEqualLengthAndContent(t *testing.T) {
	if !bytesEqual([]byte("abc"), []byte("abc")) {
		t.Fatal("expected equal byte slices")
	}
	if bytesEqual([]byte("abc"), []byte("ab")) {
		t.Fatal("expected different lengths to be unequal")
	}
	if bytesEqual([]byte("abc"), []byte("axc")) {
		t.Fatal("expected different content to be unequal")
	}
}

func TestBuildSnapshotVerificationRecordsFirstError(t *testing.T) {
	dataDir := t.TempDir()
	snapshot := filepath.Join(dataDir, "main.sqlite.2026-03-08:09-10-11.bak")
	if err := os.WriteFile(snapshot, []byte("snapshot"), 0o644); err != nil {
		t.Fatalf("write snapshot failed: %v", err)
	}

	report, err := buildSnapshotVerification(dataDir, []string{snapshot})
	if err == nil || !strings.Contains(err.Error(), "compare") {
		t.Fatalf("expected compare error, got report=%#v err=%v", report, err)
	}
	if report.Match {
		t.Fatalf("expected mismatch report when live file is missing, got %#v", report)
	}
	if len(report.Files) != 1 || report.Files[0].Error == "" {
		t.Fatalf("expected per-file error details, got %#v", report.Files)
	}
}
