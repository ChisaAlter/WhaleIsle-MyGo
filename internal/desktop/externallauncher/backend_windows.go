//go:build windows

package externallauncher

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	"whaleisle/internal/desktop/appuninstall"
	"whaleisle/internal/desktop/importer"
	"whaleisle/internal/desktop/update"
	"whaleisle/internal/paths"
	"whaleisle/internal/product"
)

func Native() *Controller {
	return New(Dependencies{Discover: Discover, Running: running, Start: func(i Installation) error { return dispatch(i) }, RequestStop: func(i Installation) error { return dispatch(i, StopArgument, i.Executable) }, Maintenance: func() (func(), error) { return importer.AcquireExternalMaintenance(paths.DshHome()) }, Install: update.StartStagedAndWait})
}
func Discover() (Installation, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, appuninstall.RegistryKey, registry.QUERY_VALUE)
	if err != nil {
		if errors.Is(err, windows.ERROR_FILE_NOT_FOUND) {
			return Installation{}, ErrNotInstalled
		}
		return Installation{}, err
	}
	defer key.Close()
	read := func(name string) string {
		v, kind, err := key.GetStringValue(name)
		if err != nil || kind != registry.SZ {
			return ""
		}
		return v
	}
	displayName := read("DisplayName")
	if read("NativeRuntime") != appuninstall.NativeMarker || (displayName != product.Name && displayName != "Whale Isle") {
		return Installation{}, ErrNotInstalled
	}
	root := read("InstallLocation")
	version := read("DisplayVersion")
	if !filepath.IsAbs(root) || version == "" {
		return Installation{}, ErrNotInstalled
	}
	root = filepath.Clean(root)
	if read("UninstallString") != `"`+filepath.Join(root, "Uninstall.exe")+`"` {
		return Installation{}, ErrNotInstalled
	}
	i := Installation{Root: root, Executable: filepath.Join(root, "whaleisle.exe"), Version: version}
	if err := validate(i); err != nil {
		return Installation{}, err
	}
	return i, nil
}
func validate(i Installation) error {
	for _, p := range []string{i.Root, i.Executable} {
		st, err := os.Lstat(p)
		if err != nil {
			return err
		}
		if p == i.Executable && !st.Mode().IsRegular() {
			return ErrNotInstalled
		}
		actual, err := filepath.EvalSymlinks(p)
		if err != nil {
			return err
		}
		if !strings.EqualFold(filepath.Clean(actual), p) {
			return ErrNotInstalled
		}
	}
	return nil
}
func dispatch(i Installation, args ...string) error {
	fresh, err := Discover()
	if err != nil {
		return err
	}
	if fresh != i {
		return errors.New("installed desktop changed; refresh and retry")
	}
	cmd := exec.Command(i.Executable, args...)
	cmd.Dir = i.Root
	if err = cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

// Match full image identity; never use a name-only match to stop another install.
func running(i Installation) (bool, error) {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return false, err
	}
	defer windows.CloseHandle(snap)
	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	for err = windows.Process32First(snap, &entry); err == nil; err = windows.Process32Next(snap, &entry) {
		if !strings.EqualFold(windows.UTF16ToString(entry.ExeFile[:]), "whaleisle.exe") {
			continue
		}
		process, e := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, entry.ProcessID)
		if e != nil {
			if errors.Is(e, windows.ERROR_INVALID_PARAMETER) {
				continue
			}
			return false, e
		}
		buf := make([]uint16, 32768)
		size := uint32(len(buf))
		e = windows.QueryFullProcessImageName(process, 0, &buf[0], &size)
		windows.CloseHandle(process)
		if e != nil {
			return false, e
		}
		if strings.EqualFold(filepath.Clean(windows.UTF16ToString(buf[:size])), i.Executable) {
			return true, nil
		}
	}
	if !errors.Is(err, windows.ERROR_NO_MORE_FILES) {
		return false, err
	}
	return false, nil
}
