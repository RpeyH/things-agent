package main

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"
)

func newTagsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tags",
		Short: "Manage Things tags",
	}
	cmd.AddCommand(
		newTagsListCmd(),
		newTagsSearchCmd(),
		newTagsAddCmd(),
		newTagsEditCmd(),
		newTagsDeleteCmd(),
	)
	return cmd
}

func newTagsListCmd() *cobra.Command {
	var query string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tags",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptListTags(cfg.bundleID, query))
		},
	}
	cmd.Flags().StringVar(&query, "query", "", "Optional filter by tag name")
	return cmd
}

func newTagsSearchCmd() *cobra.Command {
	var query string
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search tags by name",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(query) == "" {
				return errors.New("--query is required")
			}
			return runResult(ctx, cfg, scriptListTags(cfg.bundleID, query))
		},
	}
	cmd.Flags().StringVar(&query, "query", "", "Search text")
	_ = cmd.MarkFlagRequired("query")
	return cmd
}

func newTagsAddCmd() *cobra.Command {
	var name, parent string
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Create a tag",
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
			return runResult(ctx, cfg, scriptAddTag(cfg.bundleID, name, parent))
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Tag name")
	cmd.Flags().StringVar(&parent, "parent", "", "Parent tag name (optional)")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newTagsEditCmd() *cobra.Command {
	var name, newName, parent string
	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Edit a tag",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			name = strings.TrimSpace(name)
			newName = strings.TrimSpace(newName)
			parent = strings.TrimSpace(parent)
			if name == "" {
				return errors.New("--name is required")
			}
			parentChanged := cmd.Flags().Changed("parent")
			if newName == "" && !parentChanged {
				return errors.New("provide --new-name and/or --parent")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptEditTag(cfg.bundleID, name, newName, parent, parentChanged))
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Existing tag name")
	cmd.Flags().StringVar(&newName, "new-name", "", "New tag name")
	cmd.Flags().StringVar(&parent, "parent", "", "Parent tag name (empty to clear parent)")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newTagsDeleteCmd() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a tag",
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
			return runResult(ctx, cfg, scriptDeleteTag(cfg.bundleID, name))
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Tag name")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}
