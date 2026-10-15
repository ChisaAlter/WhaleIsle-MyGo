package chat

import (
	"strings"
	"testing"
	"whaleisle/internal/store/session"
)

func TestQueueAttachmentLabels(t *testing.T) {
	file := queueAttachmentLabel(session.QueueAttachment{Kind: "file", Name: "report.pdf", Bytes: 2048})
	if file != "report.pdf · 2.0 KB" {
		t.Fatal(file)
	}
	image := queueAttachmentLabel(session.QueueAttachment{Kind: "image", AttachmentID: "image-id", Width: 20, Height: 10, Bytes: 42})
	if !strings.Contains(image, "image-id") || !strings.Contains(image, "20×10") {
		t.Fatal(image)
	}
}
