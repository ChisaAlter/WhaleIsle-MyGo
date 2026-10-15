package browserpreview

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestEndFrameCancelsAnUnpaintedPreview(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	tab := &Tab{ActionMessage: "select"}
	b := &Controller{Tabs: []*Tab{tab}, operation: &previewOperation{tab: tab, ctx: ctx, cancel: cancel}}
	b.EndFrame()
	if ctx.Err() != context.Canceled || b.operation != nil || tab.ActionMessage != "" {
		t.Fatal("hidden browser left its selection active")
	}
}

func TestEndFramePreservesRenderedPreviewOperation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tab := &Tab{}
	b := &Controller{Tabs: []*Tab{tab}, rendered: true, operation: &previewOperation{tab: tab, ctx: ctx, cancel: cancel}}
	b.EndFrame()
	if ctx.Err() != nil || b.operation == nil {
		t.Fatal("visible docked browser was cancelled")
	}
}

func TestSaveBrowserScreenshotCancellationDoesNotCreateArtifact(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if path, err := saveBrowserScreenshot(ctx, []byte("png")); err != context.Canceled || path != "" {
		t.Fatalf("cancelled save = %q, %v", path, err)
	}
}

func TestBrowserPickerScriptHasNoNativeBindingOrInputValueReads(t *testing.T) {
	// The embedded source lives with the desktop adapter. Keep a repository
	// regression for the privacy boundary without exercising a real browser.
	data, err := os.ReadFile("../../desktop/webview/picker.js")
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"window.mygo", "chrome.webview", "outerHTML", ".value"} {
		if strings.Contains(string(data), forbidden) {
			t.Fatalf("picker contains forbidden capability %q", forbidden)
		}
	}
}
