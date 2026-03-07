package main

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"
)

func newListSubtasksCmd() *cobra.Command {
	var taskName, taskID string
	cmd := &cobra.Command{
		Use:   "list-checklist-items",
		Short: "List checklist items for a task",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			taskName, taskID, err = resolveTaskParentSelector(taskName, taskID)
			if err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptListSubtasks(cfg.bundleID, taskName, taskID))
		},
	}
	cmd.Flags().StringVar(&taskName, "task", "", "Task name parent")
	cmd.Flags().StringVar(&taskID, "task-id", "", "Task ID parent")
	return cmd
}

func newAddSubtaskCmd() *cobra.Command {
	var taskName, taskID, subtaskName string
	cmd := &cobra.Command{
		Use:   "add-checklist-item",
		Short: "Add a native checklist item to a task",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			taskName, taskID, err = resolveTaskParentSelector(taskName, taskID)
			if err != nil {
				return err
			}
			subtaskName = strings.TrimSpace(subtaskName)
			if subtaskName == "" {
				return errors.New("--name is required")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			token, err := requireAuthToken(cfg)
			if err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptAppendChecklistByRef(cfg.bundleID, taskName, taskID, []string{subtaskName}, token))
		},
	}
	cmd.Flags().StringVar(&taskName, "task", "", "Task name parent")
	cmd.Flags().StringVar(&taskID, "task-id", "", "Task ID parent")
	cmd.Flags().StringVar(&subtaskName, "name", "", "Checklist item name")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newEditSubtaskCmd() *cobra.Command {
	var taskName, taskID, subtaskName, newName, notes string
	var subtaskIndex int
	cmd := &cobra.Command{
		Use:   "edit-checklist-item",
		Short: "Edit a checklist item",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			taskName, taskID, err = resolveTaskParentSelector(taskName, taskID)
			if err != nil {
				return err
			}
			subtaskName = strings.TrimSpace(subtaskName)
			newName = strings.TrimSpace(newName)
			notes = strings.TrimSpace(notes)
			if subtaskIndex <= 0 && subtaskName == "" {
				return errors.New("provide --index (>=1) or --name")
			}
			if newName == "" && notes == "" {
				return errors.New("provide --new-name and/or --notes")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptEditSubtask(cfg.bundleID, taskName, taskID, subtaskName, subtaskIndex, newName, notes))
		},
	}
	cmd.Flags().StringVar(&taskName, "task", "", "Task name parent")
	cmd.Flags().StringVar(&taskID, "task-id", "", "Task ID parent")
	cmd.Flags().StringVar(&subtaskName, "name", "", "Target checklist item name")
	cmd.Flags().IntVar(&subtaskIndex, "index", 0, "Target checklist item index (1-based)")
	cmd.Flags().StringVar(&newName, "new-name", "", "New name")
	cmd.Flags().StringVar(&notes, "notes", "", "New notes")
	return cmd
}

func newDeleteSubtaskCmd() *cobra.Command {
	var taskName, taskID, subtaskName string
	var subtaskIndex int
	cmd := &cobra.Command{
		Use:   "delete-checklist-item",
		Short: "Delete a checklist item",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			taskName, taskID, err = resolveTaskParentSelector(taskName, taskID)
			if err != nil {
				return err
			}
			subtaskName = strings.TrimSpace(subtaskName)
			if subtaskIndex <= 0 && subtaskName == "" {
				return errors.New("provide --index (>=1) or --name")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptDeleteSubtask(cfg.bundleID, taskName, taskID, subtaskName, subtaskIndex))
		},
	}
	cmd.Flags().StringVar(&taskName, "task", "", "Task name parent")
	cmd.Flags().StringVar(&taskID, "task-id", "", "Task ID parent")
	cmd.Flags().StringVar(&subtaskName, "name", "", "Checklist item name")
	cmd.Flags().IntVar(&subtaskIndex, "index", 0, "Checklist item index (1-based)")
	return cmd
}

func newCompleteSubtaskCmd() *cobra.Command {
	var taskName, taskID, subtaskName string
	var subtaskIndex int
	cmd := &cobra.Command{
		Use:   "complete-checklist-item",
		Short: "Mark checklist item as completed",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			taskName, taskID, err = resolveTaskParentSelector(taskName, taskID)
			if err != nil {
				return err
			}
			subtaskName = strings.TrimSpace(subtaskName)
			if subtaskIndex <= 0 && subtaskName == "" {
				return errors.New("provide --index (>=1) or --name")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptSetSubtaskStatus(cfg.bundleID, taskName, taskID, subtaskName, subtaskIndex, true))
		},
	}
	cmd.Flags().StringVar(&taskName, "task", "", "Task name parent")
	cmd.Flags().StringVar(&taskID, "task-id", "", "Task ID parent")
	cmd.Flags().StringVar(&subtaskName, "name", "", "Checklist item name")
	cmd.Flags().IntVar(&subtaskIndex, "index", 0, "Checklist item index (1-based)")
	return cmd
}

func newUncompleteSubtaskCmd() *cobra.Command {
	var taskName, taskID, subtaskName string
	var subtaskIndex int
	cmd := &cobra.Command{
		Use:   "uncomplete-checklist-item",
		Short: "Mark checklist item as uncompleted",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			taskName, taskID, err = resolveTaskParentSelector(taskName, taskID)
			if err != nil {
				return err
			}
			subtaskName = strings.TrimSpace(subtaskName)
			if subtaskIndex <= 0 && subtaskName == "" {
				return errors.New("provide --index (>=1) or --name")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptSetSubtaskStatus(cfg.bundleID, taskName, taskID, subtaskName, subtaskIndex, false))
		},
	}
	cmd.Flags().StringVar(&taskName, "task", "", "Task name parent")
	cmd.Flags().StringVar(&taskID, "task-id", "", "Task ID parent")
	cmd.Flags().StringVar(&subtaskName, "name", "", "Checklist item name")
	cmd.Flags().IntVar(&subtaskIndex, "index", 0, "Checklist item index (1-based)")
	return cmd
}
