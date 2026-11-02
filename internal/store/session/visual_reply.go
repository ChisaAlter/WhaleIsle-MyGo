package session

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"
	"whaleisle/internal/store/storelocale"
)

const MaxVisualReplyBytes = 25 * 1024 * 1024
const DefaultVisualReplyHeight = 480

type VisualReplyFile struct {
	AttachmentID string `json:"attachmentId"`
	Name         string `json:"name"`
	Bytes        int64  `json:"bytes"`
}

type VisualReplyPublication struct {
	File   VisualReplyFile
	Title  string
	Height int
}

// PublishedVisualReply follows the native and PTC publication contracts. A file
// link or a tool name alone is never publication authority.
func PublishedVisualReply(raw json.RawMessage) (VisualReplyPublication, bool) {
	if !bytes.Contains(raw, []byte("visualReply")) && !bytes.Contains(raw, []byte("html_render")) {
		return VisualReplyPublication{}, false
	}
	var e struct {
		Type      string `json:"type"`
		SurfaceOp string `json:"surfaceOp"`
		Data      struct {
			Name      string          `json:"name"`
			IsError   bool            `json:"isError"`
			Content   json.RawMessage `json:"content"`
			Arguments json.RawMessage `json:"arguments"`
			Message   struct {
				IsError bool            `json:"isError"`
				Content json.RawMessage `json:"content"`
			} `json:"message"`
			Meta struct {
				VisualReply json.RawMessage `json:"visualReply"`
			} `json:"meta"`
		} `json:"data"`
	}
	var p VisualReplyPublication
	if json.Unmarshal(raw, &e) != nil {
		return p, false
	}
	var content, presentation json.RawMessage
	switch {
	case e.Type == "tool/result" && e.SurfaceOp == "append" && !e.Data.Message.IsError:
		content, presentation = e.Data.Message.Content, e.Data.Meta.VisualReply
		var version struct {
			Version int `json:"version"`
		}
		if json.Unmarshal(presentation, &version) != nil || version.Version != 1 {
			return p, false
		}
	case e.Type == "tool/ptc-dispatch" && e.Data.Name == "html_render" && !e.Data.IsError:
		content, presentation = e.Data.Content, e.Data.Arguments
	default:
		return p, false
	}
	var meta struct {
		Title  string          `json:"title"`
		Height json.RawMessage `json:"height"`
	}
	if json.Unmarshal(presentation, &meta) != nil || strings.TrimSpace(meta.Title) == "" {
		return p, false
	}
	p.Height = DefaultVisualReplyHeight
	if len(meta.Height) != 0 && string(meta.Height) != "null" {
		var height float64
		if json.Unmarshal(meta.Height, &height) != nil || height < 80 || height > 2000 {
			return p, false
		}
		p.Height = int(math.Ceil(height))
	}
	var parts []struct {
		Type       string          `json:"type"`
		Attachment VisualReplyFile `json:"attachment"`
	}
	if json.Unmarshal(content, &parts) != nil {
		return p, false
	}
	files := 0
	for _, part := range parts {
		switch part.Type {
		case "text":
		case "file":
			files++
			p.File = part.Attachment
		default:
			return VisualReplyPublication{}, false
		}
	}
	if files != 1 || p.File.AttachmentID == "" || !strings.HasSuffix(p.File.Name, ".html") || p.File.Bytes <= 0 {
		return VisualReplyPublication{}, false
	}
	p.Title = meta.Title
	return p, true
}

type VisualReplyState struct {
	Surface                                                   VisualReplySurface
	InteractiveExpanded                                       bool
	InteractiveLoading                                        bool
	InteractiveActive                                         bool
	InteractiveError                                          string
	PreviewLoading                                            bool
	PreviewError                                              string
	PreviewPNG                                                []byte
	PreviewID                                                 uint64
	PreviewWidth, PreviewContentHeight, PreviewCapturedHeight int
	SessionID                                                 string
	AttachmentID                                              string
	Name                                                      string
	HTML                                                      string
	Loading                                                   bool
	Ready                                                     bool
	Error                                                     string
}

type visualReplyRead struct {
	interaction   *visualReplyInteraction
	previewCancel context.CancelFunc
	VisualReplyState
	generation uint64
	token      uint64
}

// VisualReplySnapshot never exposes a previous session or previous connection's
// HTML. It is memory-only; reading neither writes files nor runs HTML.
func (s *Store) VisualReplySnapshot() VisualReplyState {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.visualReply.SessionID != s.selectedID || s.visualReply.generation != s.followGeneration || s.client == nil {
		return VisualReplyState{}
	}
	return s.visualReply.VisualReplyState
}

func (s *Store) CloseVisualReply() {
	s.mu.Lock()
	s.cancelVisualReplyLocked()
	s.mu.Unlock()
	s.fire()
}

// LoadVisualReply only accepts a published reference in the addressed transcript.
// Host independently authorizes it against durable session events. Never fall
// back to attachment/image APIs, workspace files, URLs, or generation tools.
func (s *Store) LoadVisualReply(ctx context.Context, sessionID, attachmentID string) {
	s.loadVisualReply(ctx, sessionID, attachmentID)
}

// The read token binds an interactive launch to this exact authorized read,
// including repeated requests for the same attachment. Zero means no authority.
func (s *Store) loadVisualReply(ctx context.Context, sessionID, attachmentID string) uint64 {
	s.mu.Lock()
	if s.client == nil || sessionID == "" || s.selectedID != sessionID || ctx.Err() != nil {
		s.mu.Unlock()
		return 0
	}
	var pub VisualReplyPublication
	found := false
	for _, m := range s.messages {
		if p, ok := PublishedVisualReply(m.Raw); ok && p.File.AttachmentID == attachmentID {
			pub, found = p, true
			break
		}
	}
	if !found {
		s.mu.Unlock()
		return 0
	}
	client, generation := s.client, s.followGeneration
	s.cancelVisualReplyLocked()
	s.visualReplyToken++
	token := s.visualReplyToken
	s.visualReply = visualReplyRead{VisualReplyState: VisualReplyState{SessionID: sessionID, AttachmentID: attachmentID, Name: pub.File.Name, Loading: true}, generation: generation, token: token}
	s.mu.Unlock()
	s.fire()
	var out struct {
		Attachment VisualReplyFile `json:"attachment"`
		HTML       string          `json:"html"`
	}
	err := client.Call(ctx, "session/readVisualReply", map[string]any{"request": map[string]string{"sessionId": sessionID, "attachmentId": attachmentID}}, &out)
	if err == nil && (out.Attachment != pub.File || len(out.HTML) > MaxVisualReplyBytes || !utf8.ValidString(out.HTML) || int64(len(out.HTML)) != out.Attachment.Bytes) {
		err = fmt.Errorf("Host returned an invalid or mismatched published HTML attachment")
	}
	s.mu.Lock()
	if s.client != client || s.selectedID != sessionID || s.followGeneration != generation || s.visualReply.token != token || s.visualReplyToken != token {
		s.mu.Unlock()
		return 0
	}
	s.visualReply.Loading = false
	if err != nil {
		s.visualReply.Error = storelocale.T(storelocale.KeyStoreSessionPublishedHtmlUnavailable) + err.Error()
	} else {
		s.visualReply.HTML = out.HTML
		s.visualReply.Ready = true
	}
	s.mu.Unlock()
	s.fire()
	if err != nil {
		return 0
	}
	return token
}
