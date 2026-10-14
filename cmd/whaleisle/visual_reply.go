package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/htmlpreview"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/store/session"
)

func readPublishedHTML(s *session.Store, sessionID, attachmentID string) {
	if s == nil || s.Snapshot().SelectedID != sessionID {
		return
	}
	// Check on an explicit read, never on every paint. Source reading executes no
	// HTML and does not depend on the generation enabled setting.
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

// Each explicit click reuses the durable authorized read; generation preference
// is irrelevant to reading an already-published page.
func previewPublishedHTML(s *session.Store, sessionID, attachmentID string) {
	if s == nil || s.Snapshot().SelectedID != sessionID {
		return
	}
	caps := harness.ReadCurrentHostCapabilities(paths.VendorDir(), harness.CurrentHostBridges{})
	if !caps.VisualReplyReader {
		s.SetLastAction("Published HTML unavailable: " + caps.Reason)
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		// Never fetch an arbitrary attachment or use a previously switched session.
		s.LoadVisualReply(ctx, sessionID, attachmentID)
		s.PreviewVisualReply(ctx, sessionID, attachmentID, func(ctx context.Context, html string) (session.VisualReplyCapture, error) {
			if !htmlpreview.Available() {
				status := htmlpreview.Probe(ctx)
				if !status.Available {
					return session.VisualReplyCapture{}, fmt.Errorf("%s", status.Reason)
				}
			}
			result, err := htmlpreview.RenderSaved(ctx, html, 728, "dark")
			return session.VisualReplyCapture{PNG: result.PNG, Width: result.Width, ContentHeight: result.ContentHeight, CapturedHeight: result.CapturedHeight}, err
		})
	}()
}

// Used only on the GUI thread. One bitmap is retained, cleared immediately when
// the selected Reader state is closed/replaced, and never written to disk.
var publishedHTMLImage struct {
	id     uint64
	bitmap *ui.Bitmap
}

func publishedHTMLBitmap(page session.VisualReplyState) *ui.Bitmap {
	if page.PreviewID == 0 {
		publishedHTMLImage.id = 0
		publishedHTMLImage.bitmap = nil
		return nil
	}
	if publishedHTMLImage.id != page.PreviewID {
		bitmap, err := ui.DecodeBitmap(page.PreviewPNG)
		if err != nil {
			return nil
		}
		publishedHTMLImage.id = page.PreviewID
		publishedHTMLImage.bitmap = bitmap
	}
	return publishedHTMLImage.bitmap
}
