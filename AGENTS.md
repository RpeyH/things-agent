# AGENTS - things-agent

`README.md` is for the human user who asks an AI coding agent such as Codex, Claude Code, Open Code, or similar tools to operate Things. `AGENTS.md` is for the AI operator that actually runs the CLI.

## Priority

Apply these rules in this order:

1. Session Start
2. Data Access Rule
3. Strict CLI-Only Execution Rule
4. Backup Policy
5. Verification Policy

## Session Start

- For each new interaction session touching this repository, run `things-agent session-start`.
- `session-start` is mandatory at the beginning of a session.
- At the beginning of each session, also run:
  - `things-agent date`
  - `things-agent --help`
- Then build a fresh read-only picture of Things with:
  - `things-agent areas`
  - `things-agent projects --json`
  - `things-agent lists`
  - `things-agent tasks --list "<localized-today-list>" --json`
  - `things-agent tasks --list "<localized-capture-list>" --json`
- Use the exact localized names returned by `things-agent lists` for the built-in `Today` list and the built-in capture list (`Inbox`, `À classer`, or another locale-specific label).
- The CLI keeps at most 50 backups. Required timestamp format: `YYYY-MM-DD:HH-MM-SS`.

## Data Access Rule

- The agent must never interact directly with the Things database (`.thingsdatabase`).
- All normal reads and writes must go through the CLI.
- Internal CLI restore code may use a narrowly scoped SQLite step to clear local sync metadata after a package swap; this exception is for CLI implementation only, never for agent-authored ad hoc access.

## Strict CLI-Only Execution Rule

- The agent must **only** use `things-agent` commands to change Things state.
- No bypass via ad hoc AppleScript, manual URL Scheme calls, UI automation, or direct database access.
- If the CLI lacks a feature, propose adding it to the CLI instead of bypassing it.
- If a Things-related action outside the CLI is truly needed, ask the user first and wait for explicit approval.

## General Operating Rules

- Treat the installed binary as the live command authority.
- If `things-agent --help` contradicts this file, follow the binary and report the documentation drift.
- Use `things-agent version` and `things-agent --help` before long or risky operations.
- App lifecycle helpers exist when needed: `things-agent open` and `things-agent close`.
- Resolve temporal words such as `now`, `today`, `this week`, or `current` from `things-agent date`, not from unstated assumptions.
- If the agent writes in a language other than English, it must use that language correctly, including accents, diacritics, punctuation, and spacing conventions.

## Domain Terms

- `area`: a user-managed Things area.
- `list`: a generic Things list name used for read filters and the official URL Scheme. This includes built-in lists such as `Inbox`, `Today`, `Logbook`, and `Archive`, plus area names where the Things API expects a generic list selector.
- `project`: a Things project.
- `task`: a top-level to-do.
- `checklist item`: a lightweight native checklist line inside a task.
- `child task`: a structured child to-do under a project.

## Backup Policy

- Do not create an extra backup before every mutation.
- Outside `session-start`, recommend or trigger an explicit backup only before heavy, destructive, or highly transformative operations.
- `backup [--settle <duration>]` creates an `explicit` checkpoint.
- `session-start` creates a `session` checkpoint.
- `restore` creates a `safety` checkpoint before mutating live state.
- All backup kinds share the same package snapshot format in `ThingsData-*/Backups`.
- The backup index is metadata only. It helps the agent choose a snapshot; it is not a separate restore engine.
- If Things was already open before a backup, the CLI should reopen it after the backup completes.
- Use `backup --settle 10s` or more if a checkpoint must include very recent writes.

## Verification Policy

- After each requested action, verify the result and report it.
- On failure, report the exact command and returned error.
- When multiple operations depend on shared state, use this order:
  1. optional backup if the operation is heavy, destructive, or transformative
  2. read/write action(s)
  3. verification

## Restore Policy

- Preferred restore path: `restore [--timestamp <YYYY-MM-DD:HH-MM-SS>] [--network-isolation sandbox-no-network] [--offline-hold <duration>] [--reopen-online] [--dry-run] [--json]`
- Read-only restore commands:
  - `things-agent restore list`
  - `things-agent restore verify`
  - `things-agent restore preflight`
- `restore --network-isolation sandbox-no-network` is the safest DB restore path.
- After any successful restore, explicitly tell the user to verify the restored data first and then re-enable Things Cloud manually if they use sync.
- Do not present Things Cloud reactivation as automatic or guaranteed by the CLI.

## Command Selection Rules

- Always discover the live command surface with `things-agent --help`.
- Use `things-agent <command> --help` before risky or unfamiliar operations.
- Prefer the high-level CLI over the URL bridge when both exist.
- Prefer `--id` over `--name` for mutations when an ID is already known.
- Prefer `area` wording and area commands for high-level CRUD and move flows.
- Before destructive, structural, or hard-to-reverse operations, inspect the exact command help again instead of relying on memory.
- Re-check help especially for:
  - backup and restore flows
  - create and move commands for areas, projects, and tasks
  - checklist and child-task commands
  - reorder commands
  - the URL bridge, especially `things-agent url json --data '<json-array>'`

## Agent-Facing Limits

- Reorder is partially supported.
  - `reorder-project-items` and `reorder-area-items` rely on a private/experimental backend.
  - `reorder-area-items` cannot freely interleave projects and tasks, and no stable backend exists for checklist-item reorder, heading reorder, or sidebar area reorder.
  - No stable backend is available yet for checklist-item reorder, heading reorder, or sidebar area reorder.
  - Warn before using reorder on important data, then verify the final order carefully.

- Headings are not reliably automated.
  - No reliable headless heading backend is available.
  - Runtime validation showed that `things:///json` project updates did not create visible headings, private JSON read paths did not expose headings, and `move-task --to-heading` may return `ok` even when nothing changes.
  - `move-task --to-heading` may return `ok` even when nothing changes.
  - Tell the user to create headings manually in Things, then continue with the CLI for the rest.

- Recurring tasks are not supported by the CLI yet.
  - No reliable documented automation backend is available for recurrence.
  - Tell the user recurring tasks must be created or edited manually in Things.

- Restore has a preferred mode.
  - Prefer `restore --network-isolation sandbox-no-network`.
  - Do not prefer `--reopen-online` unless the user explicitly wants that convenience/risk tradeoff.
  - The CLI restores from package snapshots in `ThingsData-*/Backups`.
  - After restore, tell the user to verify the restored data first and re-enable Things Cloud manually if needed.
