package chat

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/openui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func whaleUIToolView(c *ui.Context, m Model, msg session.Message) bool {
	doc, ok := session.WhaleUIToolDocument(msg)
	if !ok || m.FeedbackStore == nil {
		return false
	}
	msg.SourceMessageID = "tool:" + doc.CallID
	msg.Seq = doc.Seq
	msg.Streaming = false
	return openUIBlockView(c, m, msg, doc.Language, doc.Program, 0, msg.SourceMessageID, false, nil)
}
func whaleThemeVariables(m Model) map[string]string {
	p := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	prefs := theme.CurrentPreferences()
	return map[string]string{"--whale-font": theme.TextFamily(prefs.Chat.Family), "--whale-mono": theme.CodeFamily(prefs.Code.Family), "--whale-font-size": fmt.Sprintf("%gpx", prefs.Chat.Size), "--bg": p.BG, "--background": p.BG, "--surface": p.Field, "--text": p.FG, "--text-secondary": p.Secondary, "--border": p.Border, "--accent": p.Accent, "--dsw-alias-label-primary": p.FG, "--dsw-alias-label-secondary": p.Secondary, "--dsw-alias-label-tertiary": p.Muted, "--dsw-alias-label-primary-foreground": p.ButtonFG, "--dsw-alias-bg-base": p.BG, "--dsw-alias-bg-layer-1": p.Field, "--dsw-alias-border-l2": p.Border, "--dsw-alias-interactive-bg-hover": p.Hover, "--dsw-alias-interactive-bg-active": p.Pressed, "--dsw-alias-button-primary-fill": p.BrandPrimary, "--dsw-alias-button-primary-hover": p.BrandPrimaryHover, "--whale-locale": string(i18n.Locale())}
}
func equalWhaleVariables(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
func whalePreviewVariables(v map[string]string) map[string]string {
	out := map[string]string{}
	for _, key := range []string{"--bg", "--background", "--surface", "--text", "--text-secondary", "--border", "--accent"} {
		if value := v[key]; value != "" {
			out[key] = value
		}
	}
	return out
}
func whaleResolvedCurrent(e openui.Event, program string, revision uint64) bool {
	return (e.Revision == 0 && e.Program == program) || (e.Revision > 0 && e.Revision == revision && (e.Program == "" || e.Program == program))
}
func whaleActionResponse(e openui.Event, err error) openui.APIResponse {
	body := map[string]any{"ok": err == nil}
	status := 200
	if err != nil {
		body["error"] = err.Error()
		status = 409
	}
	raw, _ := json.Marshal(body)
	return openui.APIResponse{InstanceID: e.InstanceID, RequestID: e.RequestID, Status: status, Body: raw}
}

type whaleDownload struct {
	filename, mime     string
	parts, next, bytes int
	content            strings.Builder
}
type whaleDownloads map[string]*whaleDownload

func newWhaleDownloads() whaleDownloads { return make(whaleDownloads) }
func (d whaleDownloads) Add(e openui.Event) error {
	if err := openui.ValidateDownloadPart(e); err != nil {
		return err
	}
	p := d[e.RequestID]
	if p == nil {
		if e.Part != 0 || len(d) != 0 {
			return errors.New("unexpected Whale UI export fragment")
		}
		p = &whaleDownload{filename: e.Filename, mime: e.MIME, parts: e.Parts}
		d[e.RequestID] = p
	}
	if p.next != e.Part || p.parts != e.Parts || p.filename != e.Filename || p.mime != e.MIME || p.bytes+len(e.Content) > openui.MaxDownloadBytes {
		return errors.New("out-of-order or oversized Whale UI export")
	}
	p.content.WriteString(e.Content)
	p.bytes += len(e.Content)
	p.next++
	return nil
}
func (d whaleDownloads) Complete(e *openui.Event) error {
	p := d[e.RequestID]
	delete(d, e.RequestID)
	if p == nil || p.next != p.parts || p.parts != e.Parts || p.filename != e.Filename || p.mime != e.MIME {
		return errors.New("incomplete Whale UI export")
	}
	e.Content = p.content.String()
	return openui.ValidateDownload(*e)
}

// Only the native dialog selects the destination. Cancel and existing files do
// not write; the child filename is a suggestion, never an address or overwrite.
func saveWhaleDownload(parent *mygo.Window, e openui.Event, current func() bool) error {
	bytes, err := openui.DownloadBytes(e)
	if err != nil {
		return err
	}
	if !current() {
		return errors.New("Whale UI export owner changed")
	}
	path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{Parent: parent, Title: whaleLabel("保存交互页面导出", "Save interface export"), DefaultPath: e.Filename, CreateDirectories: true})
	if err != nil {
		return err
	}
	if path == "" {
		return errors.New("The export was cancelled")
	}
	if !current() {
		return errors.New("Whale UI export owner changed")
	}
	return writeWhaleExport(path, bytes)
}
func writeWhaleExport(path string, content []byte) error {
	if !filepath.IsAbs(path) {
		return errors.New("export dialog returned a relative path")
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("save export: %w (choose a new filename to preserve existing files)", err)
	}
	_, writeErr := file.Write(content)
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}
func whaleLabel(zh, en string) string {
	if i18n.Locale() == i18n.LocaleEN {
		return en
	}
	return zh
}

// Kept here to make the queue-mode nature of viewer submission explicit to
// panel captures too; no renderer-supplied Session identifier is accepted.
func captureWhalePanel(m Model, p *whalePanelState) func() (*session.ComposerTarget, func() bool, error) {
	return func() (*session.ComposerTarget, func() bool, error) {
		return m.FeedbackStore.CaptureWhaleUIPanel(m.Snapshot.PlanHistoryBinding, m.Snapshot.SelectedID, func() bool { return p.Current() })
	}
}
