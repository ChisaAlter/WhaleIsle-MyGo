//go:build windows && nativetextacceptance && !release && !nativeacceptance && !appearanceacceptance && !livedesktopacceptance && !sidebaracceptance && !sessionrecoveryacceptance && !settingsacceptance && !richhistoryacceptance && !composerflowacceptance

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/git"
	"whaleisle/internal/ui/diffsurface"
	"whaleisle/internal/ui/files"
	"whaleisle/internal/ui/i18n"
)

var nativeTextAcceptanceError error

// This tag owns a standalone process. By default its UI thread enters a private
// noninteractive station before MyGo initializes Win32/OLE. Explicit visible QA
// instead replaces the interactive clipboard with synthetic text, without
// reading or backing up its previous contents.
func TestMain(m *testing.M) {
	flag.Parse()
	if os.Getenv("WHALEISLE_TEXT_CLIPBOARD_PEER") == "1" {
		text, err := readNativeTextClipboardPeer()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		_ = json.NewEncoder(os.Stdout).Encode(text)
		os.Exit(0)
	}
	// The testing package's own timeout starts in m.Run, after the native main
	// loop. Bound this earlier phase too; process exit releases its OS objects.
	watchdog := time.AfterFunc(60*time.Second, func() {
		fmt.Fprintln(os.Stderr, "native text acceptance exceeded 60 seconds")
		os.Exit(124)
	})
	nativeTextAcceptanceError = runNativeTextAcceptance(
		os.Getenv("WHALEISLE_TEXT_ACCEPTANCE_ROOT"),
		os.Getenv("WHALEISLE_TEXT_ACCEPTANCE_OUTPUT"))
	watchdog.Stop()
	os.Exit(m.Run())
}

// An independent receiver requests delayed OLE data while the owner's actual
// Windows message loop stays responsive, just as another application does.
func readNativeTextClipboardPeer() (string, error) {
	user32, kernel32 := syscall.NewLazyDLL("user32.dll"), syscall.NewLazyDLL("kernel32.dll")
	deadline := time.Now().Add(3 * time.Second)
	for {
		if ok, _, _ := user32.NewProc("OpenClipboard").Call(0); ok != 0 {
			break
		}
		if time.Now().After(deadline) {
			return "", errors.New("independent receiver could not open clipboard")
		}
		time.Sleep(10 * time.Millisecond)
	}
	defer user32.NewProc("CloseClipboard").Call()
	h, _, err := user32.NewProc("GetClipboardData").Call(13) // CF_UNICODETEXT
	if h == 0 {
		return "", fmt.Errorf("independent CF_UNICODETEXT read: %w", err)
	}
	n, _, _ := kernel32.NewProc("GlobalSize").Call(h)
	if n == 0 || n > 64<<20 || n%2 != 0 {
		return "", fmt.Errorf("invalid Unicode clipboard allocation: %d", n)
	}
	p, _, err := kernel32.NewProc("GlobalLock").Call(h)
	if p == 0 {
		return "", fmt.Errorf("clipboard GlobalLock: %w", err)
	}
	defer kernel32.NewProc("GlobalUnlock").Call(h)
	return syscall.UTF16ToString(unsafe.Slice((*uint16)(unsafe.Pointer(p)), n/2)), nil
}

func TestNativeTextSelectionAcceptance(t *testing.T) {
	if nativeTextAcceptanceError != nil {
		t.Fatal(nativeTextAcceptanceError)
	}
}

