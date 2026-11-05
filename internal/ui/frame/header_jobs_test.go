package frame

import (
	"context"
	"encoding/json"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/jobflow"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

type headerJobStream struct{ frames chan json.RawMessage }

func (s *headerJobStream) Recv(ctx context.Context) (json.RawMessage, error) {
	select {
	case raw := <-s.frames:
		return raw, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
func (*headerJobStream) Cancel() {}

type headerJobOpen struct {
	method  string
	request map[string]any
	stream  *headerJobStream
	ctx     context.Context
}
type headerJobTransport struct {
	opens chan headerJobOpen
	kills chan map[string]any
}

func (f *headerJobTransport) Open(ctx context.Context, method string, request map[string]any) (jobflow.Stream, error) {
	stream := &headerJobStream{make(chan json.RawMessage, 8)}
	f.opens <- headerJobOpen{method, request, stream, ctx}
	return stream, nil
}
func (f *headerJobTransport) Kill(_ context.Context, request map[string]any) (string, error) {
	f.kills <- request
	return "requested", nil
}
func (*headerJobTransport) Retryable(error) bool { return false }

type headerJobsFixture struct {
	controller *jobflow.Controller
	transport  *headerJobTransport
	notified   chan struct{}
}

func newHeaderJobsFixture(t *testing.T) *headerJobsFixture {
	t.Helper()
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(old) })
	f := &headerJobsFixture{transport: &headerJobTransport{make(chan headerJobOpen, 16), make(chan map[string]any, 8)}, notified: make(chan struct{}, 32)}
	f.controller = jobflow.New(func() {
		select {
		case f.notified <- struct{}{}:
		default:
		}
	})
	f.controller.Bind(f.transport, "viewer")
	f.controller.Mount()
	t.Cleanup(f.controller.Unmount)
	return f
}
func (f *headerJobsFixture) next(t *testing.T) headerJobOpen {
	t.Helper()
	select {
	case call := <-f.transport.opens:
		return call
	case <-time.After(2 * time.Second):
		t.Fatal("isolated jobs stream missing")
		return headerJobOpen{}
	}
}
func (f *headerJobsFixture) wait(t *testing.T, ready func(jobflow.Snapshot) bool) {
	t.Helper()
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	for !ready(f.controller.Snapshot()) {
		select {
		case <-f.notified:
		case <-timer.C:
			t.Fatal("isolated jobs state did not settle")
		}
	}
}
func headerSnapshot(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	dir := os.Getenv("SECONDARY_SNAPSHOT_DIR")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(filepath.Join(dir, name+".png"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := png.Encode(file, tt.Image()); err != nil {
		t.Fatal(err)
	}
}

func TestJobsHeaderPopoverRosterOutputStopAndFinished(t *testing.T) {
	f := newHeaderJobsFixture(t)
	roster := f.next(t)
	compact := false
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			ui.Row(c).FillWidth().Height(40).Padding(4, 20).Clip().Children(func() {
				ui.Text(c, "Conversation").Grow(1)
				jobsHeaderView(c, f.controller, compact, theme.Resolve("deepseek.light").Light)
			})
			ui.Text(c, "Conversation body")
		})
	}, 900, 700)
	if tt.HasText("background job") {
		t.Fatal("empty initial watch flashed a trigger")
	}
	headerSnapshot(t, tt, "jobs-empty")
	raw := fmt.Sprintf(`{"type":"rows","jobs":[{"id":"live","owner":"exact-owner","kind":"shell","label":%q,"status":"running","startedAt":%d},{"id":"finished","kind":"shell","label":"Completed command","status":"completed","startedAt":1000,"finishedAt":2000,"output":{"total":4}}]}`, strings.Repeat("long build command ", 12), time.Now().Add(-2*time.Minute).UnixMilli())
	roster.stream.frames <- json.RawMessage(raw)
	f.wait(t, func(s jobflow.Snapshot) bool { return len(s.Rows) == 2 })
	tt.Frame()
	trigger, _ := tt.Find("1 background job running")
	if trigger.H != 28 {
		t.Fatalf("jobs trigger: %+v", trigger)
	}
	if err := tt.Click("1 background job running"); err != nil {
		t.Fatal(err)
	}
	popup, ok := tt.Find("Background jobs popover")
	if !ok || popup.W != 500 || popup.H > 480 || popup.Y < trigger.Y+trigger.H+5 || popup.X < 12 || popup.X+popup.W > 888 {
		t.Fatalf("jobs overlay clipped or unbounded: %+v", popup)
	}
	if tt.HasText("Completed command") {
		t.Fatal("finished tail did not default-fold beside live work")
	}
	headerSnapshot(t, tt, "jobs-live")
	if err := tt.Click("Toggle finished jobs"); err != nil {
		t.Fatal(err)
	}
	finished, _ := tt.Find("Job row: finished")
	live, _ := tt.Find("Job row: live")
	if finished.H >= live.H {
		t.Fatalf("settled tail not compact: live=%+v finished=%+v", live, finished)
	}
	if err := tt.Click("Toggle job output: live"); err != nil {
		t.Fatal(err)
	}
	follow := f.next(t)
	if follow.method != "job/follow" || follow.request["sessionId"] != "exact-owner" {
		t.Fatalf("wrong follow owner: %+v", follow)
	}
	follow.stream.frames <- json.RawMessage(`{"type":"opened","job":{"id":"live","status":"running","output":{"earliest":0}},"from":0}`)
	follow.stream.frames <- json.RawMessage(fmt.Sprintf(`{"type":"output","chunks":[{"at":0,"text":%q}],"next":1000}`, strings.Repeat("Isolated retained output\n", 40)))
	f.wait(t, func(s jobflow.Snapshot) bool { return s.Observed.Text != "" })
	tt.Frame()
	output, _ := tt.Find("Job output scroll: live")
	if output.H != 288 || output.W > 500 {
		t.Fatalf("output viewport: %+v", output)
	}
	headerSnapshot(t, tt, "jobs-output")
	follow.stream.frames <- json.RawMessage(`{"type":"invalid-output"}`)
	f.wait(t, func(s jobflow.Snapshot) bool { return s.Observed.Error != "" })
	tt.Frame()
	headerSnapshot(t, tt, "jobs-output-error")
	if err := tt.Click("Retry output"); err != nil {
		t.Fatal(err)
	}
	retryOutput := f.next(t)
	if retryOutput.method != "job/follow" || retryOutput.request["sessionId"] != "exact-owner" || retryOutput.request["from"] != int64(1000) {
		t.Fatalf("output retry lost owner/cursor: %+v", retryOutput)
	}
	follow = retryOutput
	if err := tt.Click("Stop task "+strings.Repeat("long build command ", 12)); err != nil {
		t.Fatal(err)
	}
	select {
	case <-f.transport.kills:
		t.Fatal("first stop press sent RPC")
	default:
	}
	if err := tt.Click("Stop task "+strings.Repeat("long build command ", 12)); err != nil {
		t.Fatal(err)
	}
	select {
	case request := <-f.transport.kills:
		if request["sessionId"] != "viewer" || request["jobId"] != "live" {
			t.Fatal(request)
		}
	case <-time.After(time.Second):
		t.Fatal("second stop press not sent")
	}
	_ = tt.Click("Stop task "+strings.Repeat("long build command ", 12))
	select {
	case <-f.transport.kills:
		t.Fatal("pending stop sent duplicate RPC")
	default:
	}
	tt.Key(0, ui.KeyEscape)
	if f.controller.Snapshot().Open || f.controller.Snapshot().Observed.Text != "" {
		t.Fatal("popup close retained visible observation")
	}
	select {
	case <-follow.ctx.Done():
	default:
		t.Fatal("popup close did not cancel output")
	}
	select {
	case <-roster.ctx.Done():
		t.Fatal("popup close canceled roster")
	default:
	}
	if err := tt.Click("1 background job running"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("Completed command") {
		t.Fatal("finished fold choice did not survive popup close")
	}
	if err := tt.Click("Clear finished jobs"); err != nil {
		t.Fatal(err)
	}
	if tt.HasText("Completed command") {
		t.Fatal("clear retained finished row")
	}
	compact = true
	tt.SetSize(360, 420)
	popup, _ = tt.Find("Background jobs popover")
	if popup.W > 328 || popup.H > 280 {
		t.Fatalf("narrow popup: %+v", popup)
	}
	trigger, _ = tt.Find("1 background job running")
	if trigger.W != 28 {
		t.Fatalf("compact jobs trigger: %+v", trigger)
	}
	headerSnapshot(t, tt, "jobs-narrow")
}

