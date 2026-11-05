package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginui"
	"whaleisle/internal/harness"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/composer"
)

type nativePluginComposerBridge struct{ state *mainState }
type nativePluginComposerOwner struct {
	binding     string
	subject     pluginui.Subject
	draft       *composer.DraftState
	sessions    *session.Store
	target      *session.ComposerTarget
	host        harness.Status
	attachments string
}

func pluginComposerStale() error {
	return errors.New(resumeText("输入框已更改，请重新读取后重试。", "The composer changed. Read it again before retrying."))
}

// RunOnMain executes immediately on the UI thread and synchronously dispatches
// socket calls. In particular, it never posts Update and then waits on that same
// UI thread. A shut-down application leaves the explicit refusal unchanged.
func (b nativePluginComposerBridge) onUI(fn func() error) error {
	err := pluginComposerStale()
	mygo.RunOnMain(func() { err = fn() })
	return err
}

func (b nativePluginComposerBridge) capture(subject pluginui.Subject) (*nativePluginComposerOwner, error) {
	s := b.state
	if s == nil {
		return nil, pluginComposerStale()
	}
	s.mu.Lock()
	p, sessions, host := s.nativePlugins.session, s.sessions, s.bootStatus
	valid := p != nil && p.visible && p.subject == subject && p.sessions == sessions && !s.quitting && host.State == harness.Ready && p.endpoint == host.Endpoint && p.generation == host.Generation
	s.mu.Unlock()
	if !valid || sessions == nil || subject.SessionID == "" || pluginSessionStoreSubject(sessions) != subject {
		return nil, pluginComposerStale()
	}
	target, err := sessions.CaptureComposerTarget(subject.SessionID)
	if err != nil || !target.NavigationCurrent() {
		return nil, pluginComposerStale()
	}
	s.mu.Lock()
	d := s.composerDraftLocked(subject.SessionID)
	type fileIdentity struct {
		ID                 uint64
		Path, Media, Error string
	}
	identity := make([]fileIdentity, 0, len(d.Attachments))
	for _, a := range d.Attachments {
		identity = append(identity, fileIdentity{a.PreparationID, a.Path, a.MediaType, a.PrepareError})
	}
	attachments, _ := json.Marshal(identity)
	o := s.pluginComposerOwner
	if o == nil || o.draft != d || o.sessions != sessions || o.subject != subject || o.host.Generation != host.Generation || o.host.Endpoint != host.Endpoint || o.attachments != string(attachments) || !o.target.NavigationCurrent() {
		var key [16]byte
		if _, err := rand.Read(key[:]); err != nil {
			s.mu.Unlock()
			return nil, err
		}
		o = &nativePluginComposerOwner{binding: hex.EncodeToString(key[:]), subject: subject, draft: d, sessions: sessions, target: target, host: host, attachments: string(attachments)}
		s.pluginComposerOwner = o
	}
	s.mu.Unlock()
	return o, nil
}
func composerPluginBinding(o *nativePluginComposerOwner) pluginui.ComposerBinding {
	d := o.draft
	return pluginui.ComposerBinding{Subject: o.subject, Binding: o.binding, Revision: d.Client.Revision(), TargetID: d.TargetID, EditSeq: d.EditSeq}
}
func (b nativePluginComposerBridge) validate(owner pluginui.ComposerBinding) (*nativePluginComposerOwner, error) {
	o, err := b.capture(owner.Subject)
	if err != nil || composerPluginBinding(o) != owner {
		return nil, pluginComposerStale()
	}
	if o.draft.Sending || o.draft.Preparing > 0 {
		return nil, errors.New(resumeText("输入框正在发送或准备附件。", "The composer is sending or preparing attachments."))
	}
	return o, nil
}
func (b nativePluginComposerBridge) Read(subject pluginui.Subject) (out pluginui.ComposerDraft, err error) {
	err = b.onUI(func() error {
		o, err := b.capture(subject)
		if err != nil {
			return err
		}
		out = pluginComposerSnapshot(o)
		return nil
	})
	return
}

