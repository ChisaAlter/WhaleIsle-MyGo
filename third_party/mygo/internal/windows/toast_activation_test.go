//go:build windows && (amd64 || arm64)

package windows

import (
	"crypto/rand"
	"os"
	"runtime"
	"testing"
	"unsafe"

	"github.com/egoist/mygo/internal/platform"
)

func TestToastCOMNativeActivation(t *testing.T) {
	if os.Getenv("MYGO_TOAST_NATIVE_TEST") != "1" {
		t.Skip("set MYGO_TOAST_NATIVE_TEST=1")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hr, _, _ := procCoInitializeEx.Call(0, coinitApartmentThreaded)
	if failed(hr) {
		t.Fatal(hresultError("test COM", hr))
	}
	defer ole32.NewProc("CoUninitialize").Call()
	clsid := "{A1BFBA63-F81E-4E4A-A6E1-29C032F0F7A9}"
	appID := "MyGo.COMFixture." + rand.Text()
	var received platform.NotificationActivationMessage
	s, err := registerToastServer(clsid, appID, func(m platform.NotificationActivationMessage) { received = m })
	if err != nil {
		t.Fatal(err)
	}
	defer s.close()
	id := guid("a1bfba63-f81e-4e4a-a6e1-29c032f0f7a9")
	var callback uintptr
	hr, _, _ = procCoCreateInstance.Call(uintptr(unsafe.Pointer(&id)), 0, 4, uintptr(unsafe.Pointer(&toastActivatorIID)), uintptr(unsafe.Pointer(&callback)))
	if failed(hr) {
		t.Fatal(hresultError("CoCreateInstance notification activator", hr))
	}
	defer release(callback)
	type pair struct{ Key, Value *uint16 }
	data := pair{u16("reply"), u16("回答")}
	args := toastActionLaunch("fixture-notification", "reply")
	if hr := comCall(callback, 3, uintptr(unsafe.Pointer(u16(appID))), uintptr(unsafe.Pointer(u16(args))), uintptr(unsafe.Pointer(&data)), 1); failed(hr) {
		t.Fatal(hresultError("Activate", hr))
	}
	if received.AppID != appID || received.Arguments != args || received.UserInput["reply"] != "回答" {
		t.Fatalf("activation=%+v", received)
	}
	if hr := comCall(callback, 3, uintptr(unsafe.Pointer(u16("other-app"))), uintptr(unsafe.Pointer(u16(args))), 0, 0); !failed(hr) {
		t.Fatal("foreign appID accepted")
	}
	s.mu.Lock()
	s.oneShot = true
	s.mu.Unlock()
	if hr := comCall(callback, 3, uintptr(unsafe.Pointer(u16(appID))), uintptr(unsafe.Pointer(u16(args))), 0, 0); uint32(hr) != 0x8001010A {
		t.Fatalf("burst activation was silently acknowledged: %#x", hr)
	}
	// COM clients can retain both interfaces after CoRevokeClassObject.
	factory := s.factory
	addRef(factory)
	defer release(factory)
	s.close()
	if hr := comCall(callback, 3, uintptr(unsafe.Pointer(u16(appID))), uintptr(unsafe.Pointer(u16(args))), 0, 0); uint32(hr) != 0x800401FD {
		t.Fatalf("retained callback after close: %#x", hr)
	}
	var late uintptr
	if hr := comCall(factory, 3, 0, uintptr(unsafe.Pointer(&toastActivatorIID)), uintptr(unsafe.Pointer(&late))); uint32(hr) != 0x800401FD || late != 0 {
		t.Fatalf("retained factory after close: %#x out=%#x", hr, late)
	}
	t.Log("real COM class factory, activation ABI and retained references fail closed after revoke")
}

func TestToastActivationEnvelope(t *testing.T) {
	id, action := toastActivationParts(toastActionLaunch("chat:&action=字", "open&reply"))
	if id != "chat:&action=字" || action != "open&reply" {
		t.Fatalf("%q %q", id, action)
	}
	id, action = toastActivationParts(toastLaunch("body"))
	if id != "body" || action != "" {
		t.Fatalf("%q %q", id, action)
	}
}
