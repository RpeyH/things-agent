package main

import (
	"strings"
	"testing"
)

func TestScriptTasksBranches(t *testing.T) {
	all := scriptTasks("bundle.id", "", "")
	if !strings.Contains(all, `return name of (every «class tstk»)`) {
		t.Fatalf("unexpected all-tasks script: %s", all)
	}

	byQuery := scriptTasks("bundle.id", "", "alpha")
	if !strings.Contains(byQuery, `name contains q or notes contains q`) {
		t.Fatalf("unexpected query-only script: %s", byQuery)
	}

	byList := scriptTasks("bundle.id", "Inbox", "")
	if !strings.Contains(byList, `set l to first list whose name is "Inbox"`) {
		t.Fatalf("unexpected list-only script: %s", byList)
	}

	byListQuery := scriptTasks("bundle.id", "Inbox", "beta")
	if !strings.Contains(byListQuery, `of l whose (name contains q or notes contains q)`) {
		t.Fatalf("unexpected list+query script: %s", byListQuery)
	}
}

func TestScriptSearchAliasesTasks(t *testing.T) {
	got := scriptSearch("bundle.id", "Inbox", "x")
	want := scriptTasks("bundle.id", "Inbox", "x")
	if got != want {
		t.Fatalf("scriptSearch must proxy scriptTasks")
	}
}

func TestScriptResolveTaskByNameEscapesInput(t *testing.T) {
	got := scriptResolveTaskByName(`foo "bar"`)
	if !strings.Contains(got, `\"bar\"`) {
		t.Fatalf("expected escaped task name, got: %s", got)
	}
	if !strings.Contains(got, "Ambiguous item name; use a unique name.") {
		t.Fatalf("expected ambiguity guard, got: %s", got)
	}
	if !strings.Contains(got, "set totalCount to projectCount + taskCount") {
		t.Fatalf("expected combined match count, got: %s", got)
	}
}