func pluginComposerSnapshot(o *nativePluginComposerOwner) pluginui.ComposerDraft {
	d := o.draft
	selection := d.Client.Selection().Range
	out := pluginui.ComposerDraft{Owner: composerPluginBinding(o), Text: d.Text, Sending: d.Sending, Preparing: d.Preparing > 0, SelectionStart: selection.Start, SelectionEnd: selection.End}
	for i, a := range d.Attachments {
		name := a.Name
		if name == "" {
			name = filepath.Base(a.Path)
		}
		message := a.PrepareError
		if a.Path != "" {
			message = strings.ReplaceAll(message, a.Path, name)
		}
		out.Attachments = append(out.Attachments, pluginui.ComposerFile{ID: a.PreparationID, Name: name, Image: strings.HasPrefix(a.MediaType, "image/"), Error: message, Size: a.TotalBytes, MediaType: composerPublicMediaType(a), Loaded: a.SentBytes, Preparing: d.AttachmentPreparing(i)})
	}
	return out
}
func (b nativePluginComposerBridge) Apply(edit pluginui.ComposerEdit) error {
	return b.onUI(func() error {
		return b.applyOnUI(edit, b.composerPost())
	})
}

// The caller owns the UI thread; post dispatches only asynchronous attachment
// progress/completion back to that same owner.
func (b nativePluginComposerBridge) applyOnUI(edit pluginui.ComposerEdit, post func(func())) error {
	o, err := b.validate(edit.Owner)
	if err != nil {
		return err
	}
	d, s := o.draft, b.state
	actions := 0
	if edit.Text != nil {
		actions++
	}
	if edit.Replace != nil {
		actions++
	}
	if edit.RemoveFileID != 0 {
		actions++
	}
	if edit.RetryFileID != 0 {
		actions++
	}
	if edit.PickFiles {
		actions++
	}
	if edit.PickImage {
		actions++
	}
	if actions != 1 {
		return errors.New(resumeText("一次只能提交一个输入框动作。", "Submit one composer action at a time."))
	}
	if edit.Text != nil {
		r := ui.TextInputRange{End: ui.UTF16Len(d.Text)}
		d.Client.ReplaceText(&r, *edit.Text)
	}
	if edit.Replace != nil {
		r := edit.Replace
		if r.Start < 0 || r.End < r.Start || r.End > ui.UTF16Len(d.Text) {
			return pluginComposerStale()
		}
		d.Client.ReplaceText(&ui.TextInputRange{Start: r.Start, End: r.End}, r.Text)
	}
	if edit.RemoveFileID != 0 {
		found := false
		for i, a := range d.Attachments {
			if a.PreparationID == edit.RemoveFileID {
				found = d.RemoveAttachment(i)
				break
			}
		}
		if !found {
			return pluginComposerStale()
		}
	}
	if edit.RetryFileID != 0 {
		found := false
		for i, a := range d.Attachments {
			if a.PreparationID == edit.RetryFileID {
				found = prepareComposerAttachment(o.target, d, i, true, post)
				break
			}
		}
		if !found {
			return pluginComposerStale()
		}
	}
	if edit.PickFiles || edit.PickImage {
		s.mu.Lock()
		win := s.win
		s.mu.Unlock()
		before := composerPluginBinding(o)
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: win, Title: resumeText("添加文件或图片", "Add files or images"), Multiple: true, DefaultPath: o.sessions.SelectionSummary().SelectedCwd})
		if err != nil {
			return err
		}
		if _, err := b.validate(before); err != nil {
			return err
		}
		s.addComposerFiles(win, o.sessions, o.subject.SessionID, d, paths)
	}
	s.mu.Lock()
	win := s.win
	s.mu.Unlock()
	if win != nil {
		win.Invalidate()
	}
	return nil
}
func (b nativePluginComposerBridge) Submit(gesture pluginui.ComposerGesture) error {
	return b.onUI(func() error {
		o, err := b.validate(gesture.Owner)
		if err != nil {
			return err
		}
		s := b.state
		s.mu.Lock()
		win := s.win
		s.mu.Unlock()
		// submitComposer captures CaptureComposerGestureTarget, including the real
		// busy-enter policy, instead of inventing a plugin prompt transport.
		v := o.sessions.Snapshot()
		s.submitComposer(win, o.sessions, o.draft, o.subject.SessionID, v.SelectedCwd, o.draft.EditSeq, gesture.Accelerated, v.ComposerHostGeneration)
		if o.draft.Error != "" {
			return errors.New(o.draft.Error)
		}
		return nil
	})
}
