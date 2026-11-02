//go:build windows && (amd64 || arm64)

package windows

import (
	"os"
	"path/filepath"
	"testing"
	"unsafe"
)

// registerToastFixture installs only a unique fixture-owned link in a unique
// directory. Windows' notification broker must discover a real Start-menu
// AUMID; a temp profile's lookalike Programs path is not a registration.
func registerToastFixture(t *testing.T, appID string) {
	t.Helper()
	programs := filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs")
	dir := filepath.Join(programs, appID)
	if _, err := os.Lstat(dir); !os.IsNotExist(err) {
		t.Fatalf("fixture path must not exist: %s", dir)
	}
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "MyGo isolated fixture.lnk")
	t.Cleanup(func() {
		// Exact fixture paths only, never RemoveAll or product/user links.
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			t.Error(err)
		}
		if err := os.Remove(dir); err != nil {
			t.Error(err)
		}
	})
	clsid := guid("00021401-0000-0000-c000-000000000046")
	shellIID := guid("000214f9-0000-0000-c000-000000000046")
	persistIID := guid("0000010b-0000-0000-c000-000000000046")
	propertyIID := guid("886d8eeb-8cf2-4446-8d02-cdba1dbdcf99")
	var link uintptr
	hr, _, _ := procCoCreateInstance.Call(uintptr(unsafe.Pointer(&clsid)), 0, 1, uintptr(unsafe.Pointer(&shellIID)), uintptr(unsafe.Pointer(&link)))
	if failed(hr) {
		t.Fatal(hresultError("fixture shell link", hr))
	}
	defer release(link)
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if hr := comCall(link, 20, uintptr(unsafe.Pointer(u16(exe)))); failed(hr) {
		t.Fatal(hresultError("fixture link target", hr))
	}
	store := queryInterface(link, &propertyIID)
	if store == 0 {
		t.Fatal("fixture property store")
	}
	defer release(store)
	key := struct {
		FMTID GUID
		PID   uint32
	}{guid("9f4c2855-9f79-4b39-a8d0-e1d42de1d5f3"), 5}
	value := struct {
		VT, R1, R2, R3 uint16
		P              *uint16
		Pad            uint64
	}{VT: 31, P: u16(appID)}
	if hr := comCall(store, 6, uintptr(unsafe.Pointer(&key)), uintptr(unsafe.Pointer(&value))); failed(hr) {
		t.Fatal(hresultError("fixture link AUMID", hr))
	}
	if hr := comCall(store, 7); failed(hr) {
		t.Fatal(hresultError("fixture link commit", hr))
	}
	file := queryInterface(link, &persistIID)
	if file == 0 {
		t.Fatal("fixture persistence")
	}
	defer release(file)
	if hr := comCall(file, 6, uintptr(unsafe.Pointer(u16(path))), 1); failed(hr) {
		t.Fatal(hresultError("fixture link save", hr))
	}
	setID := shell32.NewProc("SetCurrentProcessExplicitAppUserModelID")
	if hr, _, _ := setID.Call(uintptr(unsafe.Pointer(u16(appID)))); failed(hr) {
		t.Fatal(hresultError("fixture process identity", hr))
	}
	// Only this new fixture path is announced to Shell discovery.
	shell32.NewProc("SHChangeNotify").Call(2, 0x1005, uintptr(unsafe.Pointer(u16(path))), 0)
}
