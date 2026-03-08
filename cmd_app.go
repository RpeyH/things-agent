package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newOpenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "open",
		Short: "Open Things",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if err := (scriptAppController{runner: cfg.runner}).Activate(ctx, cfg.bundleID); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "ok")
			return nil
		},
	}
}

func newCloseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "close",
		Short: "Close Things",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if err := (scriptAppController{runner: cfg.runner}).Quit(ctx, cfg.bundleID); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "ok")
			return nil
		},
	}
}
