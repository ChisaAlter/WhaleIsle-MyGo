package main

import (
	"testing"

	"github.com/egoist/mygo"

	"whaleisle/internal/product"
)

func TestDesktopRenameKeepsNativeProfilePaths(t *testing.T) {
	t.Setenv("WHALEISLE_ISOLATED", "")
	priorApp, priorPortable := mygo.App, portableDesktopRoot
	mygo.App, portableDesktopRoot = &mygo.Application{}, ""
	t.Cleanup(func() { mygo.App, portableDesktopRoot = priorApp, priorPortable })
	mygo.App.SetName(product.LegacyNativeName)
	paths := map[mygo.PathName]string{}
	for _, key := range []mygo.PathName{mygo.PathUserData, mygo.PathCache, mygo.PathLogs} {
		path, err := mygo.App.Path(key)
		if err != nil {
			t.Fatal(err)
		}
		paths[key] = path
	}
	configureDesktopName()
	if mygo.App.Name() != product.Name {
		t.Fatalf("display name = %q", mygo.App.Name())
	}
	for key, want := range paths {
		if got, err := mygo.App.Path(key); err != nil || got != want {
			t.Errorf("%s moved during rename: %q, want %q (error %v)", key, got, want, err)
		}
	}
}
