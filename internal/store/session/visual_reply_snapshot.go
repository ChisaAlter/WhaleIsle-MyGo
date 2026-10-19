package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"unicode/utf8"

	"whaleisle/internal/dsh"
)

// Invalid published bytes or a reference absent from the selected transcript
// cannot be repaired by rereading the same publication. Connection/owner
// changes remain ordinary transient errors, since a new mount may read again.
type visualReplyContentError struct{ reason string }

func (e *visualReplyContentError) Error() string { return "published HTML unavailable: " + e.reason }

// VisualReplyRetryable matches PublishedPage's typed Host failure contract.
// Unknown failures, including transport/helper failures, retain manual retry.
func VisualReplyRetryable(err error) bool {
	if err == nil {
		return false
	}
	var content *visualReplyContentError
	if errors.As(err, &content) {
		return false
	}
	var remote *dsh.Error
	if !errors.As(err, &remote) {
		return true
	}
	if remote.Code == "session/not-found" {
		return false
	}
	if remote.Code == "session/attachment-invalid" {
		var details struct {
			Reason string `json:"reason"`
		}
		if json.Unmarshal(remote.Details, &details) == nil {
			switch details.Reason {
			case "ATTACHMENT_NOT_REFERENCED", "VISUAL_REPLY_TOO_LARGE", "VISUAL_REPLY_INVALID_UTF8",
				"ATTACHMENT_NOT_FOUND", "ATTACHMENT_CORRUPT", "INVALID_ATTACHMENT_REF":
				return false
			}
		}
	}
	return true
}

func validateVisualReplySnapshot(expected, actual VisualReplyFile, html string) error {
	switch {
	case expected != actual:
		return &visualReplyContentError{reason: "attachment reference changed"}
	case len(html) > MaxVisualReplyBytes:
		return &visualReplyContentError{reason: "HTML exceeds the published size limit"}
	case !utf8.ValidString(html):
		return &visualReplyContentError{reason: "HTML is not valid UTF-8"}
	case int64(len(html)) != actual.Bytes:
		return &visualReplyContentError{reason: "HTML byte count does not match the attachment"}
	}
	return nil
}

// ReadVisualReplySnapshot authorizes one independent inline publication without
// replacing the source reader or any sibling publication's live context.
func (s *Store) ReadVisualReplySnapshot(ctx context.Context, sid, id string) (string, func() bool, error) {
	s.mu.Lock()
	client, generation := s.client, s.followGeneration
	var pub VisualReplyPublication
	found := false
	authorized := client != nil && sid != "" && s.selectedID == sid
	if authorized {
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
		if !authorized {
			return "", nil, fmt.Errorf("published HTML session is unavailable or changed")
		}
		return "", nil, &visualReplyContentError{reason: "publication does not belong to the selected transcript"}
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
	if err == nil && !valid() {
		err = fmt.Errorf("published HTML session changed")
	}
	if err == nil {
		err = validateVisualReplySnapshot(pub.File, out.Attachment, out.HTML)
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
