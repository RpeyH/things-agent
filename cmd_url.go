package main

import (
	"context"
	"errors"
	"strings"

	"github.com/spf13/cobra"
)

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
