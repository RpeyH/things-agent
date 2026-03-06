package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
)

func runThingsURL(ctx context.Context, cfg *runtimeConfig, command string, params map[string]string) error {
	thingsURL := "things:///" + command
	if encoded := encodeThingsURLParams(params); encoded != "" {
		thingsURL += "?" + encoded
	}
	return runResult(ctx, cfg, scriptOpenURL(cfg.bundleID, thingsURL))
}

func encodeThingsURLParams(params map[string]string) string {
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	return strings.ReplaceAll(values.Encode(), "+", "%20")
}

func scriptOpenURL(bundleID, rawURL string) string {
	return fmt.Sprintf(`tell application id "%s"
  open location "%s"
end tell
return "ok"`, escapeApple(bundleID), escapeApple(rawURL))
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
	var targetFile, timestamp string
	var unsafeLegacyRestore bool
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

			targetFile = strings.TrimSpace(targetFile)
			timestamp = strings.TrimSpace(timestamp)
			if targetFile != "" && timestamp != "" {
				return errors.New("use either --timestamp or --file, not both")
			}
			if targetFile != "" {
				if !unsafeLegacyRestore {
					return errors.New("--file requires --unsafe-legacy-restore")
				}
				if err := bm.RestoreFile(ctx, targetFile); err != nil {
					return err
				}
				fmt.Println(targetFile)
				return nil
			}

			if timestamp == "" {
				ts, err := bm.Latest(ctx)
				if err != nil {
					return err
				}
				timestamp = ts
			}

			restored, err := bm.Restore(ctx, timestamp)
			if err != nil {
				return err
			}
			for _, p := range restored {
				fmt.Println(p)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&timestamp, "timestamp", "", "Backup timestamp set to restore")
	cmd.Flags().StringVar(&targetFile, "file", "", "Legacy direct backup file path (unsafe)")
	cmd.Flags().BoolVar(&unsafeLegacyRestore, "unsafe-legacy-restore", false, "Allow legacy direct-file restore")
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
