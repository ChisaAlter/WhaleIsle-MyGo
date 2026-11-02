package pluginui

import (
	"context"
	"fmt"
	"time"

	"github.com/egoist/mygo"
)

func (o *Owner) cancelMirrorTextAt(source *mygo.Window, expected *mirrorTextFocus) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	now, err := mirrorEval[*mirrorTextFocus](ctx, source.Page(), "window.__whaleInlineMirror?.textFocus()")
	if err != nil || !expected.same(now) {
		return
	}
	key := fmt.Sprintf("%q", expected.DocumentKey)
	blocked, err := mirrorEval[bool](ctx, source.Page(), "window.__whaleInlineMirror.textCancel("+key+")")
	if err != nil || !blocked {
		return
	}
	_ = source.Page().TextInput(ctx, mygo.BrowserTextEvent{Kind: mygo.BrowserTextCancelComposition})
	_, _ = source.Page().EvalContext(ctx, "window.__whaleInlineMirror.textResume("+key+")")
}
