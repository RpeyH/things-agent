package main

import (
	"fmt"
	"strings"
)

func scriptAllLists(bundleID string) string {
	return fmt.Sprintf(`tell application id "%s"
  get name of lists
end tell`, bundleID)
}

func scriptResolveTaskByName(taskName string) string {
	taskName = escapeApple(taskName)
	return fmt.Sprintf(`  try
    set projectMatches to every project whose name is "%s"
    set taskMatches to every «class tstk» whose name is "%s"
    set projectCount to count of projectMatches
    set taskCount to count of taskMatches
    set totalCount to projectCount + taskCount
    if totalCount is 0 then error "No item found with this name."
    if totalCount is greater than 1 then error "Ambiguous item name; use a unique name."
    if projectCount is 1 then
      set t to item 1 of projectMatches
    else
      set t to item 1 of taskMatches
    end if
  on error errMsg
    error errMsg
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
