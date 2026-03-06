package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type runner struct {
	bundleID string
}

func newRunner(bundleID string) *runner {
	return &runner{
		bundleID: bundleID,
	}
}

func (r *runner) run(ctx context.Context, script string) (string, error) {
	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func (r *runner) ensureReachable(ctx context.Context) error {
	script := fmt.Sprintf(`tell application id "%s"
  return name
end tell`, r.bundleID)
	if _, err := r.run(ctx, script); err != nil {
		return fmt.Errorf("Things app not found (%s): %w", r.bundleID, err)
	}
	return nil
}

func scriptAllLists(bundleID string) string {
	return fmt.Sprintf(`tell application id "%s"
  get name of lists
end tell`, bundleID)
}

func scriptResolveTaskByName(taskName string) string {
	taskName = escapeApple(taskName)
	return fmt.Sprintf(`  try
    set t to first project whose name is "%s"
  on error
    set t to first «class tstk» whose name is "%s"
  end try
`, taskName, taskName)
}

func scriptAllProjects(bundleID string) string {
	return fmt.Sprintf(`tell application id "%s"
  get name of projects
end tell`, bundleID)
}

func scriptTasks(bundleID, listName, query string) string {
	listName = strings.TrimSpace(listName)
	query = strings.TrimSpace(query)
	if listName == "" && query == "" {
		return fmt.Sprintf(`tell application id "%s"
  return name of (every «class tstk»)
end tell`, bundleID)
	}
	if listName == "" {
		return fmt.Sprintf(`tell application id "%s"
  set q to "%s"
  return name of (every «class tstk» whose (name contains q or notes contains q))
end tell`, bundleID, escapeApple(query))
	}
	if query == "" {
		return fmt.Sprintf(`tell application id "%s"
  set l to first list whose name is "%s"
  return name of (every «class tstk» of l)
end tell`, bundleID, escapeApple(listName))
	}
	return fmt.Sprintf(`tell application id "%s"
  set q to "%s"
  set l to first list whose name is "%s"
  return name of (every «class tstk» of l whose (name contains q or notes contains q))
end tell`, bundleID, escapeApple(query), escapeApple(listName))
}

func scriptSearch(bundleID, listName, query string) string {
	return scriptTasks(bundleID, listName, query)
}

func scriptAddTask(bundleID, listName, name, notes, tags, due string) string {
	if strings.TrimSpace(listName) == "" {
		listName = envOrDefault("THINGS_DEFAULT_LIST", defaultListName)
	}
	parts := []string{fmt.Sprintf(`name:"%s"`, escapeApple(name))}
	if strings.TrimSpace(notes) != "" {
		parts = append(parts, fmt.Sprintf(`notes:"%s"`, escapeApple(notes)))
	}
	if strings.TrimSpace(tags) != "" {
		parts = append(parts, fmt.Sprintf(`tag names:"%s"`, escapeApple(tags)))
	}
	script := fmt.Sprintf(`tell application id "%s"
  set targetList to first list whose name is "%s"
  set t to make new «class tstk» at end of to dos of targetList with properties {%s}
`, bundleID, escapeApple(listName), strings.Join(parts, ", "))
	if strings.TrimSpace(due) != "" {
		script += fmt.Sprintf(`  set due date of t to date "%s"
`, due)
	}
	script += `  return id of t
end tell`
	return script
}

func requireAuthToken(cfg *runtimeConfig) (string, error) {
	token := strings.TrimSpace(cfg.authToken)
	if token == "" {
		return "", errors.New("auth-token is required for native checklist (Things > Settings > General). Use --auth-token or THINGS_AUTH_TOKEN")
	}
	return token, nil
}

func urlEncodeChecklist(items []string) string {
	return url.QueryEscape(strings.Join(items, "\n"))
}

func scriptSetChecklistByID(bundleID, taskID string, items []string, authToken string) string {
	return fmt.Sprintf(`tell application id "%s"
  set t to first to do whose id is "%s"
  set tid to id of t
end tell
open location "things:///update?auth-token=%s&id=" & tid & "&checklist-items=%s"
return tid`, bundleID, escapeApple(taskID), escapeApple(url.QueryEscape(authToken)), escapeApple(urlEncodeChecklist(items)))
}

func scriptAppendChecklistByName(bundleID, taskName string, items []string, authToken string) string {
	return fmt.Sprintf(`tell application id "%s"
  set t to first to do whose name is "%s"
  set tid to id of t
end tell
open location "things:///update?auth-token=%s&id=" & tid & "&append-checklist-items=%s"
return tid`, bundleID, escapeApple(taskName), escapeApple(url.QueryEscape(authToken)), escapeApple(urlEncodeChecklist(items)))
}

func parseCSVList(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

func scriptListLiteral(values []string) string {
	if len(values) == 0 {
		return "{}"
	}
	items := make([]string, 0, len(values))
	for _, value := range values {
		items = append(items, fmt.Sprintf(`"%s"`, escapeApple(value)))
	}
	return "{" + strings.Join(items, ", ") + "}"
}

func scriptAddProject(bundleID, listName, name, notes string) string {
	if strings.TrimSpace(listName) == "" {
		listName = envOrDefault("THINGS_DEFAULT_LIST", defaultListName)
	}
	script := fmt.Sprintf(`tell application id "%s"
  set targetList to first list whose name is "%s"
  set p to make new project at end of to dos of targetList with properties {name:"%s"}
`, bundleID, escapeApple(listName), escapeApple(name))
	if strings.TrimSpace(notes) != "" {
		script += fmt.Sprintf(`  set notes of p to "%s"
`, escapeApple(notes))
	}
	script += `  return id of p
end tell`
	return script
}

func scriptEditTask(bundleID, source, newName, notes, tags, moveTo, due, completion, creation, cancel string) (string, error) {
	if source == "" {
		return "", errors.New("source name is required")
	}
	script := fmt.Sprintf(`tell application id "%s"
%s`, bundleID, scriptResolveTaskByName(source))
	if strings.TrimSpace(newName) != "" {
		script += fmt.Sprintf(`  set name of t to "%s"
`, escapeApple(newName))
	}
	if strings.TrimSpace(notes) != "" {
		script += fmt.Sprintf(`  set notes of t to "%s"
`, escapeApple(notes))
	}
	if strings.TrimSpace(tags) != "" {
		script += fmt.Sprintf(`  set tag names of t to "%s"
`, escapeApple(tags))
	}
	if strings.TrimSpace(moveTo) != "" {
		script += fmt.Sprintf(`  move t to end of to dos of (first list whose name is "%s")
`, escapeApple(moveTo))
	}
	if strings.TrimSpace(due) != "" {
		script += fmt.Sprintf(`  set due date of t to date "%s"
`, due)
	}
	if strings.TrimSpace(completion) != "" {
		script += fmt.Sprintf(`  set completion date of t to date "%s"
`, completion)
	}
	if strings.TrimSpace(creation) != "" {
		script += fmt.Sprintf(`  set creation date of t to date "%s"
`, creation)
	}
	if strings.TrimSpace(cancel) != "" {
		script += fmt.Sprintf(`  set cancellation date of t to date "%s"
`, cancel)
	}
	script += `  return id of t
end tell`
	return script, nil
}

func scriptEditProject(bundleID, source, newName, notes string) string {
	script := fmt.Sprintf(`tell application id "%s"
  set p to first project whose name is "%s"
`, bundleID, escapeApple(source))
	if strings.TrimSpace(newName) != "" {
		script += fmt.Sprintf(`  set name of p to "%s"
`, escapeApple(newName))
	}
	if strings.TrimSpace(notes) != "" {
		script += fmt.Sprintf(`  set notes of p to "%s"
`, escapeApple(notes))
	}
	script += `  return id of p
end tell`
	return script
}

func scriptSetTaskNotes(bundleID, taskName, notes string) string {
	return fmt.Sprintf(`tell application id "%s"
%s  set notes of t to "%s"
  return id of t
end tell`, bundleID, scriptResolveTaskByName(taskName), escapeApple(notes))
}

func scriptAppendTaskNotes(bundleID, taskName, notes, separator string) string {
	if strings.TrimSpace(separator) == "" {
		separator = "\n"
	}
	return fmt.Sprintf(`tell application id "%s"
%s  if (notes of t is missing value) or (notes of t is "") then
    set notes of t to "%s"
  else
    set notes of t to (notes of t & "%s" & "%s")
  end if
  return id of t
end tell`, bundleID, scriptResolveTaskByName(taskName), escapeApple(notes), escapeApple(separator), escapeApple(notes))
}

func scriptSetTaskDate(bundleID, taskName, dueDate, deadlineDate string, clear bool) string {
	script := fmt.Sprintf(`tell application id "%s"
%s`, bundleID, scriptResolveTaskByName(taskName))
	if clear {
		script += `  set due date of t to missing value
`
	}
	if strings.TrimSpace(dueDate) != "" {
		script += fmt.Sprintf(`  set due date of t to date "%s"
`, dueDate)
	}
	if strings.TrimSpace(deadlineDate) != "" {
		script += fmt.Sprintf(`  set due date of t to date "%s"
`, deadlineDate)
	}
	script += `  return id of t
	end tell`
	return script
}

func scriptClearTaskDeadlineByName(bundleID, taskName, authToken string) string {
	return fmt.Sprintf(`tell application id "%s"
%s  set tid to id of t
end tell
open location "things:///update?auth-token=%s&id=" & tid & "&deadline="
return tid`, bundleID, scriptResolveTaskByName(taskName), escapeApple(url.QueryEscape(authToken)))
}

func scriptSetTaskTags(bundleID, taskName string, tags []string) string {
	tagText := strings.Join(tags, ", ")
	return fmt.Sprintf(`tell application id "%s"
%s  set tag names of t to "%s"
  return id of t
end tell`, bundleID, scriptResolveTaskByName(taskName), escapeApple(tagText))
}

func scriptAddTaskTags(bundleID, taskName string, tags []string) string {
	return fmt.Sprintf(`tell application id "%s"
%s  set existingTags to {}
  try
    set existingTags to tag names of t
  end try
  if existingTags is missing value then
    set existingTags to {}
  else if class of existingTags is text then
    set existingTags to {existingTags as string}
  end if
  repeat with aTag in %s
    if not (aTag is in existingTags) then
      set end of existingTags to (aTag as string)
    end if
  end repeat
  set AppleScript's text item delimiters to ", "
  set mergedTagsText to existingTags as text
  set AppleScript's text item delimiters to ""
  set tag names of t to mergedTagsText
  return id of t
end tell`, bundleID, scriptResolveTaskByName(taskName), scriptListLiteral(tags))
}

func scriptRemoveTaskTags(bundleID, taskName string, tags []string) string {
	return fmt.Sprintf(`tell application id "%s"
%s  set existingTags to {}
  try
    set existingTags to tag names of t
  end try
  if existingTags is missing value then
    set existingTags to {}
  else if class of existingTags is text then
    set existingTags to {existingTags as string}
  end if
  set filteredTags to {}
  repeat with aTag in existingTags
    if not (aTag is in %s) then
      set end of filteredTags to aTag
    end if
  end repeat
  set AppleScript's text item delimiters to ", "
  set filteredTagsText to filteredTags as text
  set AppleScript's text item delimiters to ""
  set tag names of t to filteredTagsText
  return id of t
end tell`, bundleID, scriptResolveTaskByName(taskName), scriptListLiteral(tags))
}

func scriptListSubtasks(bundleID, taskName string) string {
	taskName = strings.TrimSpace(taskName)
	return fmt.Sprintf(`tell application id "%s"
%s  try
    set subtasks to to dos of t
    set out to ""
    repeat with i from 1 to count subtasks
      set s to item i of subtasks
      set outLine to (i as string) & ". " & (name of s)
      if (notes of s is not missing value) and (notes of s is not "") then
        set outLine to outLine & " | " & (notes of s)
      end if
      if out is "" then
        set out to outLine
      else
        set out to out & linefeed & outLine
      end if
    end repeat
    if out is "" then
      return "No subtasks"
    end if
    return out
  on error
    return "No subtasks"
  end try
end tell`, bundleID, scriptResolveTaskByName(taskName))
}

func scriptAddSubtask(bundleID, taskName, subtaskName, notes string) string {
	script := fmt.Sprintf(`tell application id "%s"
%s  try
    set s to make new to do at end of to dos of t with properties {name:"%s"}
`, bundleID, scriptResolveTaskByName(taskName), escapeApple(subtaskName))
	if strings.TrimSpace(notes) != "" {
		script += fmt.Sprintf(`  set notes of s to "%s"
`, escapeApple(notes))
	}
	script += `  return id of s
  on error
    error "Cannot add a subtask to this item."
  end try
end tell`
	return script
}

func scriptFindSubtask(bundleID, taskName, subtaskName string, index int) string {
	taskName = strings.TrimSpace(taskName)
	subtaskName = strings.TrimSpace(subtaskName)
	var target string
	if index > 0 {
		target = fmt.Sprintf("item %d of to dos of t", index)
	} else {
		target = fmt.Sprintf(`first to do of to dos of t whose name is "%s"`, escapeApple(subtaskName))
	}
	return fmt.Sprintf(`tell application id "%s"
%s  try
    set s to %s
  on error
    error "No subtask found on this item."
  end try
`, bundleID, scriptResolveTaskByName(taskName), target)
}

func scriptShowTask(bundleID, taskName string, withSubtasks bool) string {
	subtasksBlock := "false"
	if withSubtasks {
		subtasksBlock = "true"
	}
	return fmt.Sprintf(`tell application id "%s"
%s  set out to "ID: " & (id of t)
  set out to out & linefeed & "Name: " & (name of t)
  set out to out & linefeed & "Type: " & (class of t as string)
  set out to out & linefeed & "Statut: " & (status of t as string)
  if due date of t is not missing value then
    set out to out & linefeed & "Due: " & (due date of t as string)
  else
    set out to out & linefeed & "Due: "
  end if
  if completion date of t is not missing value then
    set out to out & linefeed & "Completed on: " & (completion date of t as string)
  else
    set out to out & linefeed & "Completed on: "
  end if
  if creation date of t is not missing value then
    set out to out & linefeed & "Created on: " & (creation date of t as string)
  else
    set out to out & linefeed & "Created on: "
  end if
  set tagText to ""
  try
    set taskTags to tag names of t
    repeat with i from 1 to count taskTags
      set tagLine to item i of taskTags
      if tagText is "" then
        set tagText to tagLine
      else
        set tagText to tagText & ", " & tagLine
      end if
    end repeat
  end try
  set out to out & linefeed & "Tags: " & tagText
  if notes of t is missing value then
    set out to out & linefeed & "Notes: "
  else
    set out to out & linefeed & "Notes: " & (notes of t)
  end if
  if %s then
    try
      set subtasks to to dos of t
      set subtaskLines to "No subtasks"
      if (count subtasks) > 0 then
        set subtaskLines to ""
        repeat with i from 1 to count subtasks
          set s to item i of subtasks
          set lineItem to (i as string) & ". " & (name of s) & " [" & (status of s as string) & "]"
          if (notes of s is not missing value) and (notes of s is not "") then
            set lineItem to lineItem & " | " & (notes of s)
          end if
          if subtaskLines is "" then
            set subtaskLines to lineItem
          else
            set subtaskLines to subtaskLines & linefeed & lineItem
          end if
        end repeat
      end if
      set out to out & linefeed & "Subtasks:" & linefeed & subtaskLines
    on error
      set out to out & linefeed & "Subtasks: not supported"
    end try
  end if
  return out
end tell`, bundleID, scriptResolveTaskByName(taskName), subtasksBlock)
}

func scriptEditSubtask(bundleID, taskName, subtaskName string, index int, newName, notes string) string {
	script := scriptFindSubtask(bundleID, taskName, subtaskName, index)
	if newName != "" {
		script += fmt.Sprintf(`  set name of s to "%s"
`, escapeApple(newName))
	}
	if notes != "" {
		script += fmt.Sprintf(`  set notes of s to "%s"
`, escapeApple(notes))
	}
	script += `  return id of s
end tell`
	return script
}

func scriptDeleteSubtask(bundleID, taskName, subtaskName string, index int) string {
	script := scriptFindSubtask(bundleID, taskName, subtaskName, index)
	script += `  delete s
  return "ok"
end tell`
	return script
}

func scriptSetSubtaskStatus(bundleID, taskName, subtaskName string, index int, done bool) string {
	state := "open"
	if done {
		state = "completed"
	}
	script := scriptFindSubtask(bundleID, taskName, subtaskName, index)
	script += fmt.Sprintf(`  set status of s to %s
  return id of s
end tell`, state)
	return script
}

func scriptDelete(bundleID, kind, name string) (string, error) {
	var subject string
	switch kind {
	case "task":
		subject = "«class tstk»"
	case "project":
		subject = "project"
	case "list":
		subject = "list"
	default:
		return "", fmt.Errorf("unknown kind: %s", kind)
	}
	return fmt.Sprintf(`tell application id "%s"
  delete first %s whose name is "%s"
end tell`, bundleID, subject, escapeApple(name)), nil
}

func scriptCompleteTask(bundleID, name string, done bool) string {
	state := "open"
	if done {
		state = "completed"
	}
	return fmt.Sprintf(`tell application id "%s"
  set t to first «class tstk» whose name is "%s"
  set status of t to %s
  return id of t
end tell`, bundleID, escapeApple(name), state)
}

type backupManager struct {
	dataDir string
}

func newBackupManager(dataDir string) *backupManager {
	return &backupManager{dataDir: dataDir}
}

func (bm *backupManager) Create(ctx context.Context) ([]string, error) {
	_ = ctx
	dir, err := bm.ensureBackupDir()
	if err != nil {
		return nil, err
	}
	ts := time.Now().Format(backupTSFormat)
	var created []string
	for _, base := range []string{"main.sqlite", "main.sqlite-shm", "main.sqlite-wal"} {
		src := filepath.Join(bm.dataDir, base)
		if _, err := os.Stat(src); err != nil {
			continue
		}
		dst := filepath.Join(dir, base+"."+ts+".bak")
		if err := copyFile(src, dst); err != nil {
			return nil, err
		}
		created = append(created, dst)
	}
	if len(created) == 0 {
		return nil, errors.New("no backupable database file found")
	}
	if err := bm.prune(ctx, maxBackupsToKeep); err != nil {
		return nil, fmt.Errorf("backup created but retention failed: %w", err)
	}
	sort.Strings(created)
	return created, nil
}

func (bm *backupManager) Latest(ctx context.Context) (string, error) {
	_ = ctx
	candidates, err := bm.allTimestamps()
	if err != nil {
		return "", err
	}
	if len(candidates) == 0 {
		return "", errors.New("no backup available")
	}
	return candidates[0], nil
}

func (bm *backupManager) FilesForTimestamp(ctx context.Context, ts string) ([]string, error) {
	_ = ctx
	var paths []string
	for _, base := range []string{"main.sqlite", "main.sqlite-shm", "main.sqlite-wal"} {
		candidate := filepath.Join(bm.backupPath(), base+"."+ts+".bak")
		if _, err := os.Stat(candidate); err == nil {
			paths = append(paths, candidate)
		}
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no file for timestamp %s", ts)
	}
	return paths, nil
}

func (bm *backupManager) Restore(ctx context.Context, ts string) ([]string, error) {
	_ = ctx
	files, err := bm.FilesForTimestamp(ctx, ts)
	if err != nil {
		return nil, err
	}
	for _, src := range files {
		if err := bm.RestoreFile(ctx, src); err != nil {
			return nil, err
		}
	}
	return files, nil
}

func (bm *backupManager) RestoreFile(ctx context.Context, path string) error {
	_ = ctx
	base := filepath.Base(path)
	var baseTarget string
	if strings.HasPrefix(base, "main.sqlite.") {
		baseTarget = "main.sqlite"
	} else if strings.HasPrefix(base, "main.sqlite-shm.") {
		baseTarget = "main.sqlite-shm"
	} else if strings.HasPrefix(base, "main.sqlite-wal.") {
		baseTarget = "main.sqlite-wal"
	} else {
		return fmt.Errorf("nom de backup invalide: %s", base)
	}
	dst := filepath.Join(bm.dataDir, baseTarget)
	return copyFile(path, dst)
}

func (bm *backupManager) prune(ctx context.Context, keep int) error {
	_ = ctx
	if keep <= 0 {
		return nil
	}
	timestamps, err := bm.allTimestamps()
	if err != nil {
		return err
	}
	if len(timestamps) <= keep {
		return nil
	}
	for _, ts := range timestamps[keep:] {
		for _, base := range []string{"main.sqlite", "main.sqlite-shm", "main.sqlite-wal"} {
			target := filepath.Join(bm.backupPath(), base+"."+ts+".bak")
			if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
	}
	return nil
}

func (bm *backupManager) allTimestamps() ([]string, error) {
	dir, err := bm.ensureBackupDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	tsSet := map[string]struct{}{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ts := extractTimestamp(e.Name())
		if ts != "" {
			tsSet[ts] = struct{}{}
		}
	}
	var ts []string
	for k := range tsSet {
		ts = append(ts, k)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(ts)))
	return ts, nil
}

func (bm *backupManager) backupPath() string {
	return filepath.Join(bm.dataDir, backupDirName)
}

func (bm *backupManager) ensureBackupDir() (string, error) {
	path := bm.backupPath()
	if err := os.MkdirAll(path, 0o755); err != nil {
		return "", err
	}
	return path, nil
}

func parseToAppleDate(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	t, err := parseDate(value)
	if err != nil {
		return "", err
	}
	return t.Format("2006-01-02 15:04:05"), nil
}

func parseDate(v string) (time.Time, error) {
	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
		"02/01/2006 15:04:05",
		"02/01/2006 15:04",
		"02/01/2006",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, v); err == nil {
			return t, nil
		}
	}
	if t, err := time.ParseInLocation("2006-01-02", v, time.Local); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("unrecognized date format: %s", v)
}

func inferTimestamp(file string) string {
	base := filepath.Base(file)
	candidates := []string{
		"main.sqlite.",
		"main.sqlite-shm.",
		"main.sqlite-wal.",
	}
	for _, p := range candidates {
		if strings.HasPrefix(base, p) && strings.HasSuffix(base, ".bak") {
			return strings.TrimSuffix(strings.TrimPrefix(base, p), ".bak")
		}
	}
	return ""
}

func extractTimestamp(file string) string {
	base := filepath.Base(file)
	return inferTimestamp(base)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

func resolveDataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	pattern := filepath.Join(home, thingsDataPattern)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", fmt.Errorf("failed to resolve Things data dir: %w", err)
	}
	sort.Strings(matches)
	for _, candidate := range matches {
		if st, err := os.Stat(filepath.Join(candidate, "main.sqlite")); err == nil && !st.IsDir() {
			return candidate, nil
		}
	}
	return "", errors.New("could not resolve Things data dir automatically; set THINGS_DATA_DIR")
}

func envOrDefault(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok && strings.TrimSpace(value) != "" {
		return value
	}
	return defaultValue
}

func escapeApple(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "\"", "\\\"")
	value = strings.ReplaceAll(value, "\n", "\\n")
	return value
}
