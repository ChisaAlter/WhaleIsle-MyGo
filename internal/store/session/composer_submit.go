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
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

// ComposerAttachment retains its staged receipt and exact session/Host owner.
type ComposerAttachment struct {
	PreparationID                    uint64 `json:"-"`
	PrepareError                     string // Runtime-only failure; retry keeps the local file and draft owner.
	Path, Name, SessionID, HostScope string
	Upload                           api.FileUploadValue
	ImageData, MediaType             string
}

// ComposerTarget pins every step of one admission to one Host and mode. A
// navigation may change the selected session, but never retarget this request.
type ComposerTarget struct {
	draftPresetID, draftPresetCwd string
	SessionID                     string
	store                         *Store
	client                        *dsh.Client
	epoch, followGeneration       uint64
	mode                          PromptMode
}

func (s *Store) CaptureComposerTarget(id string) (*ComposerTarget, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.client == nil {
		return nil, ErrNoClient
	}
	return &ComposerTarget{SessionID: id, store: s, client: s.client, epoch: s.composerHostGeneration, followGeneration: s.followGeneration, mode: NormalizePromptMode(string(s.promptMode)), draftPresetID: s.stagedPresetLocked(s.cwd), draftPresetCwd: filepathClean(s.cwd)}, nil
}
func (t *ComposerTarget) currentLocked() bool {
	return t.store.client == t.client && t.store.composerHostGeneration == t.epoch
}
func (t *ComposerTarget) Current() bool {
	t.store.mu.Lock()
	defer t.store.mu.Unlock()
	return t.currentLocked()
}
func (t *ComposerTarget) scope() string { return fmt.Sprintf("%p:%d", t.client, t.epoch) }
func (t *ComposerTarget) check(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !t.Current() {
		return errors.New("Host changed; draft retained, retry on the current Host")
	}
	return nil
}

func (s *Store) PrepareComposerAttachment(ctx context.Context, id string, a ComposerAttachment) (ComposerAttachment, error) {
	t, err := s.CaptureComposerTarget(id)
	if err != nil {
		return a, err
	}
	return t.PrepareAttachment(ctx, a)
}
func (t *ComposerTarget) PrepareAttachment(ctx context.Context, a ComposerAttachment) (ComposerAttachment, error) {
	if a.PrepareError != "" {
		return a, errors.New("retry or remove the failed attachment before sending")
	}
	if err := t.check(ctx); err != nil {
		return a, err
	}
	if t.SessionID == "" {
		return a, ErrNoClient
	}
	if a.SessionID == t.SessionID && a.HostScope == t.scope() && (a.Upload.ReceiptId != "" || a.ImageData != "") {
		return a, nil
	}
	// Never reuse the prior Host's receipt, even when session IDs are equal.
	a.Upload = api.FileUploadValue{}
	a.ImageData = ""
	a.SessionID = ""
	a.HostScope = ""
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
		a.Upload, err = t.uploadFile(ctx, a.Path)
		if err != nil {
			return a, err
		}
	}
	if err = t.check(ctx); err != nil {
		return a, err
	}
	a.SessionID, a.HostScope = t.SessionID, t.scope()
	return a, nil
}
func (s *Store) UploadFileForSession(ctx context.Context, id, path string) (api.FileUploadValue, error) {
	t, err := s.CaptureComposerTarget(id)
	if err != nil {
		return api.FileUploadValue{}, err
	}
	return t.uploadFile(ctx, path)
}
func (t *ComposerTarget) uploadFile(ctx context.Context, path string) (api.FileUploadValue, error) {
	var out api.FileUploadValue
	if err := t.check(ctx); err != nil {
		return out, err
	}
	if t.SessionID == "" {
		return out, ErrNoClient
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return out, err
	}
	name := filepath.Base(path)
	err = t.client.Call(ctx, "fileUploads/upload", map[string]any{"agentId": t.SessionID, "request": api.EncodedFileUploadRequest{Data: base64.StdEncoding.EncodeToString(raw), Name: &name}}, &out)
	if err != nil {
		return out, err
	}
	if err = t.check(ctx); err != nil {
		return api.FileUploadValue{}, err
	}
	if strings.TrimSpace(out.ReceiptId) == "" || strings.TrimSpace(out.File.AttachmentId) == "" {
		return out, errors.New("upload returned an incomplete file receipt")
	}
	return out, nil
}
func (s *Store) CreateComposerSession(ctx context.Context, cwd string) (string, error) {
	t, err := s.CaptureComposerTarget("")
	if err != nil {
		return "", err
	}
	return t.CreateSession(ctx, cwd)
}
func (t *ComposerTarget) CreateSession(ctx context.Context, cwd string) (string, error) {
	if err := t.check(ctx); err != nil {
		return "", err
	}
	if strings.TrimSpace(cwd) == "" {
		return "", errors.New("choose a workspace before sending")
	}
	request := api.SessionCreateRequest{Cwd: &cwd}
	if t.draftPresetCwd == filepathClean(cwd) && t.draftPresetID != "" {
		request.AgentPreset = &t.draftPresetID
	}
	var out api.SessionCreateValue
	if err := t.client.Call(ctx, "session/create", map[string]any{"request": request}, &out); err != nil {
		return "", err
	}
	if err := t.check(ctx); err != nil {
		return "", err
	}
	if out.SessionId == "" {
		return "", errors.New("session/create returned no session")
	}
	t.store.mu.Lock()
	if request.AgentPreset != nil && t.store.stagedPresetLocked(cwd) == t.draftPresetID {
		delete(t.store.presetDraft.staged, filepathClean(cwd))
	}
	t.store.mu.Unlock()
	t.SessionID = out.SessionId
	return out.SessionId, nil
}

