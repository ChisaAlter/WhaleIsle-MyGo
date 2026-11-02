//go:build windows

package notifications

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unsafe"
)

type shortcutDetails struct {
	Target, Args, Icon, AppID string
	IconIndex                 int32
	Activator                 guid
	HasActivator              bool
}

func readShortcut(path string) (shortcutDetails, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hr, _, _ := procCoInitializeEx.Call(0, coinitApartment)
	if int32(hr) >= 0 {
		defer procCoUninitialize.Call()
	} else if uint32(hr) != 0x80010106 {
		return shortcutDetails{}, hresultErr(hr)
	}
	var p unsafe.Pointer
	hr, _, _ = procCoCreateInstance.Call(uintptr(unsafe.Pointer(&clsidShellLink)), 0, clsctxInprocServer, uintptr(unsafe.Pointer(&iidShellLinkW)), uintptr(unsafe.Pointer(&p)))
	if err := hresultErr(hr); err != nil {
		return shortcutDetails{}, err
	}
	link := (*comPtr)(p)
	defer link.release()
	file, err := link.query(&iidPersistFile)
	if err != nil {
		return shortcutDetails{}, err
	}
	defer file.release()
	name, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return shortcutDetails{}, err
	}
	if err := hresultErr(file.call(5, uintptr(unsafe.Pointer(name)), 0)); err != nil {
		return shortcutDetails{}, err
	}
	out := shortcutDetails{}
	buf := make([]uint16, 32768)
	if err := hresultErr(link.call(3, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)), 0, 4)); err != nil {
		return out, err
	}
	out.Target = syscall.UTF16ToString(buf)
	clear(buf)
	if err := hresultErr(link.call(10, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))); err != nil {
		return out, err
	}
	out.Args = syscall.UTF16ToString(buf)
	clear(buf)
	if err := hresultErr(link.call(16, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)), uintptr(unsafe.Pointer(&out.IconIndex)))); err != nil {
		return out, err
	}
	out.Icon = syscall.UTF16ToString(buf)
	store, err := link.query(&iidPropertyStore)
	if err != nil {
		return out, err
	}
	defer store.release()
	var value propVariant
	if err := hresultErr(store.call(5, uintptr(unsafe.Pointer(&appUserModelIDKey)), uintptr(unsafe.Pointer(&value)))); err != nil {
		return out, err
	}
	defer procPropVariantClear.Call(uintptr(unsafe.Pointer(&value)))
	if value.VT == vtLPWStr && value.P != nil {
		out.AppID = syscall.UTF16ToString(unsafe.Slice((*uint16)(value.P), zeroTerminatedLength((*uint16)(value.P))))
	}
	var activator propVariant
	if err := hresultErr(store.call(5, uintptr(unsafe.Pointer(&toastActivatorKey)), uintptr(unsafe.Pointer(&activator)))); err != nil {
		return out, err
	}
	defer procPropVariantClear.Call(uintptr(unsafe.Pointer(&activator)))
	if activator.VT == 72 && activator.P != nil {
		out.Activator = *(*guid)(activator.P)
		out.HasActivator = true
	}
	return out, nil
}
func zeroTerminatedLength(p *uint16) int {
	for i := 0; i < 32768; i++ {
		if *(*uint16)(unsafe.Add(unsafe.Pointer(p), i*2)) == 0 {
			return i
		}
	}
	return 32768
}
func sameShortcutTarget(a, b string) bool {
	if !filepath.IsAbs(a) || !filepath.IsAbs(b) {
		return false
	}
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}

// MigrateLegacy moves only Electron.lnk with the same executable, AppID and
// uncustomized launch/icon fields. The original bytes are preserved in a unique
// app-owned backup. Other products, installed Electron links and pins are never
// scanned. A different Electron target intentionally remains untouched.
func (s Shortcut) MigrateLegacy(backupRoot string) (string, error) {
	if s.ProgramsDir == "" || s.Target == "" || !filepath.IsAbs(backupRoot) {
		return "", nil
	}
	legacy := filepath.Join(s.ProgramsDir, "Electron.lnk")
	before, err := os.Lstat(legacy)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if !before.Mode().IsRegular() {
		return "", nil
	}
	matches := func(d shortcutDetails) bool {
		return d.AppID == AppUserModelID && sameShortcutTarget(d.Target, s.Target) && d.Args == "" && d.IconIndex == 0 && (d.Icon == "" || sameShortcutTarget(d.Icon, d.Target))
	}
	details, err := readShortcut(legacy)
	if err != nil || !matches(details) {
		return "", err
	}
	root := filepath.Join(filepath.Clean(backupRoot), "legacy-system-shortcuts")
	if err := os.MkdirAll(root, 0700); err != nil {
		return "", err
	}
	rootReal, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	parentReal, err := filepath.EvalSymlinks(filepath.Clean(backupRoot))
	if err != nil {
		return "", err
	}
	if !strings.EqualFold(rootReal, filepath.Join(parentReal, "legacy-system-shortcuts")) {
		return "", fmt.Errorf("notifications: unsafe legacy backup directory")
	}
	dir, err := os.MkdirTemp(root, "notification-")
	if err != nil {
		return "", err
	}
	backup := filepath.Join(dir, "Electron.lnk.backup")
	current, err := os.Lstat(legacy)
	if err != nil {
		return "", err
	}
	again, err := readShortcut(legacy)
	if err != nil {
		return "", err
	}
	if !os.SameFile(before, current) || before.Size() != current.Size() || !before.ModTime().Equal(current.ModTime()) || !matches(again) {
		return "", fmt.Errorf("notifications: legacy shortcut changed while inspecting")
	}
	// Both absolute paths are explicit: source is the one inspected above and
	// destination is new, inside the verified app-owned backup directory.
	if err := os.Rename(legacy, backup); err != nil {
		return "", err
	}
	shortcutShell32.NewProc("SHChangeNotify").Call(4, 0x2005, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(legacy))), 0)
	return backup, nil
}
