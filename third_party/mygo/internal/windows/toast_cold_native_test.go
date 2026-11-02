//go:build windows && (amd64 || arm64)

package windows

import (
	"crypto/rand"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
	"unsafe"
)

func TestToastColdCOMNativeActivation(t *testing.T) {
	fixture := os.Getenv("MYGO_TOAST_COLD_FIXTURE")
	if fixture == "" {
		t.Skip("build windowsgui toastfixture and set MYGO_TOAST_COLD_FIXTURE")
	}
	if !filepath.IsAbs(fixture) {
		t.Fatal("fixture path must be absolute")
	}
	if _, err := os.Stat(fixture); err != nil {
		t.Fatal(err)
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hr, _, _ := procCoInitializeEx.Call(0, coinitApartmentThreaded)
	if failed(hr) {
		t.Fatal(hresultError("test STA", hr))
	}
	defer ole32.NewProc("CoUninitialize").Call()
	var clsid GUID
	hr, _, _ = ole32.NewProc("CoCreateGuid").Call(uintptr(unsafe.Pointer(&clsid)))
	if failed(hr) {
		t.Fatal(hresultError("fixture GUID", hr))
	}
	var chars [39]uint16
	ole32.NewProc("StringFromGUID2").Call(uintptr(unsafe.Pointer(&clsid)), uintptr(unsafe.Pointer(&chars[0])), 39)
	clsidText := string(utf16Decode(chars[:38]))
	appID := "MyGo.ColdFixture." + rand.Text()
	out := filepath.Join(t.TempDir(), "activation.json")
	key := `Software\Classes\CLSID\` + clsidText
	if _, exists := regGet(hkeyCurrentUser, key+`\LocalServer32`, ""); exists {
		t.Fatal("fixture CLSID already exists")
	}
	command := `"` + fixture + `" "` + clsidText + `" "` + appID + `" "` + out + `"`
	if err := regSet(hkeyCurrentUser, key+`\LocalServer32`, "", command); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := regDeleteTree(hkeyCurrentUser, key); err != nil {
			t.Error(err)
		}
	}()
	var callback uintptr
	hr, _, _ = procCoCreateInstance.Call(uintptr(unsafe.Pointer(&clsid)), 0, 4, uintptr(unsafe.Pointer(&toastActivatorIID)), uintptr(unsafe.Pointer(&callback)))
	if failed(hr) {
		t.Fatal(hresultError("cold CoCreateInstance", hr))
	}
	type pair struct{ Key, Value *uint16 }
	data := pair{u16("reply"), u16("cold answer")}
	args := toastActionLaunch("cold-notification", "reply")
	hr = comCall(callback, 3, uintptr(unsafe.Pointer(u16(appID))), uintptr(unsafe.Pointer(u16(args))), uintptr(unsafe.Pointer(&data)), 1)
	release(callback)
	if failed(hr) {
		t.Fatal(hresultError("cold Activate", hr))
	}
	deadline := time.Now().Add(5 * time.Second)
	var raw []byte
	for {
		var err error
		raw, err = os.ReadFile(out)
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("cold fixture output: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}
	var result struct {
		AppID, Arguments string
		UserInput        map[string]string
		Error            string
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	if result.Error != "" || result.AppID != appID || result.Arguments != args || result.UserInput["reply"] != "cold answer" {
		t.Fatalf("cold result=%+v", result)
	}
	t.Log("separate windowsgui COM server cold launch, bounded STA receive, action and input passed")
}