func TestJobsReadErrorRetryLoadingAndEmptyDisposal(t *testing.T) {
	f := newHeaderJobsFixture(t)
	first := f.next(t)
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Row(c).Padding(8).Children(func() { jobsHeaderView(c, f.controller, false, theme.Resolve("deepseek.light").Light) })
	}, 600, 400)
	first.stream.frames <- json.RawMessage(`{"type":"invalid"}`)
	f.wait(t, func(s jobflow.Snapshot) bool { return s.Error != "" })
	tt.Frame()
	if err := tt.Click("Background jobs need a fresh read"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("Retry jobs") {
		t.Fatal("roster failure has no retry")
	}
	headerSnapshot(t, tt, "jobs-error")
	if err := tt.Click("Retry jobs"); err != nil {
		t.Fatal(err)
	}
	retry := f.next(t)
	tt.Frame()
	if _, ok := tt.Find("Loading jobs"); !ok {
		t.Fatal("retry loading state missing")
	}
	headerSnapshot(t, tt, "jobs-loading")
	retry.stream.frames <- json.RawMessage(`{"type":"rows","jobs":[]}`)
	f.wait(t, func(s jobflow.Snapshot) bool { return !s.Loading && s.Error == "" })
	tt.Frame()
	if _, ok := tt.Find("Background jobs popover"); ok {
		t.Fatal("empty successful roster retained popup")
	}
	if tt.HasText("Background jobs need") {
		t.Fatal("empty successful roster retained trigger")
	}
}
