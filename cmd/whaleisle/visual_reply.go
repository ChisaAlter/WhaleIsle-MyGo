package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/store/session"
)

func readPublishedHTML(s *session.Store, sessionID, attachmentID string) {
	if s == nil || s.Snapshot().SelectedID != sessionID {
		return
	}
	// Check on an explicit read, never on every paint. Generation/isolation stays
	// disabled; historical reads do not depend on the generation enabled setting.
	caps := harness.ReadCurrentHostCapabilities(paths.VendorDir(), harness.CurrentHostBridges{})
	if !caps.VisualReplyReader {
		s.SetLastAction("已发布 HTML 读取不可用 / Published HTML unavailable: " + caps.Reason)
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		s.LoadVisualReply(ctx, sessionID, attachmentID)
	}()
}

func savePublishedHTML(win *mygo.Window, s *session.Store, sessionID, attachmentID string) {
	if s == nil {
		return
	}
	page := s.VisualReplySnapshot()
	if !page.Ready || page.SessionID != sessionID || page.AttachmentID != attachmentID {
		return
	}
	name := filepath.Base(strings.ReplaceAll(page.Name, `\`, "/"))
	if name == "." || name == "" || !strings.HasSuffix(name, ".html") {
		name = "published-page.html"
	}
	target, err := mygo.Dialog.Save(mygo.SaveDialogOptions{Parent: win, Title: "另存已发布 HTML / Save published HTML", DefaultPath: name})
	if err != nil {
		mygo.Dialog.Error("Save HTML", err.Error())
		return
	}
	if target == "" {
		return
	}
	// Dismissal or navigation while the dialog was open revokes this pending save.
	current := s.VisualReplySnapshot()
	if !current.Ready || current.SessionID != sessionID || current.AttachmentID != attachmentID || current.HTML != page.HTML {
		return
	}
	if err = os.WriteFile(target, []byte(page.HTML), 0600); err != nil {
		mygo.Dialog.Error("Save HTML", err.Error())
	}
}
