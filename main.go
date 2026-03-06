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
