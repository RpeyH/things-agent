# agent-things

CLI Go pour piloter Things (macOS) via AppleScript uniquement avec `cobra`.

Voir [AGENTS.md](/workspace/things-agent/AGENTS.md) pour les règles opérationnelles (backup initial de session, retention, sécurité, conventions).

## Prérequis

- macOS
- Application Things installée
- `osascript`

Le CLI n'accède jamais directement à la base SQLite de Things.

## Installation

```bash
cd /workspace/things-agent
go mod tidy
go build -o /usr/local/bin/agent-things .
```

Vous pouvez aussi définir un nom de binaire différent lors du build.

## Utilisation

```bash
agent-things session-start
agent-things backup
agent-things tasks --list "À classer"
agent-things search --query "Wagner"
agent-things add-task --name "Dizer ola" --notes "Mensagem" --list "À classer"
agent-things complete-task --name "Dizer ola"
agent-things list-subtasks --task "Dizer ola"
agent-things add-subtask --task "Dizer ola" --name "Vérifier le message"
```

### Commandes utiles

- `session-start`
- `backup`, `restore [--file <chemin ou timestamp>]`
- `lists`, `projects`
- `tasks [--list <nom>] [--query <texte>]`
- `search --query <texte> [--list <nom>]`
- `add-task`, `edit-task`, `delete-task`, `complete-task`, `uncomplete-task`
- `add-project`, `edit-project`, `delete-project`
- `add-list`, `edit-list`, `delete-list`
- `add-subtask`, `edit-subtask`, `delete-subtask`, `complete-subtask`, `uncomplete-subtask`, `list-subtasks`
- `set-tags`
- `version`
