package assistant

import (
	"sort"
	"strings"

	"whaleisle/internal/desktop/pluginrpc"
)

// Keep the original BotPage search projections and ordering on existing data;
// the native controls retain their selected-Bot scope and action identities.
func visibleBotTasks(rows []pluginrpc.DshbotTask, query, status string) []pluginrpc.DshbotTask {
	needle := strings.ToLower(strings.TrimSpace(query))
	var visible []pluginrpc.DshbotTask
	for _, task := range rows {
		if status != "all" && task.Status != status {
			continue
		}
		text := strings.Join([]string{task.ID, task.Task, task.FromName, task.ToName, task.FromID, task.ToID, task.ResultSummary, task.Error}, "\n")
		if needle == "" || strings.Contains(strings.ToLower(text), needle) {
			visible = append(visible, task)
		}
	}
	sort.SliceStable(visible, func(i, j int) bool {
		if visible[i].UpdatedAt != visible[j].UpdatedAt {
			return visible[i].UpdatedAt > visible[j].UpdatedAt
		}
		return visible[i].CreatedAt > visible[j].CreatedAt
	})
	return visible
}

func visibleBotRoutines(rows []pluginrpc.DshbotRoutine, botName, query string) []pluginrpc.DshbotRoutine {
	needle := strings.ToLower(strings.TrimSpace(query))
	var visible []pluginrpc.DshbotRoutine
	for _, routine := range rows {
		text := strings.Join([]string{routine.ID, routine.Name, routine.Prompt, routine.LastError, routine.BotID, botName}, "\n")
		if needle == "" || strings.Contains(strings.ToLower(text), needle) {
			visible = append(visible, routine)
		}
	}
	sort.SliceStable(visible, func(i, j int) bool {
		if visible[i].NextRunAt != visible[j].NextRunAt {
			return visible[i].NextRunAt < visible[j].NextRunAt
		}
		return visible[i].Name < visible[j].Name
	})
	return visible
}
