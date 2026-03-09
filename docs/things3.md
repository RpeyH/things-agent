# Things 3 Automation Notes

This file is a compact working reference for future `things-agent` development.
It focuses on Things' official local automation surfaces and the practical limits that matter for this project.

## Overview

Things does not provide a public API.
The official local automation surfaces are:

- AppleScript on macOS
- the Things URL Scheme
- Apple Shortcuts

For `things-agent`, the main surfaces are AppleScript and the URL Scheme.
Shortcuts matter mainly as an official signal of what exists beyond AppleScript.

Project rule:

- normal reads and writes should stay on top of official automation surfaces
- direct database access is out of bounds for normal operations
- the only narrow exception is the internal restore path

## AppleScript

### What it is good for

AppleScript is the most natural fit for local app control on macOS when we want to:

- read areas, projects, to-dos, and tags
- create, edit, move, complete, or delete common entities
- drive simple UI actions such as `show`, `edit`, or quick entry
- work against the live app model instead of assembling URLs

This is the stable base for most everyday CLI flows.

### Important capabilities

Officially documented AppleScript coverage includes:

- built-in lists, areas, projects, to-dos, and tags
- creation and deletion of to-dos, projects, and areas
- moving to-dos and projects between lists/parents
- setting notes, tags, completion, cancellation, and other documented properties
- UI interactions such as `show`, `edit`, and `show quick entry panel`

### Important limits

AppleScript is Mac-only.

If a feature is not documented in Things' AppleScript docs, treat it as unsupported.
Cultured Code explicitly says that if it is not documented, it is not possible via AppleScript.

Important practical limit for this project:

- headings and checklist editing are not the right AppleScript backend
- for more power, Cultured Code points users toward Apple Shortcuts

### Important semantic trap

In Things' AppleScript model, `due date` means deadline, not start date.

That matters because a naive implementation will easily confuse:

- start/schedule semantics
- deadline semantics

Any code touching dates should be checked carefully against the official docs and runtime behavior.

### Practical notes

- Built-in list names are localized and must match the names shown in the app.
- Scriptability is limited to what the Things dictionary documents.
- Script Editor can inspect the live dictionary via `File -> Open Dictionary...`.

## URL Scheme

### What it is good for

The URL Scheme is the best fit when we want:

- official command-style automation
- checklist mutation
- heading-aware placement or updates
- JSON import of larger structures
- x-callback style integrations

For `things-agent`, this is the preferred backend for capabilities that AppleScript does not cover well.

### Supported command families

Official command surface:

- `add`
- `add-project`
- `update`
- `update-project`
- `show`
- `search`
- `version`
- `json`

Also note:

- `add-json` exists only as deprecated legacy behavior
- `json` is the canonical developer-oriented endpoint

### Important capabilities

The URL Scheme supports features that matter directly to this project:

- `auth-token` protected updates of existing data
- checklist item replacement/prepend/append on to-dos
- `list` / `list-id` targeting of projects or areas
- `heading` / `heading-id` targeting inside projects
- `creation-date` and `completion-date` in update-style commands
- JSON import with projects, to-dos, headings, and checklist items
- x-callback parameters such as `x-success`, `x-error`, and `x-cancel`

### Important limits and traps

- update-style commands require an auth token
- the first URL invocation may require explicit user enablement in Things settings
- natural-language dates must be in English
- `show` and `search` are navigation-oriented, not general read APIs
- JSON payloads must be a top-level array

### JSON notes

The `json` command is the most expressive official write surface.
It is useful for:

- importing projects with nested to-dos
- creating headings
- creating checklist items structurally

Cultured Code also provides `ThingsJSONCoder`, a helper repository for generating this JSON.

## Apple Shortcuts

`things-agent` does not use Shortcuts as its primary backend, but Shortcuts matter for capability boundaries.

Cultured Code's docs explicitly position Shortcuts as offering functionality beyond AppleScript, including editing headings and checklists.

That is useful as a product signal:

- if AppleScript cannot do it and the docs point to Shortcuts, do not pretend AppleScript can do it
- URL Scheme and Shortcuts are often the right official fallback for richer mutations

## Backend choice for this project

Use AppleScript when:

- reading current state
- doing common local CRUD on areas, projects, tasks, and tags
- performing simple live-app actions

Use the URL Scheme when:

- updating checklist items
- targeting headings
- importing structured trees with `json`
- using callback-oriented integration patterns
- updating fields whose support is clearer or safer through URL commands

Treat Shortcuts as:

- an official capability reference
- a sign that some richer actions are supported by Things, but not necessarily through AppleScript

## Known traps for future work

- localized built-in list names can break assumptions
- AppleScript and URL Scheme do not expose the same feature set
- date semantics are easy to get wrong, especially around deadline vs start/schedule
- auth-token requirements must stay scoped to the URL update surfaces that need them
- unsupported automation should fail explicitly instead of silently claiming success

## Official references

- AppleScript intro:
  https://culturedcode.com/things/support/articles/2803572/
- AppleScript commands:
  https://culturedcode.com/things/support/articles/4562654/
- URL Scheme:
  https://culturedcode.com/things/support/articles/2803573/
- Shortcuts actions:
  https://culturedcode.com/things/support/articles/9596775/
- FAQ / API statement:
  https://culturedcode.com/things/support/articles/2967034/
- JSON helper repo:
  https://github.com/culturedcode/ThingsJSONCoder

## Project takeaway

For `things-agent`, the durable model is:

- AppleScript for most reads and straightforward local mutations
- URL Scheme for richer structured writes and capability gaps
- no normal direct database access
- restore as the only narrow exception requiring deeper storage handling
