package files

import (
	"context"
	"testing"
	"time"
)

func TestImageReadQueueReplacesGenerationAndCancels(t *testing.T) {
	q := &imageReadQueue{}
	owner := imageReadOwner{tab: &FileTab{}, path: "same.svg"}
	started, canceled, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	q.request(owner, func(ctx context.Context) *imagePreview {
		close(started)
		<-ctx.Done()
		close(canceled)
		<-release // Simulate a decoder/syscall that cannot stop immediately.
		return &imagePreview{note: "stale"}
	})
	<-started
	q.request(owner, func(context.Context) *imagePreview { return &imagePreview{note: "latest"} })
	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("old decode was not canceled")
	}
	if _, ok := q.take(owner); ok {
		t.Fatal("premature result")
	}
	close(release)
	deadline := time.After(time.Second)
	for {
		if result, ok := q.take(owner); ok {
			if result.note != "latest" {
				t.Fatal("stale generation applied")
			}
			break
		}
		select {
		case <-deadline:
			t.Fatal("latest decode did not finish")
		default:
			time.Sleep(time.Millisecond)
		}
	}
}

func TestImageReadQueueCloseAndReopenedOwner(t *testing.T) {
	q := &imageReadQueue{}
	old := imageReadOwner{tab: &FileTab{}, path: "same.svg"}
	reopened := imageReadOwner{tab: &FileTab{}, path: "same.svg"}
	started, done := make(chan struct{}), make(chan struct{})
	q.request(old, func(ctx context.Context) *imagePreview {
		close(started)
		<-ctx.Done()
		close(done)
		return &imagePreview{note: "late"}
	})
	<-started
	q.stop(true)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("close did not cancel")
	}
	if _, ok := q.take(reopened); ok {
		t.Fatal("old result reached reopened tab")
	}
	if _, ok := q.take(old); ok {
		t.Fatal("closed queue published result")
	}
	q.request(reopened, func(context.Context) *imagePreview { t.Error("closed queue restarted"); return nil })
}

func TestImageReadLifecyclePreservesView(t *testing.T) {
	tab := &FileTab{Path: "image.svg", image: &imagePreview{loading: true}, Draft: "draft"}
	tab.imageView.scale = 2
	tabs := &Tabs{imageReader: &imageReadQueue{}, imageReadTab: tab}
	tabs.stopImageRead(false)
	if tab.image != nil || tab.imageView.scale != 2 || tab.Draft != "draft" {
		t.Fatal("cancel changed view/draft or kept loading placeholder")
	}
	if tabs.imageReadTab != nil {
		t.Fatal("canceled owner retained")
	}
}

func TestImageReadPauseBindDisposeCancel(t *testing.T) {
	for _, name := range []string{"pause", "bind", "dispose"} {
		t.Run(name, func(t *testing.T) {
			tab := &FileTab{Path: "image.svg", image: &imagePreview{loading: true}}
			tabs := &Tabs{imageReader: &imageReadQueue{}, imageReadTab: tab}
			started, canceled := make(chan struct{}), make(chan struct{})
			tabs.imageReader.request(imageReadOwner{tab: tab}, func(ctx context.Context) *imagePreview {
				close(started)
				<-ctx.Done()
				close(canceled)
				return &imagePreview{}
			})
			<-started
			switch name {
			case "pause":
				tabs.Pause()
			case "bind":
				tabs.Bind("new-owner", "new-root")
			case "dispose":
				tabs.Dispose()
			}
			select {
			case <-canceled:
			case <-time.After(time.Second):
				t.Fatal("lifecycle did not cancel decode")
			}
			if tab.image != nil {
				t.Fatal("unfinished placeholder retained")
			}
		})
	}
}
