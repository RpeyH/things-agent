package main

import (
	"testing"
)

func TestRootCommandBuildsAndRunsVersion(t *testing.T) {
	root := newRootCmd()
	if root == nil {
		t.Fatal("expected root command")
	}
	root.SetArgs([]string{"version"})
	if err := root.Execute(); err != nil {
		t.Fatalf("version execute failed: %v", err)
	}
}

func TestRootHelp(t *testing.T) {
	root := newRootCmd()
	root.SetArgs([]string{"--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("help execute failed: %v", err)
	}
}