// SubmitComposer keeps the synchronous adapter for existing callers. UI sends
// capture their target before detaching, then submit with their echo request ID.
func (s *Store) SubmitComposer(ctx context.Context, id, text string, attachments []ComposerAttachment, editSeq float64) error {
	t, err := s.CaptureComposerTarget(id)
	if err != nil {
		return err
	}
	return t.Submit(ctx, text, attachments, editSeq, "")
}
func (t *ComposerTarget) Submit(ctx context.Context, text string, attachments []ComposerAttachment, editSeq float64, requestID string) (err error) {
	s, id := t.store, t.SessionID
	if requestID == "" {
		requestID = newRequestID()
	}
	defer func() { err = t.finishSubmission(requestID, err) }()
	if err = t.check(ctx); err != nil {
		return err
	}
	if id == "" {
		return ErrNoClient
	}
	if strings.TrimSpace(text) == "" && len(attachments) == 0 {
		return errors.New("message is empty")
	}
	parts := make([]json.RawMessage, 0, len(attachments)+1)
	for _, a := range attachments {
		if a.PrepareError != "" {
			return errors.New("retry or remove the failed attachment before sending")
		}
		if a.SessionID != id || a.HostScope != t.scope() {
			return errors.New("attachment belongs to another session or Host; retry upload")
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
	if editSeq == 0 && strings.HasPrefix(line, "/") {
		var commands api.ListResult3
		if err = t.client.Call(ctx, "commands/list", map[string]any{"agentId": id}, &commands); err != nil {
			return err
		}
		if err = t.check(ctx); err != nil {
			return err
		}
		if command, ok := commandForLine(commands, line); ok {
			// The original command adjudication bypasses the prompt sink. Retire
			// the editor's provisional prompt echo before taking that branch;
			// failures still return through finishSubmission for draft recovery.
			t.retireCommandSubmission(requestID)
			if len(parts) > 0 && (command.Input == nil || command.Input.Attachments == nil || !*command.Input.Attachments) {
				return fmt.Errorf("/%s does not accept attachments", command.Name)
			}
			err = executeCommand(ctx, t.client, id, line, parts)
			if err != nil {
				return err
			}
			return t.check(ctx)
		}
	}
	if strings.TrimSpace(text) != "" {
		b, _ := json.Marshal(map[string]string{"type": "text", "text": text})
		parts = append([]json.RawMessage{b}, parts...)
	}
	req := api.SessionPromptRequest{RequestId: requestID, SessionId: id, Mode: string(t.mode), Content: parts}
	if editSeq > 0 {
		req.EditMessageSeq = &editSeq
	}
	s.mu.Lock()
	if !t.currentLocked() {
		s.mu.Unlock()
		return errors.New("Host changed; draft retained")
	}
	if editSeq > 0 && (s.selectedID != id || !CanEditMessage(ViewState{SelectedID: id, Messages: projectMessageEdits(s.messages), PromptBusy: s.promptBusy || s.composerPromptBusyLocked() || s.composerPendingForSelectedLocked() || s.turnStartMs > 0}, editSeq)) {
		s.mu.Unlock()
		return errors.New("the edited message is no longer the latest idle user turn")
	}
	if s.composerAdmissions == nil {
		s.composerAdmissions = make(map[string]*ComposerTarget)
	}
	s.composerAdmissions[requestID] = t
	s.markPromptAttemptedLocked(id)
	s.mu.Unlock()
	s.fire()
	var result struct {
		Accepted bool `json:"accepted"`
	}
	err = t.client.Call(ctx, "session/prompt", map[string]any{"request": req}, &result)
	if err == nil && !result.Accepted {
		err = errors.New("prompt was not accepted; draft retained")
	}
	return err
}
