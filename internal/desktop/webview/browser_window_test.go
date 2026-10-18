package webview

import (
	"strings"
	"testing"

	"github.com/egoist/mygo"
)

func TestBrowserWindowCreationReportsUnsupportedRuntime(t *testing.T) {
	w, err := recoverBackendWindowCreation(func() *mygo.Window {
		panic("mygo: cannot create window: persistent Browser partitions require macOS 14 or later")
	})
	if w != nil || err == nil || !strings.Contains(err.Error(), "macOS 14") {
		t.Fatalf("lost native initialization failure: %v, %v", w, err)
	}
}
func TestBrowserWindowCreationDoesNotHideUnrelatedPanic(t *testing.T) {
	sentinel := "mygo: invalid window options"
	defer func() {
		if recover() != sentinel {
			t.Fatal("unrelated panic was swallowed")
		}
	}()
	_, _ = recoverBackendWindowCreation(func() *mygo.Window { panic(sentinel) })
}
func TestBrowserWindowCreationPreservesWindow(t *testing.T) {
	want := &mygo.Window{}
	got, err := recoverBackendWindowCreation(func() *mygo.Window { return want })
	if got != want || err != nil {
		t.Fatalf("valid window changed: %v", err)
	}
}
