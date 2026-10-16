package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/htmlpreview"
	"whaleisle/internal/desktop/previewprocess"
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
			result, err := captureHTMLInHelper(ctx, html, 728, "dark")
			if err != nil {
				return session.VisualReplyCapture{}, err
			}
			png, err := base64.StdEncoding.DecodeString(result.PNG)
			return session.VisualReplyCapture{PNG: png, Width: result.Width, ContentHeight: result.ContentHeight, CapturedHeight: result.CapturedHeight}, err
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

// This callback is offered only on the platform with the existing AppContainer
// helper. An explicit gesture mounts its child surface in the selected card.
func interactivePublishedHTMLHandler(win *mygo.Window, s *session.Store, sessionID string, dark bool) func(string) {
	if runtime.GOOS != "windows" || (runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64") {
		return nil
	}
	return func(attachmentID string) {
		if s == nil || s.Snapshot().SelectedID != sessionID {
			return
		}
		caps := harness.ReadCurrentHostCapabilities(paths.VendorDir(), harness.CurrentHostBridges{})
		if !caps.VisualReplyReader {
			s.SetLastAction("Published HTML unavailable: " + caps.Reason)
			return
		}
		appearance := "light"
		if dark {
			appearance = "dark"
		}
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
			defer cancel()
			s.OpenInteractiveVisualReply(ctx, sessionID, attachmentID, func(ctx context.Context, html string) (session.VisualReplyInteractive, error) {
				executable, err := os.Executable()
				if err != nil {
					return nil, err
				}
				helper := previewprocess.HelperPath(executable)
				info, err := os.Stat(helper)
				if err != nil || !info.Mode().IsRegular() {
					return nil, fmt.Errorf("隔离 helper 未随应用安装 / Isolated preview helper is not installed")
				}
				// readVisualReply returns the already-prepared attachment: local
				// images were embedded by the original publication. Never resolve
				// workspace/file/remote references or pass Host credentials here.
				handle, err := previewprocess.OpenExperimental(ctx, previewprocess.Options{Enabled: true, HelperPath: helper},
					previewprocess.Document{HTML: html, Width: 900, Appearance: appearance, Inline: true})
				if err != nil {
					return nil, err
				}
				return newPublishedHTMLSurface(handle, win), nil
			})
		}()
	}
}

// Shared by the authenticated capture endpoint and an authorized saved reply.
// The helper receives bytes only, never Host credentials or workspace paths.
func captureHTMLInHelper(ctx context.Context, html string, width int, appearance string) (htmlpreview.CaptureResult, error) {
	executable, err := os.Executable()
	if err != nil {
		return htmlpreview.CaptureResult{}, err
	}
	helper := previewprocess.HelperPath(executable)
	return previewprocess.CaptureExperimental(ctx, previewprocess.Options{Enabled: true, HelperPath: helper}, previewprocess.Document{HTML: html, Width: width, Appearance: appearance})
}
