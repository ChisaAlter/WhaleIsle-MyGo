package composer

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/voiceinput"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestComposerIntakeCountsImagesAndBytes(t *testing.T) {
	d := NewDraftState("")
	d.Attachments = []session.ComposerAttachment{
		{Path: "a.png", MediaType: "image/png", ImageData: strings.Repeat("x", 4096)},
		{Path: "b.txt", MediaType: "text/plain"},
		{Path: "c.jpg"},
	}
	d.Attachments[2].Upload.File.Bytes = 2048
	images, bytes := composerIntake(d)
	if images != 2 {
		t.Fatalf("counted %d images, want 2", images)
	}
	if want := int64(4096*3/4) + 2048; bytes != want {
		t.Fatalf("counted %d bytes, want %d", bytes, want)
	}
	if nilImages, nilBytes := composerIntake(nil); nilImages != 0 || nilBytes != 0 {
		t.Fatal("nil draft counted")
	}
	if got := composerFileSize(composerMaxImageBytes); got != "10 MB" {
		t.Fatalf("budget reads %q", got)
	}
	if got := composerFileSize(composerMaxTotalBytes); got != "20 MB" {
		t.Fatalf("total budget reads %q", got)
	}
}

func TestVoiceRecordingShowsElapsedClock(t *testing.T) {
	d := NewDraftState("")
	stopped, canceled := 0, 0
	m := Model{State: d, Draft: &d.Text, TextInput: d.Client, UseTextInputClient: true,
		SuggestList: &ui.ListState{Selected: new(int)},
		Snapshot: session.ViewState{ClientReady: true, SelectedID: "A", SpeechSelection: "provider/en",
			SpeechReadinessKnown: true, SpeechReady: true},
		Voice:       voiceinput.Status{Phase: "recording"},
		OnStopVoice: func() { stopped++ }, OnCancelVoice: func() { canceled++ }}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 720, 360)
	start := time.Unix(7000, 0)
	tt.SetTime(start)
	if !tt.HasText("0:00") {
		t.Fatalf("recording has no elapsed clock: %v", tt.Texts())
	}
	tt.SetTime(start.Add(65 * time.Second))
	tt.Frame()
	if !tt.HasText("1:05") {
		t.Fatalf("clock did not advance: %v", tt.Texts())
	}
	m.Voice.Phase = "transcribing"
	tt.SetTime(start.Add(70 * time.Second))
	tt.Frame()
	if tt.HasText("1:05") || tt.HasText("1:10") {
		t.Fatal("elapsed clock outlived the recording phase")
	}
	if !d.VoiceStarted.IsZero() {
		t.Fatal("recording clock state was not retired")
	}
}

func TestAttachmentRetrySurvivesDecodeFailure(t *testing.T) {
	d := NewDraftState("")
	d.Attachments = []session.ComposerAttachment{attachmentFixture("broken.png", []byte("not an image"))}
	retries := 0
	m := Model{State: d, OnRetryAttachment: func(int) { retries++ }}
	tt := ui.NewTester(func(c *ui.Context) {
		paintAttachmentRail(c, m, ui.RGB(0, 0, 0), ui.RGB(80, 80, 80))
	}, 600, 300)
	retryLabel := "Retry upload: broken.png"
	// Decoding runs in a goroutine, so under load the first frame can still be
	// the loading state; wait for the retry seat instead of clicking blind.
	deadline := time.Now().Add(2 * time.Second)
	for !tt.HasText(retryLabel) {
		if time.Now().After(deadline) {
			t.Fatalf("a failed preview never showed its retry: %v", tt.Texts())
		}
		tt.Frame()
		time.Sleep(time.Millisecond)
	}
	if err := tt.Click(retryLabel); err != nil {
		t.Fatalf("a failed preview hid its retry: %v", err)
	}
	if retries != 1 {
		t.Fatalf("retry ran %d times", retries)
	}
}

func TestMenusListTheirKeyboardHints(t *testing.T) {
	m := permissionModelFixture()
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 880, 420)
	if err := tt.Click(i18n.T(i18n.KeyComposerPermissionWorkspace)); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText(i18n.T(i18n.KeyComposerMenuFooter)) {
		t.Fatalf("permission menu has no keyboard hints: %v", tt.Texts())
	}
	if !tt.HasText(i18n.T(i18n.KeyComposerPermissionFullAccess)) {
		t.Fatal("permission menu lost its rows")
	}
}

func TestComposerDropWarningExplainsRefusals(t *testing.T) {
	if got := composerDropWarning(0, 0); got != "" {
		t.Fatalf("an empty draft warned: %q", got)
	}
	if got := composerDropWarning(7, composerMaxImageBytes); got != "" {
		t.Fatalf("a drop that still fits warned: %q", got)
	}
	if got, want := composerDropWarning(composerMaxImages, 0),
		fmt.Sprintf(i18n.T(i18n.KeyComposerDropFull), composerMaxImages); got != want {
		t.Fatalf("full intake reads %q, want %q", got, want)
	}
	if got, want := composerDropWarning(1, composerMaxImageBytes+1),
		fmt.Sprintf(i18n.T(i18n.KeyComposerDropTooLarge), "20 MB"); got != want {
		t.Fatalf("oversized intake reads %q, want %q", got, want)
	}
}
