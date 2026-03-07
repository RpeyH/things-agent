package main

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

const (
	networkIsolationNone             = "none"
	networkIsolationSandboxNoNetwork = "sandbox-no-network"
)

type offlineAppLaunchFunc func(context.Context, string) error

var newOfflineAppLaunch = func(mode string) (offlineAppLaunchFunc, error) {
	switch strings.TrimSpace(mode) {
	case "", networkIsolationNone:
		return nil, nil
	case networkIsolationSandboxNoNetwork:
		return launchAppSandboxNoNetwork, nil
	default:
		return nil, fmt.Errorf("unsupported network isolation mode %q", mode)
	}
}

func launchAppSandboxNoNetwork(ctx context.Context, bundleID string) error {
	cmd := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-n", "no-network", "open", "-b", bundleID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(output))
		if msg == "" {
			return fmt.Errorf("launch Things offline: %w", err)
		}
		return fmt.Errorf("launch Things offline: %w: %s", err, msg)
	}
	return nil
}
