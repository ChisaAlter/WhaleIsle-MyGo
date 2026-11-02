//go:build windows

package notifications

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"unsafe"
)

var (
	ole32                = syscall.NewLazyDLL("ole32.dll")
	procCoInitializeEx   = ole32.NewProc("CoInitializeEx")
	procCoUninitialize   = ole32.NewProc("CoUninitialize")
	procCoCreateInstance = ole32.NewProc("CoCreateInstance")
	procPropVariantClear = ole32.NewProc("PropVariantClear")
	shortcutShell32      = syscall.NewLazyDLL("shell32.dll")
	procSetProcessAppID  = shortcutShell32.NewProc("SetCurrentProcessExplicitAppUserModelID")
)

const (
	clsctxInprocServer = 0x1
	coinitApartment    = 0x2
	vtLPWStr           = 31
)

type guid struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

type propertyKey struct {
	FmtID guid
	PID   uint32
}

// propVariant is PROPVARIANT: vt, three reserved words, then the 16-byte union
// (a pointer here), so the struct keeps the native 24-byte size.
type propVariant struct {
	VT         uint16
	R1, R2, R3 uint16
	P          unsafe.Pointer
	Pad        uint64
}

var (
	clsidShellLink    = guid{0x00021401, 0, 0, [8]byte{0xC0, 0, 0, 0, 0, 0, 0, 0x46}}
	iidShellLinkW     = guid{0x000214F9, 0, 0, [8]byte{0xC0, 0, 0, 0, 0, 0, 0, 0x46}}
	iidPersistFile    = guid{0x0000010B, 0, 0, [8]byte{0xC0, 0, 0, 0, 0, 0, 0, 0x46}}
	iidPropertyStore  = guid{0x886D8EEB, 0x8CF2, 0x4446, [8]byte{0x8D, 0x02, 0xCD, 0xBA, 0x1D, 0xBD, 0xCF, 0x99}}
	appUserModelIDKey = propertyKey{
		FmtID: guid{0x9F4C2855, 0x9F79, 0x4B39, [8]byte{0xA8, 0xD0, 0xE1, 0xD4, 0x2D, 0xE1, 0xD5, 0xF3}},
		PID:   5, // PKEY_AppUserModel_ID
	}
)

// comPtr calls one COM vtable slot; the first three slots are QueryInterface,
// AddRef and Release on every interface.
type comPtr struct{ vtbl *[32]uintptr }

//go:uintptrescapes
func (c *comPtr) call(index int, args ...uintptr) uintptr {
	all := append([]uintptr{uintptr(unsafe.Pointer(c))}, args...)
	hr, _, _ := syscall.SyscallN((*c.vtbl)[index], all...)
	return hr
}

func hresultErr(hr uintptr) error {
	if int32(hr) >= 0 {
		return nil
	}
	return fmt.Errorf("notifications: COM 0x%08X", uint32(hr))
}

func (c *comPtr) query(iid *guid) (*comPtr, error) {
	var out unsafe.Pointer
	hr := c.call(0, uintptr(unsafe.Pointer(iid)), uintptr(unsafe.Pointer(&out)))
	if err := hresultErr(hr); err != nil {
		return nil, err
	}
	return (*comPtr)(out), nil
}

func (c *comPtr) release() {
	if c != nil {
		c.call(2)
	}
}

// createShortcut writes (or overwrites) the Start-menu link and sets its
// System.AppUserModel.ID, the identity Windows uses to name toasts.
func createShortcut(lnk, target, appID string) error {
	if lnk != "" {
		if info, err := os.Lstat(lnk); err == nil {
			if !info.Mode().IsRegular() {
				return fmt.Errorf("notifications: existing shortcut is not a regular file")
			}
			d, err := readShortcut(lnk)
			if err != nil {
				return err
			}
			if d.AppID != appID || !sameShortcutTarget(d.Target, target) || d.Args != "" || d.IconIndex != 0 || (d.Icon != "" && !sameShortcutTarget(d.Icon, d.Target)) || (d.HasActivator && d.Activator != toastActivatorGUID) {
				return fmt.Errorf("notifications: preserving an existing foreign or customized shortcut")
			}
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	return writeShortcut(lnk, target, appID, true)
}

func writeShortcut(lnk, target, appID string, setProcessID bool) error {
	if lnk == "" || target == "" || appID == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(lnk), 0o755); err != nil {
		return err
	}
	// Every interface below belongs to this apartment for its full lifetime.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	initHR, _, _ := procCoInitializeEx.Call(0, coinitApartment)
	if int32(initHR) >= 0 {
		defer procCoUninitialize.Call()
	} else if uint32(initHR) != 0x80010106 /* RPC_E_CHANGED_MODE */ {
		return hresultErr(initHR)
	}

	var unknown unsafe.Pointer
	hr, _, _ := procCoCreateInstance.Call(
		uintptr(unsafe.Pointer(&clsidShellLink)), 0, clsctxInprocServer,
		uintptr(unsafe.Pointer(&iidShellLinkW)), uintptr(unsafe.Pointer(&unknown)),
	)
	if err := hresultErr(hr); err != nil {
		return err
	}
	link := (*comPtr)(unknown)
	defer link.release()

	target16, err := syscall.UTF16PtrFromString(target)
	if err != nil {
		return err
	}
	if hr := link.call(20 /* SetPath */, uintptr(unsafe.Pointer(target16))); hresultErr(hr) != nil {
		return hresultErr(hr)
	}
	if hr := link.call(17 /* SetIconLocation */, uintptr(unsafe.Pointer(target16)), 0); hresultErr(hr) != nil {
		return hresultErr(hr)
	}

	file, err := link.query(&iidPersistFile)
	if err != nil {
		return err
	}
	defer file.release()
	lnk16, err := syscall.UTF16PtrFromString(lnk)
	if err != nil {
		return err
	}
	if hr := file.call(6 /* Save */, uintptr(unsafe.Pointer(lnk16)), 1); hresultErr(hr) != nil {
		return hresultErr(hr)
	}

	props, err := link.query(&iidPropertyStore)
	if err != nil {
		return err
	}
	defer props.release()
	id16, err := syscall.UTF16PtrFromString(appID)
	if err != nil {
		return err
	}
	// AUMID is process-wide and is needed by CreateToastNotifierWithId.
	// Set it only after link creation succeeded so failures are observable.
	if setProcessID {
		if hr, _, _ := procSetProcessAppID.Call(uintptr(unsafe.Pointer(id16))); hresultErr(hr) != nil {
			return hresultErr(hr)
		}
	}
	value := propVariant{VT: vtLPWStr, P: unsafe.Pointer(id16)}
	if hr := props.call(6 /* SetValue */, uintptr(unsafe.Pointer(&appUserModelIDKey)), uintptr(unsafe.Pointer(&value))); hresultErr(hr) != nil {
		return hresultErr(hr)
	}
	if setProcessID {
		if err := setToastActivatorProperty(props); err != nil {
			return err
		}
	}
	if hr := props.call(7 /* Commit */); hresultErr(hr) != nil {
		return hresultErr(hr)
	}
	// The property store updates the link in memory; persist once more so the
	// AUMID is written into the .lnk next to the target path.
	if hr := file.call(6, uintptr(unsafe.Pointer(lnk16)), 1); hresultErr(hr) != nil {
		return hresultErr(hr)
	}
	return nil
}
