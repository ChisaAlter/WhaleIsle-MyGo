package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"whaleisle/internal/desktop/notifications"
	"whaleisle/internal/paths"
)

const toastActivationFlag = "--whaleisle-toast-activation="
const maxToastActivationBytes = 48 << 10

type systemToastPayload struct {
	Version   int               `json:"version"`
	AppID     string            `json:"appID"`
	Arguments string            `json:"arguments"`
	UserInput map[string]string `json:"userInput"`
}

var systemToastState struct {
	sync.Mutex
	setup bool
	seen  map[string]bool
}

// receiveSystemToastActivation is called before the ordinary single-instance
// lock. Only the installer's exact server flag enters the bounded STA wait.
// It returns false when this invocation was a server request that cannot be
// received, so a stale server request does not cold-start an empty desktop.
func receiveSystemToastActivation() bool {
	server := false
	kept := []string{os.Args[0]}
	for _, arg := range os.Args[1:] {
		if arg == notifications.ToastServerFlag {
			server = true
			continue
		}
		if arg == "-Embedding" || arg == "/Embedding" {
			continue
		}
		kept = append(kept, arg)
	}
	if !server {
		return true
	}
	if runtime.GOOS != "windows" || isolatedDesktopRun() {
		return false
	}
	message, err := mygo.ReceiveNotificationActivation(mygo.NotificationActivatorOptions{CLSID: notifications.ToastActivatorCLSID, AppID: notifications.AppUserModelID}, 10*time.Second)
	if err != nil {
		log.Println("toast activation:", err)
		return false
	}
	payload := systemToastPayload{Version: 1, AppID: message.AppID, Arguments: message.Arguments, UserInput: message.UserInput}
	raw, err := json.Marshal(payload)
	if err != nil || len(raw) > maxToastActivationBytes {
		return false
	}
	if _, err = decodeSystemToastPayload(base64.RawURLEncoding.EncodeToString(raw)); err != nil {
		log.Println("toast activation:", err)
		return false
	}
	os.Args = append(kept, toastActivationFlag+base64.RawURLEncoding.EncodeToString(raw))
	return true
}

func decodeSystemToastPayload(encoded string) (systemToastPayload, error) {
	bad := func() (systemToastPayload, error) {
		return systemToastPayload{}, fmt.Errorf("invalid system toast activation payload")
	}
	if len(encoded) == 0 || len(encoded) > base64.RawURLEncoding.EncodedLen(maxToastActivationBytes) {
		return bad()
	}
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || len(raw) > maxToastActivationBytes {
		return bad()
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	var p systemToastPayload
	if err := d.Decode(&p); err != nil {
		return bad()
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return bad()
	}
	if p.Version != 1 || p.AppID != notifications.AppUserModelID || len(p.Arguments) == 0 || len(p.Arguments) > 16384 || !strings.HasPrefix(p.Arguments, "mygo-notification:") || strings.ContainsAny(p.Arguments, "\x00\r\n") || len(p.UserInput) > 128 {
		return bad()
	}
	for key, value := range p.UserInput {
		if len(key) == 0 || len(key) > 256 || len(value) > 4096 || strings.ContainsRune(key, 0) || strings.ContainsRune(value, 0) {
			return bad()
		}
	}
	return p, nil
}

// consumeSystemToastActivation handles inert data once. It never opens a URI or
// executes an argument. The existing notification listener owns navigation.
func consumeSystemToastActivation(args []string) bool {
	consumed := false
	for _, arg := range args {
		if !strings.HasPrefix(arg, toastActivationFlag) {
			continue
		}
		consumed = true
		encoded := strings.TrimPrefix(arg, toastActivationFlag)
		payload, err := decodeSystemToastPayload(encoded)
		if err != nil {
			log.Println("toast activation:", err)
			continue
		}
		systemToastState.Lock()
		if systemToastState.seen == nil {
			systemToastState.seen = map[string]bool{}
		}
		seen := systemToastState.seen[encoded]
		systemToastState.seen[encoded] = true
		systemToastState.Unlock()
		if seen || isolatedDesktopRun() {
			continue
		}
		mygo.DispatchNotificationActivation(mygo.NotificationActivationMessage{AppID: payload.AppID, Arguments: payload.Arguments, UserInput: payload.UserInput})
	}
	return consumed
}

// setupSystemNotifications is called by the existing support seam after the
// app is ready. It registers no product integration for isolated/portable runs.
func setupSystemNotifications() error {
	if runtime.GOOS != "windows" || isolatedDesktopRun() {
		return nil
	}
	systemToastState.Lock()
	if systemToastState.setup {
		systemToastState.Unlock()
		return nil
	}
	systemToastState.Unlock()
	err := desktopSystemIntegration(func() error {
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		shortcut := notifications.Shortcut{ProgramsDir: notifications.StartMenuProgramsDir(), Target: exe}
		if err := shortcut.Ensure(); err != nil {
			return err
		}
		if err := shortcut.RegisterActivator(); err != nil {
			return err
		}
		if _, err := shortcut.MigrateLegacy(filepath.Join(paths.UserDataDir(), "system-integration")); err != nil {
			log.Println("legacy notification shortcut:", err)
		}
		off, err := mygo.RegisterNotificationActivator(mygo.NotificationActivatorOptions{CLSID: notifications.ToastActivatorCLSID, AppID: notifications.AppUserModelID})
		if err != nil {
			return err
		}
		mygo.App.OnQuit(off)
		// A notification from an earlier process has no per-object listener.
		// Its only safe default is to reveal the existing desktop; current
		// notification listeners still own their specific, validated routes.
		mygo.App.OnNotificationClick(func(id string) { state.showMainWindow() })
		return nil
	})
	if err == nil {
		systemToastState.Lock()
		systemToastState.setup = true
		systemToastState.Unlock()
	}
	return err
}
