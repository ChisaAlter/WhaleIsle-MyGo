package session

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

// ListSlashCommands is the synchronous adapter for non-UI callers.
// Paint uses EnsureSlashCommands and Snapshot instead.
func (s *Store) ListSlashCommands(ctx context.Context) []api.ListResultItem2 {
	s.mu.Lock()
	client, sessionID := s.client, s.selectedID
	s.mu.Unlock()
	if client == nil || sessionID == "" {
		return nil
	}
	var out api.ListResult3
	if err := client.Call(ctx, "commands/list", map[string]any{"agentId": sessionID}, &out); err != nil {
		return nil
	}
	return cloneCommandDescriptors(out)
}

// ExecuteCommandLine runs commands/execute for the captured session and reports
// application failures. Attachment-free commands always submit an empty array.
func (s *Store) ExecuteCommandLine(ctx context.Context, line string) error {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}
	s.mu.Lock()
	client, sessionID, generation := s.client, s.selectedID, s.followGeneration
	s.mu.Unlock()
	if client == nil || sessionID == "" {
		return ErrNoClient
	}
	err := executeCommand(ctx, client, sessionID, line, nil)
	s.mu.Lock()
	current := s.sessionResultCurrentLocked(ctx, client, sessionID, generation)
	if current && err != nil {
		if s.lastAction != err.Error() {
			s.invalidateViewLocked()
		}
		s.lastAction = err.Error()
	}
	s.mu.Unlock()
	if current && err != nil {
		s.fire()
	}
	return err
}

func executeCommand(ctx context.Context, client *dsh.Client, id, line string, attachments []json.RawMessage) error {
	if attachments == nil {
		attachments = make([]json.RawMessage, 0)
	}
	var result api.ExecuteResult
	if err := client.Call(ctx, "commands/execute", map[string]any{
		"agentId": id, "line": line, "submittedAttachments": attachments,
	}, &result); err != nil {
		return err
	}
	var outcome struct {
		Kind string `json:"kind"`
		Text string `json:"text"`
	}
	if len(result.Result) == 0 || string(result.Result) == "null" {
		return errors.New("command was not accepted; check its name and retry")
	}
	if err := json.Unmarshal(result.Result, &outcome); err != nil {
		return err
	}
	switch outcome.Kind {
	case "success":
		return nil
	case "error":
		if strings.TrimSpace(outcome.Text) != "" {
			return errors.New(outcome.Text)
		}
		return errors.New("command failed")
	default:
		return errors.New("command was not accepted; check its name and retry")
	}
}

// commandForLine follows the original enter-resolution rule: a free-input
// command owns its arguments; a no-input command owns only its bare token.
func commandForLine(commands []api.ListResultItem2, line string) (api.ListResultItem2, bool) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "/") {
		return api.ListResultItem2{}, false
	}
	token := line
	if end := strings.IndexAny(line, " \t\r\n"); end >= 0 {
		token = line[:end]
	}
	for _, command := range commands {
		if token == "/"+command.Name && (command.Input != nil || token == line) {
			return command, true
		}
	}
	return api.ListResultItem2{}, false
}
