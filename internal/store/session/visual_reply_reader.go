package session

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"whaleisle/internal/dsh"
)

// ReadPublishedVisualReply reads an already-published attachment for an
// independent native owner, such as a Project delivery. The original Host
// reader establishes durable publication authority; no selected Session,
// workspace path, attachment fallback, or generation tool participates.
func ReadPublishedVisualReply(ctx context.Context, client *dsh.Client, sid string, file VisualReplyFile, valid func() bool) (string, error) {
	if client == nil || sid == "" || file.AttachmentID == "" || !strings.HasSuffix(file.Name, ".html") || file.Bytes < 1 || file.Bytes > MaxVisualReplyBytes || valid == nil || !valid() {
		return "", fmt.Errorf("published HTML reference or owner is unavailable")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	var out struct {
		Attachment VisualReplyFile `json:"attachment"`
		HTML       string          `json:"html"`
	}
	err := client.Call(ctx, "session/readVisualReply", map[string]any{"request": map[string]string{"sessionId": sid, "attachmentId": file.AttachmentID}}, &out)
	if err == nil {
		err = ctx.Err()
	}
	if err == nil && (!valid() || out.Attachment != file || len(out.HTML) > MaxVisualReplyBytes || !utf8.ValidString(out.HTML) || int64(len(out.HTML)) != file.Bytes) {
		err = fmt.Errorf("published HTML owner or attachment changed")
	}
	if err != nil {
		return "", err
	}
	return out.HTML, nil
}