func isolateNativeTextClipboard() error {
	user32 := syscall.NewLazyDLL("user32.dll")
	previous, _, err := user32.NewProc("GetProcessWindowStation").Call()
	if previous == 0 {
		return fmt.Errorf("GetProcessWindowStation: %w", err)
	}
	// Anonymous stations use a login-session name and can already be occupied.
	// A process-specific name avoids that collision; Windows may require an
	// administrator account for named creation. Any denial is terminal: never
	// reopen an existing station or fall back to the interactive clipboard.
	stationName := fmt.Sprintf("WhaleIsleTextAcceptance-%d-%d", os.Getpid(), time.Now().UnixNano())
	stationNameUTF16, _ := syscall.UTF16PtrFromString(stationName)
	// ACCESSCLIPBOARD | CREATEDESKTOP | ACCESSGLOBALATOMS | READATTRIBUTES.
	station, _, err := user32.NewProc("CreateWindowStationW").Call(uintptr(unsafe.Pointer(stationNameUTF16)), 1, 0x002e, 0)
	if station == 0 {
		return fmt.Errorf("create private window station: %w", err)
	}
	closeStation := func() { user32.NewProc("CloseWindowStation").Call(station) }
	type objectFlags struct{ inherit, reserved, flags uint32 }
	var flags objectFlags
	var needed uint32
	ok, _, err := user32.NewProc("GetUserObjectInformationW").Call(station, 1,
		uintptr(unsafe.Pointer(&flags)), unsafe.Sizeof(flags), uintptr(unsafe.Pointer(&needed)))
	if ok == 0 || flags.flags&1 != 0 || station == previous {
		closeStation()
		return fmt.Errorf("private station is not verified noninteractive: flags=%d error=%v", flags.flags, err)
	}
	ok, _, err = user32.NewProc("SetProcessWindowStation").Call(station)
	if ok == 0 {
		closeStation()
		return fmt.Errorf("SetProcessWindowStation: %w", err)
	}
	name, _ := syscall.UTF16PtrFromString("WhaleIsleTextAcceptance")
	desktop, _, err := user32.NewProc("CreateDesktopW").Call(uintptr(unsafe.Pointer(name)), 0, 0, 0, 0x0087, 0)
	if desktop == 0 {
		// Nothing from MyGo has initialized; returning to the old station is safe
		// on this error path, before any clipboard call or native window exists.
		user32.NewProc("SetProcessWindowStation").Call(previous)
		closeStation()
		return fmt.Errorf("CreateDesktopW: %w", err)
	}
	ok, _, err = user32.NewProc("SetThreadDesktop").Call(desktop)
	if ok == 0 {
		user32.NewProc("CloseDesktop").Call(desktop)
		user32.NewProc("SetProcessWindowStation").Call(previous)
		closeStation()
		return fmt.Errorf("SetThreadDesktop: %w", err)
	}
	current, _, _ := user32.NewProc("GetProcessWindowStation").Call()
	objectName := func(handle uintptr) (string, error) {
		var name [256]uint16
		var needed uint32
		ok, _, err := user32.NewProc("GetUserObjectInformationW").Call(handle, 2,
			uintptr(unsafe.Pointer(&name[0])), unsafe.Sizeof(name), uintptr(unsafe.Pointer(&needed)))
		if ok == 0 {
			return "", err
		}
		return syscall.UTF16ToString(name[:]), nil
	}
	createdName, createdErr := objectName(station)
	currentName, currentErr := objectName(current)
	thread, _, _ := syscall.NewLazyDLL("kernel32.dll").NewProc("GetCurrentThreadId").Call()
	currentDesktop, _, _ := user32.NewProc("GetThreadDesktop").Call(thread)
	desktopName, desktopErr := objectName(currentDesktop)
	if createdErr != nil || currentErr != nil || desktopErr != nil || createdName == "" ||
		!strings.EqualFold(createdName, stationName) || !strings.EqualFold(createdName, currentName) || strings.EqualFold(currentName, "WinSta0") ||
		desktopName != "WhaleIsleTextAcceptance" {
		return errors.New("current process station/thread desktop did not match isolated objects")
	}
	// Do not close these handles before process exit: the backend creates its
	// message window and OLE windows on this desktop after App.Run begins.
	return nil
}

