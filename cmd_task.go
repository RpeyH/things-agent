package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newShowTaskCmd() *cobra.Command {
	var name string
	var withSubtasks bool
	cmd := &cobra.Command{
		Use:   "show-task",
		Short: "Show full details for a task or project",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			name = strings.TrimSpace(name)
			if name == "" {
				return errors.New("--name is required")
			}
			return runResult(ctx, cfg, scriptShowTask(cfg.bundleID, name, withSubtasks))
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task or project name")
	cmd.Flags().BoolVar(&withSubtasks, "with-subtasks", true, "Include subtasks")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newAddTaskCmd() *cobra.Command {
	var name, notes, tags, listName, due, subtasks string
	cmd := &cobra.Command{
		Use:   "add-task",
		Short: "Add a task",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(name) == "" {
				return errors.New("--name is required")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			dueDate, err := parseToAppleDate(due)
			if err != nil {
				return err
			}
			subtasksList := parseCSVList(subtasks)
			out, err := cfg.runner.run(ctx, scriptAddTask(cfg.bundleID, strings.TrimSpace(listName), name, notes, tags, dueDate))
			if err != nil {
				return err
			}
			taskID := strings.TrimSpace(out)
			if taskID == "" {
				return errors.New("could not retrieve created task id")
			}
			if len(subtasksList) > 0 {
				token, err := requireAuthToken(cfg)
				if err != nil {
					return err
				}
				if _, err := cfg.runner.run(ctx, scriptSetChecklistByID(cfg.bundleID, taskID, subtasksList, token)); err != nil {
					return err
				}
			}
			fmt.Println(taskID)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task name")
	cmd.Flags().StringVar(&notes, "notes", "", "Notes")
	cmd.Flags().StringVar(&tags, "tags", "", "Tags (comma-separated)")
	cmd.Flags().StringVar(&listName, "list", envOrDefault("THINGS_DEFAULT_LIST", defaultListName), "Destination area")
	cmd.Flags().StringVar(&due, "due", "", "Due date (YYYY-MM-DD [HH:mm[:ss]])")
	cmd.Flags().StringVar(&subtasks, "subtasks", "", "Subtasks (name1, name2, ...)")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newEditTaskCmd() *cobra.Command {
	var sourceName, newName, notes, tags, moveTo, due, completion, creation, cancel string
	cmd := &cobra.Command{
		Use:   "edit-task",
		Short: "Edit a task (by name)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(sourceName) == "" {
				return errors.New("--name is required")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}

			dueDate, err := parseToAppleDate(due)
			if err != nil {
				return err
			}
			completionDate, err := parseToAppleDate(completion)
			if err != nil {
				return err
			}
			creationDate, err := parseToAppleDate(creation)
			if err != nil {
				return err
			}
			cancelDate, err := parseToAppleDate(cancel)
			if err != nil {
				return err
			}

			script, err := scriptEditTask(
				cfg.bundleID,
				sourceName,
				newName,
				notes,
				tags,
				moveTo,
				dueDate,
				completionDate,
				creationDate,
				cancelDate,
			)
			if err != nil {
				return err
			}
			return runResult(ctx, cfg, script)
		},
	}
	cmd.Flags().StringVar(&sourceName, "name", "", "Task name to edit")
	cmd.Flags().StringVar(&newName, "new-name", "", "New name")
	cmd.Flags().StringVar(&notes, "notes", "", "New notes")
	cmd.Flags().StringVar(&tags, "tags", "", "Tags")
	cmd.Flags().StringVar(&moveTo, "move-to", "", "New area")
	cmd.Flags().StringVar(&due, "due", "", "New due date")
	cmd.Flags().StringVar(&completion, "completion", "", "Completion date")
	cmd.Flags().StringVar(&creation, "creation", "", "Creation date")
	cmd.Flags().StringVar(&cancel, "cancel", "", "Cancellation date")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newDeleteTaskCmd() *cobra.Command {
	return newDeleteCmd("task", "delete-task")
}

func newCompleteTaskCmd() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "complete-task",
		Short: "Mark task as completed",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(name) == "" {
				return errors.New("--name is required")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptCompleteTask(cfg.bundleID, name, true))
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task name")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newUncompleteTaskCmd() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "uncomplete-task",
		Short: "Mark task as uncompleted",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(name) == "" {
				return errors.New("--name is required")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptCompleteTask(cfg.bundleID, name, false))
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task name")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newSetTagsCmd() *cobra.Command {
	var name, tags string
	cmd := &cobra.Command{
		Use:   "set-tags",
		Short: "Set tags on a task or project",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(name) == "" || strings.TrimSpace(tags) == "" {
				return errors.New("--name and --tags are required")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			script := fmt.Sprintf(`tell application id "%s"
%s  set tag names of t to "%s"
  return id of t
end tell`, cfg.bundleID, scriptResolveTaskByName(name), escapeApple(tags))
			return runResult(ctx, cfg, script)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task name")
	cmd.Flags().StringVar(&tags, "tags", "", "Comma-separated tagss")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("tags")
	return cmd
}

func newSetTaskTagsCmd() *cobra.Command {
	var name, tags string
	cmd := &cobra.Command{
		Use:   "set-task-tags",
		Short: "Set task tags exactly",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(name) == "" || strings.TrimSpace(tags) == "" {
				return errors.New("--name and --tags are required")
			}
			tagList := parseCSVList(tags)
			if len(tagList) == 0 {
				return errors.New("specify at least one tag in --tags")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptSetTaskTags(cfg.bundleID, name, tagList))
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task name")
	cmd.Flags().StringVar(&tags, "tags", "", "Comma-separated tagss")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("tags")
	return cmd
}

func newAddTaskTagsCmd() *cobra.Command {
	var name, tags string
	cmd := &cobra.Command{
		Use:   "add-task-tags",
		Short: "Add tags to a task (merge with existing tags)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(name) == "" || strings.TrimSpace(tags) == "" {
				return errors.New("--name and --tags are required")
			}
			tagList := parseCSVList(tags)
			if len(tagList) == 0 {
				return errors.New("specify at least one tag in --tags")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptAddTaskTags(cfg.bundleID, name, tagList))
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task name")
	cmd.Flags().StringVar(&tags, "tags", "", "Comma-separated tagss")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("tags")
	return cmd
}

func newRemoveTaskTagsCmd() *cobra.Command {
	var name, tags string
	cmd := &cobra.Command{
		Use:   "remove-task-tags",
		Short: "Remove tags from a task",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(name) == "" || strings.TrimSpace(tags) == "" {
				return errors.New("--name and --tags are required")
			}
			tagList := parseCSVList(tags)
			if len(tagList) == 0 {
				return errors.New("specify at least one tag in --tags")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptRemoveTaskTags(cfg.bundleID, name, tagList))
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task name")
	cmd.Flags().StringVar(&tags, "tags", "", "Comma-separated tagss")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("tags")
	return cmd
}

func newSetTaskNotesCmd() *cobra.Command {
	var name, notes string
	cmd := &cobra.Command{
		Use:   "set-task-notes",
		Short: "Set task notes",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(name) == "" {
				return errors.New("--name is required")
			}
			if strings.TrimSpace(notes) == "" {
				return errors.New("--notes is required")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptSetTaskNotes(cfg.bundleID, name, notes))
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task name")
	cmd.Flags().StringVar(&notes, "notes", "", "New notes")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("notes")
	return cmd
}

func newAppendTaskNotesCmd() *cobra.Command {
	var name, notes, separator string
	cmd := &cobra.Command{
		Use:   "append-task-notes",
		Short: "Append notes to task notes",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(name) == "" {
				return errors.New("--name is required")
			}
			if strings.TrimSpace(notes) == "" {
				return errors.New("--notes is required")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptAppendTaskNotes(cfg.bundleID, name, notes, separator))
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task name")
	cmd.Flags().StringVar(&notes, "notes", "", "Text to append to notes")
	cmd.Flags().StringVar(&separator, "separator", "\n", "Append separator (default: newline)")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("notes")
	return cmd
}

func newSetTaskDateCmd() *cobra.Command {
	var name, due, deadline string
	var clear bool
	cmd := &cobra.Command{
		Use:   "set-task-date",
		Short: "Set/update task due date",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(name) == "" {
				return errors.New("--name is required")
			}
			dueDate, err := parseToAppleDate(due)
			if err != nil {
				return err
			}
			deadlineDate, err := parseToAppleDate(deadline)
			if err != nil {
				return err
			}
			if !clear && dueDate == "" && deadlineDate == "" {
				return errors.New("provide --due, --deadline, or --clear")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			if clear && dueDate == "" && deadlineDate == "" {
				token, err := requireAuthToken(cfg)
				if err != nil {
					return err
				}
				return runResult(ctx, cfg, scriptClearTaskDeadlineByName(cfg.bundleID, name, token))
			}
			return runResult(ctx, cfg, scriptSetTaskDate(cfg.bundleID, name, dueDate, deadlineDate, clear))
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task name")
	cmd.Flags().StringVar(&due, "due", "", "New due date (YYYY-MM-DD [HH:mm[:ss]])")
	cmd.Flags().StringVar(&deadline, "deadline", "", "Due date alias (same format)")
	cmd.Flags().BoolVar(&clear, "clear", false, "Clear due date")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}
