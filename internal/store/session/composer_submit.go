package session

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"whaleisle/internal/dsh/api"
)

// ComposerAttachment retains the upload receipt, not just its durable file ID.
// SessionID prevents a staged receipt from being reused in a different session.
type ComposerAttachment struct {
	Path      string
	Name      string
	SessionID string
	Upload    api.FileUploadValue
	ImageData string
	MediaType string
}

// PrepareComposerAttachment reads actual local bytes and uses the original wire
// contract: encoded images, or staged file upload receipts for ordinary files.
func (s *Store) PrepareComposerAttachment(ctx context.Context, id string, a ComposerAttachment) (ComposerAttachment, error) {
	if a.SessionID == id && (a.Upload.ReceiptId != "" || a.ImageData != "") {
		return a, nil
	}
	raw, err := os.ReadFile(a.Path)
	if err != nil {
		return a, err
	}
	a.Name = filepath.Base(a.Path)
	a.MediaType = http.DetectContentType(raw)
	switch a.MediaType {
	case "image/png", "image/jpeg", "image/gif", "image/webp":
		a.ImageData = base64.StdEncoding.EncodeToString(raw)
	default:
		a.Upload, err = s.UploadFileForSession(ctx, id, a.Path)
		if err != nil {
			return a, err
		}
	}
	a.SessionID = id
	return a, nil
}

func (s *Store) UploadFileForSession(ctx context.Context, id, path string) (api.FileUploadValue, error) {
	var out api.FileUploadValue
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if client == nil || id == "" {
		return out, ErrNoClient
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return out, err
	}
	name := filepath.Base(path)
	err = client.Call(ctx, "fileUploads/upload", map[string]any{"agentId": id, "request": api.EncodedFileUploadRequest{Data: base64.StdEncoding.EncodeToString(raw), Name: &name}}, &out)
	if err != nil {
		return out, err
	}
	if strings.TrimSpace(out.ReceiptId) == "" || strings.TrimSpace(out.File.AttachmentId) == "" {
		return out, errors.New("upload returned an incomplete file receipt")
	}
	return out, nil
}

// CreateComposerSession creates a target for a home draft without switching an
// unrelated session selected while the request was in flight.
func (s *Store) CreateComposerSession(ctx context.Context, cwd string) (string, error) {
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if client == nil {
		return "", ErrNoClient
	}
	if strings.TrimSpace(cwd) == "" {
		return "", errors.New("choose a workspace before sending")
	}
	var out api.SessionCreateValue
	err := client.Call(ctx, "session/create", map[string]any{"request": api.SessionCreateRequest{Cwd: &cwd}}, &out)
	if err != nil {
		return "", err
	}
	if out.SessionId == "" {
		return "", errors.New("session/create returned no session")
	}
	return out.SessionId, nil
}

// SubmitComposer is synchronous so its owner can clear only after acceptance.
// The caller runs it off the UI thread and retains the draft on every error.
func (s *Store) SubmitComposer(ctx context.Context, id, text string, attachments []ComposerAttachment, editSeq float64) error {
	s.mu.Lock()
	client := s.client
	mode := NormalizePromptMode(string(s.promptMode))
	s.mu.Unlock()
	if client == nil || id == "" {
		return ErrNoClient
	}
	if strings.TrimSpace(text) == "" && len(attachments) == 0 {
		return errors.New("message is empty")
	}
	var parts []json.RawMessage
	for _, a := range attachments {
		if a.SessionID != id {
			return errors.New("attachment belongs to a different session; retry upload")
		}
		var value any
		if a.ImageData != "" {
			value = map[string]string{"type": "image", "mediaType": a.MediaType, "data": a.ImageData, "name": a.Name}
		} else if a.Upload.ReceiptId != "" {
			value = map[string]string{"type": "file", "receiptId": a.Upload.ReceiptId}
		} else {
			return errors.New("attachment is not uploaded")
		}
		b, _ := json.Marshal(value)
		parts = append(parts, b)
	}
	line := strings.TrimSpace(text)
	if strings.HasPrefix(line, "/") && !strings.Contains(line, "\n") {
		if len(parts) > 0 {
			var commands api.ListResult3
			if err := client.Call(ctx, "commands/list", map[string]any{"agentId": id}, &commands); err != nil {
				return err
			}
			name := strings.TrimPrefix(strings.Fields(line)[0], "/")
			allowed := false
			for _, command := range commands {
				if command.Name == name && command.Input != nil && command.Input.Attachments != nil && *command.Input.Attachments {
					allowed = true
				}
			}
			if !allowed {
				return fmt.Errorf("/%s does not accept attachments", name)
			}
		}
		var result api.ExecuteResult
		if err := client.Call(ctx, "commands/execute", map[string]any{"agentId": id, "line": line, "submittedAttachments": parts}, &result); err != nil {
			return err
		}
		var outcome struct {
			Kind string `json:"kind"`
			Text string `json:"text"`
		}
		if err := json.Unmarshal(result.Result, &outcome); err != nil {
			return err
		}
		if outcome.Kind == "error" {
			return errors.New(outcome.Text)
		}
		if outcome.Kind != "success" {
			return errors.New("command was not accepted; check its name and retry")
		}
		return nil
	}
	if strings.TrimSpace(text) != "" {
		b, _ := json.Marshal(map[string]string{"type": "text", "text": text})
		parts = append([]json.RawMessage{b}, parts...)
	}
	req := api.SessionPromptRequest{RequestId: newRequestID(), SessionId: id, Mode: string(mode), Content: parts}
	if editSeq > 0 {
		req.EditMessageSeq = &editSeq
	}
	s.mu.Lock()
	if s.selectedID == id {
		s.promptBusy = true
	}
	s.mu.Unlock()
	s.fire()
	var result json.RawMessage
	err := client.Call(ctx, "session/prompt", map[string]any{"request": req}, &result)
	s.mu.Lock()
	if s.selectedID == id {
		s.promptBusy = false
		if err != nil {
			s.lastAction = err.Error()
		} else {
			s.lastAction = "prompt sent"
		}
	}
	s.mu.Unlock()
	s.fire()
	return err
}