func runNativeTextAcceptance(root, output string) (failure error) {
	if !filepath.IsAbs(root) || !filepath.IsAbs(output) || os.Getenv("WHALEISLE_ISOLATED") != "1" {
		return errors.New("absolute owned root/output and WHALEISLE_ISOLATED=1 required")
	}
	if _, err := os.Stat(root); !errors.Is(err, os.ErrNotExist) {
		return errors.New("acceptance root must be fresh and absent")
	}
	visible := os.Getenv("WHALEISLE_TEXT_ACCEPTANCE_MODE") == "visible"
	if visible && os.Getenv("WHALEISLE_ISOLATED_VISIBLE") != "1" {
		return errors.New("visible clipboard QA requires WHALEISLE_ISOLATED_VISIBLE=1 and authorization to replace the clipboard with synthetic text")
	}
	record := map[string]any{
		"schemaVersion": 1, "input": "WindowDriver SurfaceEvent injection into a real hidden MyGo window",
		"clipboard":         "private noninteractive Windows window station; real OLE write and CF_UNICODETEXT read",
		"capture":           "CapturePage Content scene rasterization, not an OS/GPU framebuffer",
		"userClipboardRead": false, "userClipboardWritten": false, "windowsShown": false,
		"systemIMEExercised": false, "hostStarted": false, "modelRequests": 0,
	}
	if visible {
		record["input"] = "WindowDriver SurfaceEvent injection into a real visible MyGo window"
		record["clipboard"] = "interactive Windows clipboard; synthetic text only; no original clipboard backup or read"
		record["userClipboardWritten"], record["windowsShown"] = true, true
		record["clipboardReceiver"] = "independent process reading Win32 CF_UNICODETEXT while owner pumps its normal native message loop"
	}
	checks := []string{}
	defer func() {
		record["checks"], record["pass"] = checks, failure == nil
		if failure != nil {
			record["error"] = failure.Error()
		}
		raw, err := json.MarshalIndent(record, "", "  ")
		if err == nil {
			err = os.WriteFile(output, append(raw, '\n'), 0600)
		}
		failure = errors.Join(failure, err)
	}()
	if !visible {
		if err := isolateNativeTextClipboard(); err != nil {
			return err
		}
		record["privateWindowStationVerified"] = true
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		return err
	}
	// The invoking task owns this one directory and removes it after this
	// process exits, including any test executable placed beside it.
	record["ownedRoot"] = root
	mygo.App.SetName("WhaleIsle hidden text selection acceptance")
	mygo.App.SetPath(mygo.PathUserData, filepath.Join(root, "native"))
	mygo.App.SetPath(mygo.PathCache, filepath.Join(root, "cache"))
	mygo.App.SetPath(mygo.PathLogs, filepath.Join(root, "logs"))
	i18n.SetLocale("en")
	mygo.App.OnReady(func() {
		go func() {
			defer mygo.App.Quit()
			defer mygo.Clipboard.Clear() // Only this run's synthetic clipboard contents.
			if visible {
				fmt.Println("NATIVE_TEXT_STAGE direct synthetic write")
				const probe = "WhaleIsle clipboard QA 中文🙂"
				if err := mygo.Clipboard.Write(transfer.TextData(probe)); err != nil {
					failure = fmt.Errorf("direct synthetic clipboard write: %w", err)
					return
				}
				fmt.Println("NATIVE_TEXT_STAGE direct native read")
				if got := mygo.Clipboard.ReadText(); got != probe {
					failure = fmt.Errorf("direct native clipboard read got %q, want %q", got, probe)
					return
				}
				checks = append(checks, "direct synthetic OLE write and CF_UNICODETEXT read")
			}
			failure = nativeTextSelectionScenarios(root, output, &checks, record)
		}()
	})
	return errors.Join(mygo.App.Run(), failure)
}

