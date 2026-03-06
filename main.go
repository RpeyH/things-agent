package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

const (
	defaultBundleID   = "com.culturedcode.ThingsMac"
	backupDirName     = "backups"
	backupTSFormat    = "2006-01-02:15-04-05"
	maxBackupsToKeep  = 50
	defaultListName   = "Inbox"
	cliVersion        = "0.3.0"
	thingsDataPattern = "Library/Group Containers/*.com.culturedcode.ThingsMac/ThingsData-*/Things Database.thingsdatabase"
)

var config = struct {
	bundleID  string
	dataDir   string
	authToken string
}{
	bundleID: envOrDefault("THINGS_BUNDLE_ID", defaultBundleID),
}

type runtimeConfig struct {
	bundleID  string
	dataDir   string
	authToken string
	runner    *runner
}

func main() {
	cmd := newRootCmd()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "things-agent",
		SilenceErrors: false,
		SilenceUsage:  true,
		Short:         "Things CLI via AppleScript (no direct DB access)",
		Long: `This CLI controls Things through AppleScript only.
It creates a timestamped backup in YYYY-MM-DD:hh-mm-ss format
before each write action.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	root.PersistentFlags().StringVar(&config.bundleID, "bundle-id", envOrDefault("THINGS_BUNDLE_ID", defaultBundleID), "Things app bundle id")
	root.PersistentFlags().StringVar(&config.dataDir, "data-dir", envOrDefault("THINGS_DATA_DIR", ""), "Things database path")
	root.PersistentFlags().StringVar(&config.authToken, "auth-token", envOrDefault("THINGS_AUTH_TOKEN", ""), "Things URL Scheme auth token (Settings > General)")

	root.AddCommand(
		newBackupCmd(),
		newRestoreCmd(),
		newSessionStartCmd(),
		newURLCmd(),
		newListsCmd(),
		newProjectsCmd(),
		newTasksCmd(),
		newSearchCmd(),
		newShowTaskCmd(),
		newAddTaskCmd(),
		newAddProjectCmd(),
		newAddListCmd(),
		newEditTaskCmd(),
		newEditProjectCmd(),
		newEditListCmd(),
		newDeleteTaskCmd(),
		newDeleteProjectCmd(),
		newDeleteListCmd(),
		newCompleteTaskCmd(),
		newUncompleteTaskCmd(),
		newSetTagsCmd(),
		newSetTaskTagsCmd(),
		newAddTaskTagsCmd(),
		newRemoveTaskTagsCmd(),
		newSetTaskNotesCmd(),
		newAppendTaskNotesCmd(),
		newSetTaskDateCmd(),
		newListSubtasksCmd(),
		newAddSubtaskCmd(),
		newEditSubtaskCmd(),
		newDeleteSubtaskCmd(),
		newCompleteSubtaskCmd(),
		newUncompleteSubtaskCmd(),
		&cobra.Command{
			Use:   "version",
			Short: "Show version",
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Println("things", cliVersion)
			},
		},
	)

	return root
}

func resolveRuntimeConfig(ctx context.Context) (*runtimeConfig, error) {
	dataDir := strings.TrimSpace(config.dataDir)
	if dataDir == "" {
		var err error
		dataDir, err = resolveDataDir()
		if err != nil {
			return nil, err
		}
	}

	r := newRunner(config.bundleID)
	if err := r.ensureReachable(ctx); err != nil {
		return nil, err
	}

	return &runtimeConfig{
		bundleID:  config.bundleID,
		dataDir:   dataDir,
		authToken: strings.TrimSpace(config.authToken),
		runner:    r,
	}, nil
}

func backupIfNeeded(ctx context.Context, cfg *runtimeConfig) error {
	bm := newBackupManager(cfg.dataDir)
	paths, err := bm.Create(ctx)
	if err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}
	_ = paths
	return nil
}

func runResult(ctx context.Context, cfg *runtimeConfig, script string) error {
	out, err := cfg.runner.run(ctx, script)
	if err != nil {
		return err
	}
	out = strings.TrimSpace(out)
	if out != "" {
		fmt.Println(out)
	}
	return nil
}

func runThingsURL(ctx context.Context, cfg *runtimeConfig, command string, params map[string]string) error {
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	thingsURL := "things:///" + command
	if encoded := values.Encode(); encoded != "" {
		thingsURL += "?" + encoded
	}
	return runResult(ctx, cfg, scriptOpenURL(thingsURL))
}

func scriptOpenURL(rawURL string) string {
	return fmt.Sprintf(`open location "%s"
return "ok"`, escapeApple(rawURL))
}

func setIfNotEmpty(params map[string]string, key, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	params[key] = value
}

func setIfChanged(cmd *cobra.Command, params map[string]string, key, value string) {
	if !cmd.Flags().Changed(key) {
		return
	}
	params[key] = strings.TrimSpace(value)
}

func setBoolIfChanged(cmd *cobra.Command, params map[string]string, key string, value bool) {
	if !cmd.Flags().Changed(key) {
		return
	}
	if value {
		params[key] = "true"
		return
	}
	params[key] = "false"
}

func normalizeChecklistInput(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.Contains(raw, "\n") {
		return raw
	}
	items := parseCSVList(raw)
	if len(items) == 0 {
		return raw
	}
	return strings.Join(items, "\n")
}

func newBackupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "backup",
		Short: "Create a Things DB backup",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			paths, err := newBackupManager(cfg.dataDir).Create(ctx)
			if err != nil {
				return err
			}
			for _, p := range paths {
				fmt.Println(p)
			}
			return nil
		},
	}
}

func newRestoreCmd() *cobra.Command {
	var target string
	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Restore a backup (latest by default)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			bm := newBackupManager(cfg.dataDir)
			if strings.TrimSpace(target) == "" {
				ts, err := bm.Latest(ctx)
				if err != nil {
					return err
				}
				restored, err := bm.Restore(ctx, ts)
				if err != nil {
					return err
				}
				for _, p := range restored {
					fmt.Println(p)
				}
				return nil
			}

			if info, err := os.Stat(target); err == nil && !info.IsDir() {
				if err := bm.RestoreFile(ctx, target); err != nil {
					return err
				}
				fmt.Println(target)
				return nil
			}

			ts := inferTimestamp(target)
			if ts == "" {
				ts = target
			}
			restored, err := bm.Restore(ctx, ts)
			if err != nil {
				return err
			}
			for _, p := range restored {
				fmt.Println(p)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&target, "file", "", "Chemin du fichier backup (optionnel)")
	return cmd
}

func newSessionStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "session-start",
		Short: "Initialiser la session (backup + purge des anciens backups)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			paths, err := newBackupManager(cfg.dataDir).Create(ctx)
			if err != nil {
				return err
			}
			for _, p := range paths {
				fmt.Println(p)
			}
			return nil
		},
	}
}

func newListsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "lists",
		Short: "Lister les domaines",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptAllLists(cfg.bundleID))
		},
	}
}

func newProjectsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "projects",
		Short: "List projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptAllProjects(cfg.bundleID))
		},
	}
}

func newTasksCmd() *cobra.Command {
	var listName, query string
	cmd := &cobra.Command{
		Use:   "tasks",
		Short: "List tasks (optionally filtered)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptTasks(cfg.bundleID, listName, query))
		},
	}
	cmd.Flags().StringVar(&listName, "list", "", "Domaine")
	cmd.Flags().StringVar(&query, "query", "", "Filter by name / notes")
	return cmd
}

func newSearchCmd() *cobra.Command {
	var listName, query string
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(query) == "" {
				return errors.New("--query is required")
			}
			return runResult(ctx, cfg, scriptSearch(cfg.bundleID, listName, query))
		},
	}
	cmd.Flags().StringVar(&query, "query", "", "Search text")
	cmd.Flags().StringVar(&listName, "list", "", "Limit to area")
	_ = cmd.MarkFlagRequired("query")
	return cmd
}

func newURLCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "url",
		Short: "Things URL Scheme commands (official API)",
	}
	cmd.AddCommand(
		newURLAddCmd(),
		newURLUpdateCmd(),
		newURLAddProjectCmd(),
		newURLUpdateProjectCmd(),
		newURLShowCmd(),
		newURLSearchCmd(),
		newURLVersionCmd(),
		newURLAddJSONCmd(),
	)
	return cmd
}

func newURLAddCmd() *cobra.Command {
	var (
		title, notes, when, deadline, tags, checklistItems, listName, listID, heading, headingID, notesTemplate string
		completed, canceled, reveal                                                                             bool
	)
	cmd := &cobra.Command{
		Use:   "add",
		Short: "things:///add",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			params := map[string]string{}
			setIfNotEmpty(params, "title", title)
			setIfNotEmpty(params, "notes", notes)
			setIfNotEmpty(params, "when", when)
			setIfChanged(cmd, params, "deadline", deadline)
			setIfNotEmpty(params, "tags", tags)
			setIfNotEmpty(params, "checklist-items", normalizeChecklistInput(checklistItems))
			setIfNotEmpty(params, "list", listName)
			setIfNotEmpty(params, "list-id", listID)
			setIfNotEmpty(params, "heading", heading)
			setIfNotEmpty(params, "heading-id", headingID)
			setIfNotEmpty(params, "notes-template", notesTemplate)
			setBoolIfChanged(cmd, params, "completed", completed)
			setBoolIfChanged(cmd, params, "canceled", canceled)
			setBoolIfChanged(cmd, params, "reveal", reveal)
			return runThingsURL(ctx, cfg, "add", params)
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "Title")
	cmd.Flags().StringVar(&notes, "notes", "", "Notes")
	cmd.Flags().StringVar(&when, "when", "", "When field (today, tomorrow, evening, someday, etc.)")
	cmd.Flags().StringVar(&deadline, "deadline", "", "Deadline (vide pour effacer)")
	cmd.Flags().StringVar(&tags, "tags", "", "Comma-separated tags")
	cmd.Flags().StringVar(&checklistItems, "checklist-items", "", "Checklist (lignes ou CSV)")
	cmd.Flags().StringVar(&listName, "list", "", "Destination project/area name")
	cmd.Flags().StringVar(&listID, "list-id", "", "Destination project/area ID")
	cmd.Flags().StringVar(&heading, "heading", "", "Destination heading name")
	cmd.Flags().StringVar(&headingID, "heading-id", "", "ID du heading destination")
	cmd.Flags().StringVar(&notesTemplate, "notes-template", "", "replace-title|replace-notes|replace-checklist-items")
	cmd.Flags().BoolVar(&completed, "completed", false, "Create as completed")
	cmd.Flags().BoolVar(&canceled, "canceled", false, "Create as canceled")
	cmd.Flags().BoolVar(&reveal, "reveal", false, "Reveal after creation")
	return cmd
}

func newURLUpdateCmd() *cobra.Command {
	var (
		id, title, notes, prependNotes, appendNotes, when, deadline, tags, addTags, checklistItems, prependChecklist, appendChecklist string
		listName, listID, heading, headingID                                                                                          string
		completed, canceled, reveal, duplicate                                                                                        bool
		creationDate, completionDate                                                                                                  string
	)
	cmd := &cobra.Command{
		Use:   "update",
		Short: "things:///update",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(id) == "" {
				return errors.New("--id is required")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			token, err := requireAuthToken(cfg)
			if err != nil {
				return err
			}
			params := map[string]string{
				"auth-token": token,
				"id":         id,
			}
			setIfNotEmpty(params, "title", title)
			setIfChanged(cmd, params, "notes", notes)
			setIfChanged(cmd, params, "prepend-notes", prependNotes)
			setIfChanged(cmd, params, "append-notes", appendNotes)
			setIfChanged(cmd, params, "when", when)
			setIfChanged(cmd, params, "deadline", deadline)
			setIfChanged(cmd, params, "tags", tags)
			setIfChanged(cmd, params, "add-tags", addTags)
			setIfChanged(cmd, params, "checklist-items", normalizeChecklistInput(checklistItems))
			setIfChanged(cmd, params, "prepend-checklist-items", normalizeChecklistInput(prependChecklist))
			setIfChanged(cmd, params, "append-checklist-items", normalizeChecklistInput(appendChecklist))
			setIfChanged(cmd, params, "list", listName)
			setIfChanged(cmd, params, "list-id", listID)
			setIfChanged(cmd, params, "heading", heading)
			setIfChanged(cmd, params, "heading-id", headingID)
			setBoolIfChanged(cmd, params, "completed", completed)
			setBoolIfChanged(cmd, params, "canceled", canceled)
			setBoolIfChanged(cmd, params, "reveal", reveal)
			setBoolIfChanged(cmd, params, "duplicate", duplicate)
			setIfChanged(cmd, params, "creation-date", creationDate)
			setIfChanged(cmd, params, "completion-date", completionDate)
			return runThingsURL(ctx, cfg, "update", params)
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "ID of the to-do to update")
	cmd.Flags().StringVar(&title, "title", "", "New title")
	cmd.Flags().StringVar(&notes, "notes", "", "New notes (empty to clear)")
	cmd.Flags().StringVar(&prependNotes, "prepend-notes", "", "Prepend notes")
	cmd.Flags().StringVar(&appendNotes, "append-notes", "", "Append notes")
	cmd.Flags().StringVar(&when, "when", "", "When")
	cmd.Flags().StringVar(&deadline, "deadline", "", "Deadline (vide pour effacer)")
	cmd.Flags().StringVar(&tags, "tags", "", "Replace tags")
	cmd.Flags().StringVar(&addTags, "add-tags", "", "Add tags")
	cmd.Flags().StringVar(&checklistItems, "checklist-items", "", "Replace checklist (lines or CSV)")
	cmd.Flags().StringVar(&prependChecklist, "prepend-checklist-items", "", "Prepend checklist")
	cmd.Flags().StringVar(&appendChecklist, "append-checklist-items", "", "Append checklist")
	cmd.Flags().StringVar(&listName, "list", "", "Destination project/area")
	cmd.Flags().StringVar(&listID, "list-id", "", "Destination project/area ID")
	cmd.Flags().StringVar(&heading, "heading", "", "Heading destination")
	cmd.Flags().StringVar(&headingID, "heading-id", "", "ID heading destination")
	cmd.Flags().BoolVar(&completed, "completed", false, "Set completed status")
	cmd.Flags().BoolVar(&canceled, "canceled", false, "Set canceled status")
	cmd.Flags().BoolVar(&reveal, "reveal", false, "Reveal item")
	cmd.Flags().BoolVar(&duplicate, "duplicate", false, "Duplicate before update")
	cmd.Flags().StringVar(&creationDate, "creation-date", "", "Creation date ISO8601")
	cmd.Flags().StringVar(&completionDate, "completion-date", "", "Completion date ISO8601")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func newURLAddProjectCmd() *cobra.Command {
	var (
		title, notes, when, deadline, tags, area, areaID, todos, creationDate, completionDate string
		completed, canceled, reveal                                                           bool
	)
	cmd := &cobra.Command{
		Use:   "add-project",
		Short: "things:///add-project",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			params := map[string]string{}
			setIfNotEmpty(params, "title", title)
			setIfNotEmpty(params, "notes", notes)
			setIfNotEmpty(params, "when", when)
			setIfChanged(cmd, params, "deadline", deadline)
			setIfNotEmpty(params, "tags", tags)
			setIfNotEmpty(params, "area", area)
			setIfNotEmpty(params, "area-id", areaID)
			setIfNotEmpty(params, "to-dos", normalizeChecklistInput(todos))
			setIfChanged(cmd, params, "creation-date", creationDate)
			setIfChanged(cmd, params, "completion-date", completionDate)
			setBoolIfChanged(cmd, params, "completed", completed)
			setBoolIfChanged(cmd, params, "canceled", canceled)
			setBoolIfChanged(cmd, params, "reveal", reveal)
			return runThingsURL(ctx, cfg, "add-project", params)
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "Project title")
	cmd.Flags().StringVar(&notes, "notes", "", "Notes")
	cmd.Flags().StringVar(&when, "when", "", "When")
	cmd.Flags().StringVar(&deadline, "deadline", "", "Deadline (vide pour effacer)")
	cmd.Flags().StringVar(&tags, "tags", "", "Tags")
	cmd.Flags().StringVar(&area, "area", "", "Destination area name")
	cmd.Flags().StringVar(&areaID, "area-id", "", "Destination area ID")
	cmd.Flags().StringVar(&todos, "to-dos", "", "Initial to-dos (lines or CSV)")
	cmd.Flags().StringVar(&creationDate, "creation-date", "", "Creation date ISO8601")
	cmd.Flags().StringVar(&completionDate, "completion-date", "", "Completion date ISO8601")
	cmd.Flags().BoolVar(&completed, "completed", false, "Create as completed")
	cmd.Flags().BoolVar(&canceled, "canceled", false, "Create as canceled")
	cmd.Flags().BoolVar(&reveal, "reveal", false, "Reveal project")
	return cmd
}

func newURLUpdateProjectCmd() *cobra.Command {
	var (
		id, title, notes, prependNotes, appendNotes, when, deadline, tags, addTags, area, areaID, creationDate, completionDate string
		completed, canceled, reveal, duplicate                                                                                 bool
	)
	cmd := &cobra.Command{
		Use:   "update-project",
		Short: "things:///update-project",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(id) == "" {
				return errors.New("--id is required")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			token, err := requireAuthToken(cfg)
			if err != nil {
				return err
			}
			params := map[string]string{
				"auth-token": token,
				"id":         id,
			}
			setIfChanged(cmd, params, "title", title)
			setIfChanged(cmd, params, "notes", notes)
			setIfChanged(cmd, params, "prepend-notes", prependNotes)
			setIfChanged(cmd, params, "append-notes", appendNotes)
			setIfChanged(cmd, params, "when", when)
			setIfChanged(cmd, params, "deadline", deadline)
			setIfChanged(cmd, params, "tags", tags)
			setIfChanged(cmd, params, "add-tags", addTags)
			setIfChanged(cmd, params, "area", area)
			setIfChanged(cmd, params, "area-id", areaID)
			setIfChanged(cmd, params, "creation-date", creationDate)
			setIfChanged(cmd, params, "completion-date", completionDate)
			setBoolIfChanged(cmd, params, "completed", completed)
			setBoolIfChanged(cmd, params, "canceled", canceled)
			setBoolIfChanged(cmd, params, "reveal", reveal)
			setBoolIfChanged(cmd, params, "duplicate", duplicate)
			return runThingsURL(ctx, cfg, "update-project", params)
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "Project ID")
	cmd.Flags().StringVar(&title, "title", "", "New title")
	cmd.Flags().StringVar(&notes, "notes", "", "New notes")
	cmd.Flags().StringVar(&prependNotes, "prepend-notes", "", "Prepend notes")
	cmd.Flags().StringVar(&appendNotes, "append-notes", "", "Append notes")
	cmd.Flags().StringVar(&when, "when", "", "When")
	cmd.Flags().StringVar(&deadline, "deadline", "", "Deadline (vide pour effacer)")
	cmd.Flags().StringVar(&tags, "tags", "", "Remplacer tags")
	cmd.Flags().StringVar(&addTags, "add-tags", "", "Add tags")
	cmd.Flags().StringVar(&area, "area", "", "Area destination")
	cmd.Flags().StringVar(&areaID, "area-id", "", "Destination area ID")
	cmd.Flags().StringVar(&creationDate, "creation-date", "", "Creation date ISO8601")
	cmd.Flags().StringVar(&completionDate, "completion-date", "", "Completion date ISO8601")
	cmd.Flags().BoolVar(&completed, "completed", false, "Set completed")
	cmd.Flags().BoolVar(&canceled, "canceled", false, "Set canceled")
	cmd.Flags().BoolVar(&reveal, "reveal", false, "Reveal project")
	cmd.Flags().BoolVar(&duplicate, "duplicate", false, "Duplicate before update")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func newURLShowCmd() *cobra.Command {
	var id, query, filter string
	cmd := &cobra.Command{
		Use:   "show",
		Short: "things:///show",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			params := map[string]string{}
			setIfNotEmpty(params, "id", id)
			setIfNotEmpty(params, "query", query)
			setIfNotEmpty(params, "filter", filter)
			if len(params) == 0 {
				return errors.New("fournir au moins --id ou --query")
			}
			return runThingsURL(ctx, cfg, "show", params)
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "ID to reveal (or built-in list)")
	cmd.Flags().StringVar(&query, "query", "", "Recherche quick find")
	cmd.Flags().StringVar(&filter, "filter", "", "Tags de filtre (CSV)")
	return cmd
}

func newURLSearchCmd() *cobra.Command {
	var query string
	cmd := &cobra.Command{
		Use:   "search",
		Short: "things:///search",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(query) == "" {
				return errors.New("--query is required")
			}
			return runThingsURL(ctx, cfg, "search", map[string]string{"query": query})
		},
	}
	cmd.Flags().StringVar(&query, "query", "", "Search text")
	_ = cmd.MarkFlagRequired("query")
	return cmd
}

func newURLVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "things:///version",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			return runThingsURL(ctx, cfg, "version", map[string]string{})
		},
	}
}

func newURLAddJSONCmd() *cobra.Command {
	var data string
	var reveal bool
	cmd := &cobra.Command{
		Use:   "add-json",
		Short: "things:///add-json",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(data) == "" {
				return errors.New("--data is required")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			params := map[string]string{"data": data}
			if strings.Contains(data, `"operation":"update"`) || strings.Contains(data, `"operation": "update"`) {
				token, err := requireAuthToken(cfg)
				if err != nil {
					return err
				}
				params["auth-token"] = token
			}
			setBoolIfChanged(cmd, params, "reveal", reveal)
			return runThingsURL(ctx, cfg, "add-json", params)
		},
	}
	cmd.Flags().StringVar(&data, "data", "", "Payload JSON")
	cmd.Flags().BoolVar(&reveal, "reveal", false, "Reveal created item")
	_ = cmd.MarkFlagRequired("data")
	return cmd
}

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

func newAddProjectCmd() *cobra.Command {
	var name, notes, listName string
	cmd := &cobra.Command{
		Use:   "add-project",
		Short: "Add a project",
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
			return runResult(ctx, cfg, scriptAddProject(cfg.bundleID, strings.TrimSpace(listName), name, notes))
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Project name")
	cmd.Flags().StringVar(&notes, "notes", "", "Notes")
	cmd.Flags().StringVar(&listName, "list", envOrDefault("THINGS_DEFAULT_LIST", defaultListName), "Destination area")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newAddListCmd() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "add-list",
		Short: "Add an area",
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
			script := fmt.Sprintf(`tell application id "%s"
  make new area with properties {name:"%s"}
  return "ok"
end tell`, cfg.bundleID, escapeApple(name))
			return runResult(ctx, cfg, script)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Area name")
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

func newEditProjectCmd() *cobra.Command {
	var sourceName, newName, notes string
	cmd := &cobra.Command{
		Use:   "edit-project",
		Short: "Edit a project (by name)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(sourceName) == "" {
				return errors.New("--name is required")
			}
			if strings.TrimSpace(newName) == "" && strings.TrimSpace(notes) == "" {
				return errors.New("specify --new-name and/or --notes")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptEditProject(cfg.bundleID, sourceName, newName, notes))
		},
	}
	cmd.Flags().StringVar(&sourceName, "name", "", "Project name")
	cmd.Flags().StringVar(&newName, "new-name", "", "New name")
	cmd.Flags().StringVar(&notes, "notes", "", "New notes")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newEditListCmd() *cobra.Command {
	var sourceName, newName string
	cmd := &cobra.Command{
		Use:   "edit-list",
		Short: "Rename an area",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(sourceName) == "" {
				return errors.New("--name is required")
			}
			if strings.TrimSpace(newName) == "" {
				return errors.New("--new-name is required")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			script := fmt.Sprintf(`tell application id "%s"
  set l to first list whose name is "%s"
  set name of l to "%s"
  return "ok"
end tell`, cfg.bundleID, escapeApple(sourceName), escapeApple(newName))
			return runResult(ctx, cfg, script)
		},
	}
	cmd.Flags().StringVar(&sourceName, "name", "", "Area name")
	cmd.Flags().StringVar(&newName, "new-name", "", "New name")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newDeleteTaskCmd() *cobra.Command {
	return newDeleteCmd("task", "delete-task")
}

func newDeleteProjectCmd() *cobra.Command {
	return newDeleteCmd("project", "delete-project")
}

func newDeleteListCmd() *cobra.Command {
	return newDeleteCmd("list", "delete-list")
}

func newDeleteCmd(kind, name string) *cobra.Command {
	var target string
	cmd := &cobra.Command{
		Use:   name,
		Short: "Delete an item",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(target) == "" {
				return errors.New("--name is required")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			script, err := scriptDelete(cfg.bundleID, kind, target)
			if err != nil {
				return err
			}
			return runResult(ctx, cfg, script)
		},
	}
	cmd.Flags().StringVar(&target, "name", "", "Item name")
	_ = cmd.MarkFlagRequired("name")
	return cmd
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

func newListSubtasksCmd() *cobra.Command {
	var taskName string
	cmd := &cobra.Command{
		Use:   "list-subtasks",
		Short: "List task subtasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(taskName) == "" {
				return errors.New("--task is required")
			}
			return runResult(ctx, cfg, scriptListSubtasks(cfg.bundleID, taskName))
		},
	}
	cmd.Flags().StringVar(&taskName, "task", "", "Task name parent")
	_ = cmd.MarkFlagRequired("task")
	return cmd
}

func newAddSubtaskCmd() *cobra.Command {
	var taskName, subtaskName string
	cmd := &cobra.Command{
		Use:   "add-subtask",
		Short: "Add a native checklist item to a task",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			taskName = strings.TrimSpace(taskName)
			subtaskName = strings.TrimSpace(subtaskName)
			if taskName == "" || subtaskName == "" {
				return errors.New("--task and --name are required")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			token, err := requireAuthToken(cfg)
			if err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptAppendChecklistByName(cfg.bundleID, taskName, []string{subtaskName}, token))
		},
	}
	cmd.Flags().StringVar(&taskName, "task", "", "Task name parent")
	cmd.Flags().StringVar(&subtaskName, "name", "", "Subtask name")
	_ = cmd.MarkFlagRequired("task")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newEditSubtaskCmd() *cobra.Command {
	var taskName, subtaskName, newName, notes string
	var subtaskIndex int
	cmd := &cobra.Command{
		Use:   "edit-subtask",
		Short: "Edit a subtask",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			taskName = strings.TrimSpace(taskName)
			subtaskName = strings.TrimSpace(subtaskName)
			newName = strings.TrimSpace(newName)
			notes = strings.TrimSpace(notes)
			if taskName == "" {
				return errors.New("--task is required")
			}
			if subtaskIndex <= 0 && subtaskName == "" {
				return errors.New("provide --index (>=1) or --name")
			}
			if newName == "" && notes == "" {
				return errors.New("provide --new-name and/or --notes")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptEditSubtask(cfg.bundleID, taskName, subtaskName, subtaskIndex, newName, notes))
		},
	}
	cmd.Flags().StringVar(&taskName, "task", "", "Task name parent")
	cmd.Flags().StringVar(&subtaskName, "name", "", "Target subtask name")
	cmd.Flags().IntVar(&subtaskIndex, "index", 0, "Target subtask index (1-based)")
	cmd.Flags().StringVar(&newName, "new-name", "", "New name")
	cmd.Flags().StringVar(&notes, "notes", "", "New notes")
	_ = cmd.MarkFlagRequired("task")
	return cmd
}

func newDeleteSubtaskCmd() *cobra.Command {
	var taskName, subtaskName string
	var subtaskIndex int
	cmd := &cobra.Command{
		Use:   "delete-subtask",
		Short: "Delete a subtask",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			taskName = strings.TrimSpace(taskName)
			subtaskName = strings.TrimSpace(subtaskName)
			if taskName == "" {
				return errors.New("--task is required")
			}
			if subtaskIndex <= 0 && subtaskName == "" {
				return errors.New("provide --index (>=1) or --name")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptDeleteSubtask(cfg.bundleID, taskName, subtaskName, subtaskIndex))
		},
	}
	cmd.Flags().StringVar(&taskName, "task", "", "Task name parent")
	cmd.Flags().StringVar(&subtaskName, "name", "", "Subtask name")
	cmd.Flags().IntVar(&subtaskIndex, "index", 0, "Subtask index (1-based)")
	_ = cmd.MarkFlagRequired("task")
	return cmd
}

func newCompleteSubtaskCmd() *cobra.Command {
	var taskName, subtaskName string
	var subtaskIndex int
	cmd := &cobra.Command{
		Use:   "complete-subtask",
		Short: "Mark subtask as completed",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			taskName = strings.TrimSpace(taskName)
			subtaskName = strings.TrimSpace(subtaskName)
			if taskName == "" {
				return errors.New("--task is required")
			}
			if subtaskIndex <= 0 && subtaskName == "" {
				return errors.New("provide --index (>=1) or --name")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptSetSubtaskStatus(cfg.bundleID, taskName, subtaskName, subtaskIndex, true))
		},
	}
	cmd.Flags().StringVar(&taskName, "task", "", "Task name parent")
	cmd.Flags().StringVar(&subtaskName, "name", "", "Subtask name")
	cmd.Flags().IntVar(&subtaskIndex, "index", 0, "Subtask index (1-based)")
	_ = cmd.MarkFlagRequired("task")
	return cmd
}

func newUncompleteSubtaskCmd() *cobra.Command {
	var taskName, subtaskName string
	var subtaskIndex int
	cmd := &cobra.Command{
		Use:   "uncomplete-subtask",
		Short: "Mark subtask as uncompleted",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			taskName = strings.TrimSpace(taskName)
			subtaskName = strings.TrimSpace(subtaskName)
			if taskName == "" {
				return errors.New("--task is required")
			}
			if subtaskIndex <= 0 && subtaskName == "" {
				return errors.New("provide --index (>=1) or --name")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptSetSubtaskStatus(cfg.bundleID, taskName, subtaskName, subtaskIndex, false))
		},
	}
	cmd.Flags().StringVar(&taskName, "task", "", "Task name parent")
	cmd.Flags().StringVar(&subtaskName, "name", "", "Subtask name")
	cmd.Flags().IntVar(&subtaskIndex, "index", 0, "Subtask index (1-based)")
	_ = cmd.MarkFlagRequired("task")
	return cmd
}
