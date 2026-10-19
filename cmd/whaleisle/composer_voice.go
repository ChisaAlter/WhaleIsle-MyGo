package main

import (
	"context"
	"github.com/egoist/mygo"
	"whaleisle/internal/desktop/voicecapture"
	"whaleisle/internal/desktop/voiceinput"
	"whaleisle/internal/harness"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/composer"
)

type composerVoiceOperation struct {
	control    *voiceinput.Controller
	sessions   *session.Store
	draft      *composer.DraftState
	id         string
	win        *mygo.Window
	generation int
}

func (s *mainState) voiceCurrent(op *composerVoiceOperation) bool {
	s.mu.Lock()
	visible := !s.launcherOpen && !s.settingsOpen && !s.usageOpen && !s.scheduleOpen && !s.imOpen && !s.marketOpen && !s.remoteOpen && !s.webviewOpen && !s.browserPreviewOpen && !s.assistantOpen
	current := visible && s.voice == op && !s.quitting && s.win == op.win && s.sessions == op.sessions && s.composerDrafts[op.id] == op.draft && s.bootStatus.Generation == op.generation && s.bootStatus.State == harness.Ready && s.dshMuxReady
	s.mu.Unlock()
	if !current {
		return false
	}
	snap := op.sessions.Snapshot()
	return snap.ClientReady && snap.SelectedID == op.id
}
func (s *mainState) cancelComposerVoice() { s.cancelComposerVoiceOperation(nil) }
func (s *mainState) cancelComposerVoiceOperation(expected *composerVoiceOperation) {
	s.mu.Lock()
	if expected != nil && s.voice != expected {
		s.mu.Unlock()
		return
	}
	op := s.voice
	s.voice = nil
	s.mu.Unlock()
	if op != nil {
		op.control.Cancel()
	}
}
func (s *mainState) cancelComposerVoiceIfStale() {
	s.mu.Lock()
	op := s.voice
	s.mu.Unlock()
	if op != nil && !s.voiceCurrent(op) {
		s.cancelComposerVoiceOperation(op)
	}
}
func (s *mainState) cancelComposerVoiceOnBlur() {
	s.mu.Lock()
	op := s.voice
	s.mu.Unlock()
	if op != nil {
		phase := op.control.Snapshot().Phase
		if phase == "requesting" || phase == "recording" {
			s.cancelComposerVoiceOperation(op)
		}
	}
}
func (s *mainState) composerVoiceStatus() voiceinput.Status {
	s.mu.Lock()
	op := s.voice
	s.mu.Unlock()
	if op == nil {
		return voiceinput.Status{}
	}
	return op.control.Snapshot()
}
func (s *mainState) startComposerVoice(win *mygo.Window, sessions *session.Store, id string, draft *composer.DraftState) {
	if sessions == nil || win == nil {
		return
	}
	insertion, ok := composer.CaptureSpeechInsertion(draft)
	if !ok {
		return
	}
	microphone := s.voiceMicrophone()
	device := microphone.Snapshot().Selected
	s.pauseVoiceDevices()
	s.mu.Lock()
	if s.quitting || !s.dshMuxReady || s.bootStatus.State != harness.Ready {
		s.mu.Unlock()
		draft.Error = "Host 未连接，无法开始录音 / Host offline"
		return
	}
	old := s.voice
	if old != nil && old.control.Snapshot().Busy() {
		s.mu.Unlock()
		return
	}
	op := &composerVoiceOperation{sessions: sessions, draft: draft, id: id, win: win, generation: s.bootStatus.Generation}
	s.voice = op
	op.control = voiceinput.New(voiceinput.Options{
		Capture: func(ctx context.Context, limits voicecapture.Limits) (voicecapture.Capture, error) {
			return voicecapture.StartDevice(ctx, limits, device)
		},
		Prepare: func(ctx context.Context) (voicecapture.Limits, voiceinput.Transcribe, error) {
			plan, err := sessions.PrepareSpeechRecording(ctx)
			return plan.Limits, plan.Transcribe, err
		},
		Valid:    func() bool { return s.voiceCurrent(op) },
		Dispatch: func(fn func()) { win.Update(fn) },
		Changed:  func() { win.Update(func() {}) },
		Insert:   func(text string) bool { return insertion.Insert(draft, text) },
	})
	s.mu.Unlock()
	if old != nil {
		old.control.Cancel()
	}
	if !s.voiceCurrent(op) {
		s.cancelComposerVoice()
		return
	}
	op.control.Start()
}
func (s *mainState) stopComposerVoice() {
	s.mu.Lock()
	op := s.voice
	s.mu.Unlock()
	if op != nil {
		op.control.Stop()
	}
}
func (s *mainState) insertComposerVoice() {
	s.mu.Lock()
	op := s.voice
	s.mu.Unlock()
	if op != nil && s.voiceCurrent(op) {
		op.control.InsertPending(func(text string) bool { return composer.InsertPendingSpeech(op.draft, text) })
	}
}