func nativeTextSelectionScenarios(root, output string, checks *[]string, record map[string]any) error {
	const count = 1200
	lines := make([]string, count)
	for i := range lines {
		lines[i] = fmt.Sprintf("var row_%04d = \"中文🙂 value %04d\"", i, i)
	}
	text := strings.Join(lines, "\n") + "\n"
	path := "selection.go"
	if err := os.WriteFile(filepath.Join(root, path), []byte(text), 0600); err != nil {
		return err
	}
	tab := &files.FileTab{Path: path, Preview: text, Draft: text}
	tabs := &files.Tabs{Preferences: &files.PreviewPreferences{WordWrap: true},
		Workspaces: map[string]*files.WorkspaceTabs{filepath.Clean(root): {Open: []*files.FileTab{tab}, Active: path}}}
	var list ui.ListState
	editing, previewErr := false, ""
	fileModel := files.Model{ResourceOnly: true, Tabs: tabs, WorkspaceRoot: root, SelectedFile: &path,
		Preview: &tab.Preview, EditDraft: &tab.Draft, PreviewErr: &previewErr, Editing: &editing, PreviewList: &list}
	old, added := make([]string, count), make([]string, count)
	var patch strings.Builder
	patch.WriteString("diff --git a/selection.go b/selection.go\n--- a/selection.go\n+++ b/selection.go\n")
	fmt.Fprintf(&patch, "@@ -1,%d +1,%d @@\n", count, count)
	for i := range old {
		old[i] = fmt.Sprintf("old_%04d 中文🙂", i)
		fmt.Fprintf(&patch, "-%s\n", old[i])
	}
	for i := range added {
		added[i] = fmt.Sprintf("new_%04d 中文🙂", i)
		fmt.Fprintf(&patch, "+%s\n", added[i])
	}
	patchText := patch.String()
	workbench := &diffsurface.Workbench{Root: root, Loaded: true, Repository: true, AgainstHEAD: true,
		Changes: []git.Change{{Path: path, Worktree: 'M'}}}
	defer mygo.RunOnMain(workbench.ClosePane)
	diffModel := diffsurface.Model{WorkspaceRoot: root, Workbench: workbench, Text: &patchText}
	page := "file"
	visible := record["windowsShown"] == true
	var win *mygo.Window
	mygo.RunOnMain(func() {
		win = mygo.NewWindow(mygo.WindowOptions{Title: "WhaleIsle text selection QA", Width: 1040, Height: 620, Hidden: !visible,
			Content: ui.View(func(c *ui.Context) {
				if page == "file" {
					files.View(c.Key("file"), fileModel)
				} else {
					diffsurface.View(c.Key("diff"), diffModel)
				}
			})})
	})
	defer mygo.RunOnMain(win.Destroy)
	var driver *ui.WindowDriver
	mygo.RunOnMain(func() { driver = ui.DriveWindow(win) })
	if driver == nil {
		return errors.New("hidden native window has no live UI driver")
	}
	frame := func() { mygo.RunOnMain(driver.Frame) }
	hasText := func(s string) (has bool) { mygo.RunOnMain(func() { has = driver.HasText(s) }); return }
	frame()
	if win.IsVisible() != visible {
		return errors.New("acceptance window visibility does not match the requested mode")
	}
	copyExact := func(want, label string) error {
		fmt.Println("NATIVE_TEXT_STAGE", label)
		mygo.RunOnMain(func() { driver.Key(ui.Cmd, ui.KeyC) })
		logical, logicalErr := mygo.Clipboard.ReadFormat(transfer.Text)
		fmt.Printf("NATIVE_TEXT_STAGE logical=%d error=%v\n", len(logical), logicalErr)
		var got string
		if visible {
			// The receiver uses Win32 CF_UNICODETEXT, never the in-process
			// provider cache. The owner pumps its normal native message loop.
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			peer := exec.CommandContext(ctx, os.Args[0], "-test.run=^$")
			peer.Env = append(os.Environ(), "WHALEISLE_TEXT_CLIPBOARD_PEER=1")
			var stderr bytes.Buffer
			peer.Stderr = &stderr
			raw, err := peer.Output()
			if err != nil {
				return fmt.Errorf("%s: independent clipboard receiver: %w: %s", label, err, stderr.String())
			}
			if err := json.Unmarshal(raw, &got); err != nil {
				return fmt.Errorf("clipboard receiver JSON: %w", err)
			}
		} else {
			// A private-station run keeps every clipboard access in its own
			// process and cannot read the interactive station.
			got = mygo.Clipboard.ReadText()
		}
		if got != want {
			record["failedCopyLogicalBytes"] = len(logical)
			record["failedCopyLogicalMatches"] = string(logical) == want
			record["failedCopyNativeBytes"] = len(got)
			return fmt.Errorf("%s copied %d bytes; expected exact %d-byte logical range", label, len(got), len(want))
		}
		*checks = append(*checks, label)
		return nil
	}
	clickRune := func(s string, at int, mods ui.Modifiers) (failure error) {
		mygo.RunOnMain(func() {
			x, y, ok := driver.SelectablePoint(s, at)
			w, h := win.ContentSize()
			if !ok || x <= 0 || y <= 0 || x >= float32(w) || y >= float32(h) {
				failure = fmt.Errorf("selectable endpoint is not visible: %q rune=%d point=(%g,%g), content=(%d,%d)", s, at, x, y, w, h)
				return
			}
			driver.ClickAtWith(mods, x, y)
		})
		return failure
	}
	capture := func(name string) error {
		if win.IsVisible() != visible {
			return errors.New("acceptance visibility changed")
		}
		var raw []byte
		var err error
		mygo.RunOnMain(func() { raw, err = win.CapturePage() })
		if err != nil {
			return err
		}
		img, err := png.Decode(bytes.NewReader(raw))
		if err != nil || img.Bounds().Dx() < 500 || img.Bounds().Dy() < 300 {
			return fmt.Errorf("invalid scene capture %s: %v", name, err)
		}
		return os.WriteFile(filepath.Join(filepath.Dir(output), name+".png"), raw, 0600)
	}
	if hasText(lines[count-1]) {
		return errors.New("file tail mounted before initial selection")
	}
	if err := clickRune(lines[0], 4, 0); err != nil {
		return err
	}
	mygo.RunOnMain(func() { list.ScrollTo(848, ui.Start) })
	frame()
	if err := clickRune(lines[850], 20, ui.Shift); err != nil {
		return err
	}
	want := append([]string{string([]rune(lines[0])[4:])}, lines[1:850]...)
	want = append(want, string([]rune(lines[850])[:20]))
	if err := copyExact(strings.Join(want, "\n"), "file first jump Shift selection spans 850 rows without visiting the middle and preserves rune endpoints"); err != nil {
		return err
	}
	if err := capture("native-text-file-selection"); err != nil {
		return err
	}
	mygo.RunOnMain(func() { driver.Key(ui.Cmd, ui.KeyA) })
	if err := copyExact(text, "file Select All copies all 1200 Unicode lines including trailing newline"); err != nil {
		return err
	}
	if hasText(lines[count-1]) {
		return errors.New("file Select All mounted its offscreen tail")
	}

	var clickErr error
	mygo.RunOnMain(func() {
		page = "diff"
		driver.Frame()
		clickErr = driver.Click(i18n.T(i18n.KeySurfacesDiffSplit))
	})
	if clickErr != nil {
		return clickErr
	}
	if hasText(added[count-1]) {
		return errors.New("split diff tail mounted before selection")
	}
	if err := clickRune(added[0], 4, 0); err != nil {
		return err
	}
	// One precise scroll event skips the middle of the patch. Locate a fully
	// visible ending row from the actual native layout, without mounting any
	// intermediate row or depending on platform font metrics.
	end := -1
	mygo.RunOnMain(func() {
		driver.Scroll(700, 480, 0, 17000)
		driver.Frame()
		for i := 700; i < count-1; i++ {
			_, y, ok := driver.SelectablePoint(old[i], 10)
			if ok && y > 300 && y < 450 {
				end = i
				break
			}
		}
	})
	if end < 700 {
		return errors.New("split diff scroll did not reach a distant visible endpoint")
	}
	if err := clickRune(old[end], 10, ui.Shift); err != nil {
		return err
	}
	want = []string{string([]rune(added[0])[4:])}
	for i := 1; i < end; i++ {
		want = append(want, old[i], added[i])
	}
	want = append(want, string([]rune(old[end])[:10]))
	if err := copyExact(strings.Join(want, "\n"), "split diff copies right-start to left-end across unvisited rows without leaking endpoint's right cell"); err != nil {
		return err
	}
	record["splitEndRow"] = end
	if err := capture("native-text-diff-selection"); err != nil {
		return err
	}
	mygo.RunOnMain(func() { driver.Key(ui.Cmd, ui.KeyA) })
	want = []string{"diff --git a/selection.go b/selection.go", "--- a/selection.go", "+++ b/selection.go"}
	for i := range old {
		want = append(want, old[i], added[i])
	}
	if err := copyExact(strings.Join(want, "\n"), "split diff Select All copies all old/new cells in display order and excludes controls/gutters"); err != nil {
		return err
	}
	if hasText(added[count-1]) {
		return errors.New("split diff Select All mounted its offscreen tail")
	}
	return nil
}
