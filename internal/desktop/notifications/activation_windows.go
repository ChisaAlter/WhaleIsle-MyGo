//go:build windows

package notifications

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

// ToastActivatorCLSID belongs to the Go shell alone, separate from Electron.
const ToastActivatorCLSID = "{9CC711B1-3FB8-4C05-AB6C-CF32C5077B0F}"
const ToastServerFlag = "--whaleisle-toast-server"

var toastActivatorGUID = guid{0x9cc711b1, 0x3fb8, 0x4c05, [8]byte{0xab, 0x6c, 0xcf, 0x32, 0xc5, 0x07, 0x7b, 0x0f}}
var toastActivatorKey = propertyKey{FmtID: appUserModelIDKey.FmtID, PID: 26}

// RegisterActivator installs only the Go shell's own COM server. Call behind
// desktopSystemIntegration; this package has no registration side effects at init.
func (s Shortcut) RegisterActivator() error {
	if s.Target == "" || s.ProgramsDir == "" {
		return nil
	}
	if !strings.HasSuffix(strings.ToLower(s.Target), ".exe") || strings.ContainsAny(s.Target, "\x00\r\n\"") {
		return fmt.Errorf("notifications: invalid COM server executable")
	}
	if _, err := os.Stat(s.Target); err != nil {
		return err
	}
	serverPath := `Software\Classes\CLSID\` + ToastActivatorCLSID + `\LocalServer32`
	expected := `"` + s.Target + `" ` + ToastServerFlag
	if current, e := registry.OpenKey(registry.CURRENT_USER, serverPath, registry.QUERY_VALUE); e == nil {
		value, _, readErr := current.GetStringValue("")
		current.Close()
		if readErr != nil {
			return readErr
		}
		if !strings.EqualFold(value, expected) {
			return fmt.Errorf("notifications: preserving another install's COM activator")
		}
	} else if e != registry.ErrNotExist {
		return e
	}
	key, _, err := registry.CreateKey(registry.CURRENT_USER, serverPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	return key.SetStringValue("", `"`+s.Target+`" `+ToastServerFlag)
}

func setToastActivatorProperty(props *comPtr) error {
	value := propVariant{VT: 72 /*VT_CLSID*/, P: unsafe.Pointer(&toastActivatorGUID)}
	return hresultErr(props.call(6, uintptr(unsafe.Pointer(&toastActivatorKey)), uintptr(unsafe.Pointer(&value))))
}

// SetProcessIdentity makes a previously registered app's AUMID available to
// WinRT without touching any shortcut or notification history.
func SetProcessIdentity(appID string) error {
	id, err := syscall.UTF16PtrFromString(appID)
	if err != nil {
		return err
	}
	hr, _, _ := procSetProcessAppID.Call(uintptr(unsafe.Pointer(id)))
	return hresultErr(hr)
}
