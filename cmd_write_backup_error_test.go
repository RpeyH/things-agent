package main

import "testing"

func TestWriteCommandsReturnBackupErrorWhenDBIsMissing(t *testing.T) {
	fr := &fakeRunner{output: "ok"}
	setupTestRuntime(t, t.TempDir(), fr)

	cases := []struct {
		name string
		cmd  func() error
	}{
		{
			name: "add-list",
			cmd: func() error {
				c := newAddListCmd()
				c.SetArgs([]string{"--name", "area"})
				return c.Execute()
			},
		},
		{
			name: "add-project",
			cmd: func() error {
				c := newAddProjectCmd()
				c.SetArgs([]string{"--name", "proj"})
				return c.Execute()
			},
		},
		{
			name: "add-task",
			cmd: func() error {
				c := newAddTaskCmd()
				c.SetArgs([]string{"--name", "task"})
				return c.Execute()
			},
		},
		{
			name: "set-task-notes",
			cmd: func() error {
				c := newSetTaskNotesCmd()
				c.SetArgs([]string{"--name", "task", "--notes", "x"})
				return c.Execute()
			},
		},
		{
			name: "complete-task",
			cmd: func() error {
				c := newCompleteTaskCmd()
				c.SetArgs([]string{"--name", "task"})
				return c.Execute()
			},
		},
		{
			name: "add-subtask",
			cmd: func() error {
				c := newAddSubtaskCmd()
				c.SetArgs([]string{"--task", "task", "--name", "sub"})
				return c.Execute()
			},
		},
		{
			name: "url add",
			cmd: func() error {
				c := newURLAddCmd()
				c.SetArgs([]string{"--title", "x"})
				return c.Execute()
			},
		},
	}

	for _, tc := range cases {
		if err := tc.cmd(); err == nil {
			t.Fatalf("%s should fail without backupable db files", tc.name)
		}
	}
}
