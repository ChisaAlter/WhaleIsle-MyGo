package composer

import (
	"fmt"
	"path/filepath"
	"strings"

	"whaleisle/internal/desktop/workspacefs"
)

// InsertImageMention appends a markdown image reference for absPath under root.
func InsertImageMention(draft *string, root, absPath string) error {
	rel, err := workspacefs.RelPath(root, absPath)
	if err != nil {
		return err
	}
	if rel == "" {
		return fmt.Errorf("empty path")
	}
	// Prefer forward slashes in the draft for markdown/portability.
	rel = filepath.ToSlash(rel)
	return appendDraftToken(draft, "![]("+rel+") ")
}

// InsertAttachmentImageMention appends a markdown image for a fileUploads/upload id.
func InsertAttachmentImageMention(draft *string, attachmentID string) error {
	attachmentID = strings.TrimSpace(attachmentID)
	if attachmentID == "" {
		return fmt.Errorf("empty attachment id")
	}
	return appendDraftToken(draft, "![](attachment:"+attachmentID+") ")
}

func appendDraftToken(draft *string, token string) error {
	if draft == nil {
		return nil
	}
	cur := strings.TrimRight(*draft, " \t")
	if cur == "" {
		*draft = token
		return nil
	}
	*draft = cur + " " + token
	return nil
}
