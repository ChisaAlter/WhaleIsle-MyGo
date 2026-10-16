package browserpreview

import (
	"context"
	"errors"
	"os"
)

// startAnnotation captures the draft identity before asking the page to show
// tools. Page completion stages a result only; the native button commits it.
func (b *Controller) startAnnotation(t *Tab) {
	if b.operation != nil || b.parent == nil || t == nil || t.Host == nil {
		return
	}
	target := ActionTarget{}
	if b.actionTarget != nil {
		target = b.actionTarget()
	}
	if target.Valid == nil || !target.Valid() || target.AppendAnnotation == nil {
		t.Error = "请先选择可编辑会话，再添加网页批注"
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	op := &previewOperation{tab: t, host: t.Host, target: target, ctx: ctx, cancel: cancel, kind: "annotation"}
	b.operation = op
	t.Error = ""
	t.ActionMessage = "可视批注：Shift 多选、框选、画笔、橡皮、样式与评论；Review 后在原生工具栏确认加入草稿"
	b.update()
	go func() {
		annotation, data, err := op.host.AnnotateBrowser(ctx)
		path := ""
		if err == nil && ctx.Err() == nil {
			path, err = saveBrowserScreenshot(ctx, data)
		}
		b.parent.Update(func() {
			if !b.actionCurrent(op) {
				if path != "" {
					_ = os.Remove(path)
				}
				if b.operation == op {
					b.CancelAction()
				}
				return
			}
			if err != nil {
				b.operation = nil
				cancel()
				t.ActionMessage = ""
				if !errors.Is(err, context.Canceled) {
					t.Error = err.Error()
				}
				return
			}
			op.pageCurrent = func() bool { return op.host.AnnotationCurrent(annotation) }
			if !op.pageCurrent() {
				if path != "" {
					_ = os.Remove(path)
				}
				b.CancelAction()
				return
			}
			op.annotation = annotation.Markdown()
			op.attachmentPath = path
			op.ready = true
			t.ActionMessage = "批注已准备：请检查后点击「批注和截图加入草稿」；取消会删除本次临时截图"
		})
	}()
}

func (b *Controller) appendAnnotation(op *previewOperation) {
	if !b.actionCurrent(op) || !op.ready || op.attachmentPath == "" {
		return
	}
	if !appendPreparedAnnotation(op, b.actionCurrent) {
		op.tab.Error = "草稿或所属会话已变化；批注未写入"
		return
	}
	// Composer now owns the attachment through its existing preparation/upload
	// flow. Never auto-submit, switch sessions, or remove its source underneath it.
	op.attachmentPath = ""
	op.tab.ActionMessage = "批注和裁剪截图已加入发起会话草稿，请检查后发送"
	b.operation = nil
	op.cancel()
	b.update()
}

// appendPreparedAnnotation is the final native confirmation boundary. The
// caller supplies the live owner check; page completion never calls this.
func appendPreparedAnnotation(op *previewOperation, current func(*previewOperation) bool) bool {
	if op == nil || current == nil || !current(op) || !op.ready || op.attachmentPath == "" || op.target.Valid == nil || !op.target.Valid() || (op.pageCurrent != nil && !op.pageCurrent()) || op.target.AppendAnnotation == nil {
		return false
	}
	if !op.target.AppendAnnotation(op.annotation, op.attachmentPath) {
		return false
	}
	op.attachmentPath = ""
	return true
}
