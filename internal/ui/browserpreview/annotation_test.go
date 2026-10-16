package browserpreview

import (
	"context"
	"os"
	"strings"
	"testing"
)

func setupB17Annotation() (*previewOperation, *int) {
	calls := 0
	return &previewOperation{tab: &Tab{}, ready: true, annotation: "page data", attachmentPath: "locally-generated.png", pageCurrent: func() bool { return true }, target: ActionTarget{Valid: func() bool { return true }, AppendAnnotation: func(text, path string) bool { calls++; return text == "page data" && path == "locally-generated.png" }}}, &calls
}

func TestB17NativeConfirmAndStaleTargets(t *testing.T) {
	live := func(*previewOperation) bool { return true }
	stale := func(*previewOperation) bool { return false }
	op, calls := setupB17Annotation()
	if *calls != 0 {
		t.Fatal("staging must not append")
	}
	if appendPreparedAnnotation(op, stale) || *calls != 0 {
		t.Fatal("stale owner appended")
	}
	for n, change := range []func(*previewOperation){
		func(o *previewOperation) { o.target.Valid = func() bool { return false } },
		func(o *previewOperation) { o.pageCurrent = func() bool { return false } },
		func(o *previewOperation) { o.ready = false },
		func(o *previewOperation) { o.attachmentPath = "" },
	} {
		o, c := setupB17Annotation()
		change(o)
		if appendPreparedAnnotation(o, live) || *c != 0 {
			t.Fatalf("invalid target %d appended", n)
		}
	}
	if !appendPreparedAnnotation(op, live) || *calls != 1 || op.attachmentPath != "" {
		t.Fatal("native confirmation did not transfer attachment exactly once")
	}
	if appendPreparedAnnotation(op, live) || *calls != 1 {
		t.Fatal("duplicate append")
	}
	op, calls = setupB17Annotation()
	ctx, cancel := context.WithCancel(context.Background())
	op.ctx = ctx
	op.cancel = cancel
	file, err := os.CreateTemp(t.TempDir(), "annotation-*.png")
	if err != nil {
		t.Fatal(err)
	}
	file.Close()
	op.attachmentPath = file.Name()
	b := &Controller{operation: op}
	b.CancelAction()
	if _, err = os.Stat(file.Name()); !os.IsNotExist(err) || *calls != 0 || ctx.Err() != context.Canceled {
		t.Fatal("cancel did not discard uncommitted image")
	}
}

func TestB17RendererCompletionOnlyStages(t *testing.T) {
	data, err := os.ReadFile("annotation.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	start := strings.Index(source, "func (b *Controller) startAnnotation(")
	end := strings.Index(source, "func (b *Controller) appendAnnotation(")
	if start < 0 || end <= start {
		t.Fatal("missing native staging")
	}
	stage := source[start:end]
	if strings.Contains(stage, "target.AppendAnnotation(") || strings.Contains(stage, "appendPreparedAnnotation(") || !strings.Contains(stage, "op.ready = true") {
		t.Fatal("renderer completion bypassed native confirmation")
	}
	data, err = os.ReadFile("actions.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `ui.Button(c, "批注和截图加入草稿")`) || !strings.Contains(string(data), "b.appendAnnotation(op)") {
		t.Fatal("missing native confirmation button")
	}
}
