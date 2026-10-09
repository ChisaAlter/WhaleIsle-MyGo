// Command whaleisle is the Go desktop shell of Whale Isle.
//
// This file is the entry skeleton of the app: it wires up the single
// instance lock, opens the main window as native UI (package ui, no
// webview), adds a tray icon, and keeps the window alive in the tray when
// the user closes it. The window shows a placeholder view for now; later
// milestones replace it with the real interface.
package main

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// mainState guards the window and quit flag below. Every callback runs on
// the main thread, so the mutex is cheap, but it keeps the state correct
// even if a handler ever moves off it.
type mainState struct {
	mu      sync.Mutex
	win     *mygo.Window // the main window; nil until WhenReady creates it
	quitting bool         // set before the app quits, so OnClose lets it through
}

var state mainState

// showMainWindow shows the main window and gives it keyboard focus. It is
// called when a second instance hands over its command line and from the
// tray menu's "打开主窗口" item.
func (s *mainState) showMainWindow() {
	s.mu.Lock()
	win := s.win
	s.mu.Unlock()
	if win == nil {
		return
	}
	win.Show()
	win.Restore() // undo a minimized state; no-op otherwise
	win.Focus()
}

func main() {
	// Name the app before Run: it keys the single instance lock and the
	// user data, cache and log directories, and a `go run` build has no
	// packaged configuration to read the name from.
	mygo.App.SetName("Whale Isle")

	// Single instance: a second launch passes its command line to this
	// process and exits; we answer by showing and focusing the window.
	if !mygo.App.RequestSingleInstanceLock() {
		return
	}
	mygo.App.OnSecondInstance(func(args []string, workingDir string) {
		state.showMainWindow()
	})

	mygo.App.WhenReady(func() {
		// The main window is native UI: package ui draws it on the GPU,
		// no webview starts, and the first frame paints as it shows, so
		// there is no page load to hide. BackgroundColor uses light-dark
		// so the empty window matches the desktop appearance at once.
		win := mygo.NewWindow(mygo.WindowOptions{
			Title:           "Whale Isle",
			Width:           1200,
			Height:          800,
			StateKey:        "main",
			BackgroundColor: "light-dark(#f5f5f7, #1e1e1e)",
			Content:         ui.View(placeholderView),
		})

		// Close-to-tray: hide the window instead of closing it. The tray
		// menu's 退出 clears quitting first, so Quit can close the window
		// for real; see the OnClose guard and the quit handler below.
		win.OnClose(func(e *mygo.CloseEvent) {
			state.mu.Lock()
			quitting := state.quitting
			state.mu.Unlock()
			if quitting {
				return // let the close proceed
			}
			e.PreventDefault()
			win.Hide()
		})

		state.mu.Lock()
		state.win = win
		state.mu.Unlock()

		// The tray icon sits in the notification area. Load the PNG from
		// resources, scale it down to a tray-sized bitmap, and give it a
		// menu; a tray with a menu shows it on click.
		tray, err := mygo.NewTray(mygo.TrayOptions{
			Icon:     trayIcon(),
			ToolTip:  "Whale Isle",
			Menu:     trayMenu(),
		})
		if err != nil {
			log.Println("tray:", err)
		} else {
			_ = tray // the tray lives until the app quits
		}
	})

	// Run the event loop until the app quits.
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

// trayMenu builds the tray's menu: show the window, or quit. Quit sets the
// quitting flag first so OnClose lets the windows close instead of hiding
// them; App.Quit closes every window, and the last closed window quits the
// app because we add no OnWindowAllClosed listener.
func trayMenu() *mygo.Menu {
	return mygo.NewMenu([]*mygo.MenuItem{
		{Label: "打开主窗口", Click: func(*mygo.MenuItem, *mygo.Window) {
			state.showMainWindow()
		}},
		mygo.Separator(),
		{Label: "退出", Click: func(*mygo.MenuItem, *mygo.Window) {
			state.mu.Lock()
			state.quitting = true
			state.mu.Unlock()
			mygo.App.Quit()
		}},
	})
}

// placeholderView is the main window's first interface: the product name
// and a status line. The view function runs again for every frame, so it
// only reads state and builds elements.
func placeholderView(c *ui.Context) {
	ui.Column(c).Fill().Center().Gap(12).Children(func() {
		ui.Text(c, "Whale Isle").FontSize(40).Bold()
		ui.Text(c, "骨架：窗口/托盘就绪").FontSize(14)
	})
}

// trayIcon returns the tray icon as PNG bytes, scaled to 32x32: tray icons
// want a small image (16x16 points, 32x32 pixels for high-resolution
// displays), and resources/icon.png is a 1024-pixel square that Windows
// would otherwise scale at the last moment. The file is read from the
// resources directory next to the working directory first, then next to
// the executable, so both `go run` and a built binary find it.
func trayIcon() []byte {
	for _, base := range []string{".", filepath.Dir(os.Args[0])} {
		data, err := os.ReadFile(filepath.Join(base, "resources", "icon.png"))
		if err != nil {
			continue
		}
		if icon, err := scalePNG(data, 32); err == nil {
			return icon
		}
		return data // unreadable as an image; hand the raw PNG to MyGo
	}
	log.Println("tray icon not found; running without one")
	return nil
}

// scalePNG decodes a PNG and scales it to size x size with nearest
// neighbor sampling, returning the re-encoded PNG. Stdlib only, so the
// build needs no image dependencies.
func scalePNG(data []byte, size int) ([]byte, error) {
	src, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	b := src.Bounds()
	if b.Dx() == size && b.Dy() == size {
		return data, nil // already the right size
	}
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		sy := b.Min.Y + y*b.Dy()/size
		for x := 0; x < size; x++ {
			sx := b.Min.X + x*b.Dx()/size
			dst.Set(x, y, src.At(sx, sy))
		}
	}
	var out bytes.Buffer
	if err := png.Encode(&out, dst); err != nil {
		return nil, fmt.Errorf("encode tray icon: %w", err)
	}
	return out.Bytes(), nil
}
