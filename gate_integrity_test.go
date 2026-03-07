package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"testing"
	"time"
)

var errNoTestsMatched = errors.New("go test selection matched zero tests")

type goTestEvent struct {
	Action  string `json:"Action"`
	Package string `json:"Package"`
	Test    string `json:"Test"`
	Output  string `json:"Output"`
}

func TestGateIntegrity(t *testing.T) {
	t.Run("fails when selection matches zero tests", func(t *testing.T) {
		err := runGoTestGate(context.Background(), ".", "^TestGateIntegrityFixtureMissing$")
		if !errors.Is(err, errNoTestsMatched) {
			t.Fatalf("expected zero-match gate error, got %v", err)
		}
	})

	t.Run("passes when selection executes at least one test", func(t *testing.T) {
		if err := runGoTestGate(context.Background(), ".", "^TestGateIntegrityFixturePass$"); err != nil {
			t.Fatalf("expected gate to pass for matching test selection: %v", err)
		}
	})
}

func TestCountGoTestRuns(t *testing.T) {
	stream := strings.Join([]string{
		`{"Action":"start","Package":"github.com/alnah/things-agent"}`,
		`{"Action":"run","Package":"github.com/alnah/things-agent","Test":"TestOne"}`,
		`{"Action":"output","Package":"github.com/alnah/things-agent","Test":"TestOne","Output":"=== RUN   TestOne\n"}`,
		`{"Action":"pass","Package":"github.com/alnah/things-agent","Test":"TestOne"}`,
		`{"Action":"run","Package":"github.com/alnah/things-agent","Test":"TestTwo"}`,
		`{"Action":"pass","Package":"github.com/alnah/things-agent","Test":"TestTwo"}`,
	}, "\n")

	runs, err := countGoTestRuns(strings.NewReader(stream))
	if err != nil {
		t.Fatalf("countGoTestRuns returned error: %v", err)
	}
	if runs != 2 {
		t.Fatalf("expected 2 test run events, got %d", runs)
	}
}

func TestCountGoTestRunsRejectsInvalidJSON(t *testing.T) {
	_, err := countGoTestRuns(strings.NewReader("{not-json}\n"))
	if err == nil {
		t.Fatal("expected invalid json stream to fail")
	}
}

func TestGateIntegrityFixturePass(t *testing.T) {}

func runGoTestGate(parent context.Context, pkg, runPattern string) error {
	ctx, cancel := context.WithTimeout(parent, 20*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "test", "-json", pkg, "-run", runPattern, "-count=1")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go test gate failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	runCount, err := countGoTestRuns(&stdout)
	if err != nil {
		return fmt.Errorf("parse go test -json output: %w", err)
	}
	if runCount == 0 {
		return errNoTestsMatched
	}
	return nil
}

func countGoTestRuns(stream io.Reader) (int, error) {
	scanner := bufio.NewScanner(stream)
	runCount := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var event goTestEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return 0, err
		}
		if event.Action == "run" && event.Test != "" {
			runCount++
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	return runCount, nil
}
