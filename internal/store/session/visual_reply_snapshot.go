package session

import (
	"context"
	"fmt"
	"unicode/utf8"
)

// ReadVisualReplySnapshot authorizes one independent inline publication without
// replacing the source reader or any sibling publication's live context.
func (s *Store) ReadVisualReplySnapshot(ctx context.Context, sid, id string) (string, func() bool, error) {
	s.mu.Lock()
	client, generation := s.client, s.followGeneration
	var pub VisualReplyPublication
	found := false
	if client != nil && sid != "" && s.selectedID == sid {
		for _, m := range s.messages {
			if p, ok := PublishedVisualReply(m.Raw); ok && p.File.AttachmentID == id {
				pub = p
				found = true
				break
			}
		}
	}
	s.mu.Unlock()
	if !found {
		return "", nil, fmt.Errorf("publication does not belong to the selected session")
	}
	valid := func() bool {
		s.mu.Lock()
		defer s.mu.Unlock()
		return s.client == client && s.followGeneration == generation && s.selectedID == sid
	}
	var out struct {
		Attachment VisualReplyFile `json:"attachment"`
		HTML       string          `json:"html"`
	}
	err := client.Call(ctx, "session/readVisualReply", map[string]any{"request": map[string]string{"sessionId": sid, "attachmentId": id}}, &out)
	if err == nil {
		err = ctx.Err()
	}
	if err == nil && (!valid() || out.Attachment != pub.File || len(out.HTML) > MaxVisualReplyBytes || !utf8.ValidString(out.HTML) || int64(len(out.HTML)) != out.Attachment.Bytes) {
		err = fmt.Errorf("published HTML owner or attachment changed")
	}
	if err != nil {
		return "", nil, err
	}
	return out.HTML, valid, nil
}

type InlineVisualReply struct {
	Surface        VisualReplySurface
	Height         int
	Expanded       bool
	Loading        bool
	Error          string
	ToggleExpanded func()
	Retry          func()
}
