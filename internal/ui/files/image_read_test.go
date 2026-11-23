package files

import (
	"context"
	"testing"
	"time"
)

func TestImageReadLifecyclePreservesView(t *testing.T) {
	tab := &FileTab{Path: "image.svg", image: &imagePreview{loading: true}, Draft: "draft"}
	tab.imageView.scale = 2
	tabs := &Tabs{imageReadTab: tab}
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
			started, canceled := make(chan struct{}), make(chan struct{})
			tab := &FileTab{Path: "image.svg"}
			p := newImagePreview(imageReadOwner{tab: tab, path: name}, "lifecycle\x00"+name+time.Now().String(), "image.svg", nil, func(ctx context.Context) ([]byte, error) {
				close(started)
				<-ctx.Done()
				close(canceled)
				return nil, ctx.Err()
			})
			tab.image = p
			tabs := &Tabs{imageReadTab: tab}
			p.sync(time.Now())
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
