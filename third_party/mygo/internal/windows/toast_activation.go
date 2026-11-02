//go:build windows && (amd64 || arm64)

package windows

import (
	"errors"
	"fmt"
	"log"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"

	"github.com/egoist/mygo/internal/platform"
)

var (
	procCoRegisterClassObject = ole32.NewProc("CoRegisterClassObject")
	procCoRevokeClassObject   = ole32.NewProc("CoRevokeClassObject")
	toastActivatorIID         = guid("53e31837-6600-4a81-9395-75cffe746f94")
	toastClassFactoryIID      = guid("00000001-0000-0000-c000-000000000046")
	toastActivationVtbl       [4]uintptr
	toastClassFactoryVtbl     [5]uintptr
	toastServersMu            sync.Mutex
	toastServers              = map[uintptr]*toastServerObject{}
)

type toastServerObject struct {
	vtbl         *uintptr
	refs         uint32
	factory      bool
	registration *toastServer
}
type toastServer struct {
	mu                sync.Mutex
	closed            bool
	appID             string
	cookie            uint32
	factory, callback uintptr
	onActivate        func(platform.NotificationActivationMessage)
	oneShot, claimed  bool
}

func init() {
	query := syscall.NewCallback(func(this, riid, out uintptr) uintptr {
		if out == 0 || riid == 0 {
			return 0x80004003
		}
		*(*uintptr)(native(out)) = 0
		iid := *(*GUID)(native(riid))
		toastServersMu.Lock()
		defer toastServersMu.Unlock()
		o := toastServers[this]
		if o == nil {
			return eNoInterface
		}
		want := toastActivatorIID
		if o.factory {
			want = toastClassFactoryIID
		}
		if iid != iunknownToastIID && iid != want {
			return eNoInterface
		}
		o.refs++
		*(*uintptr)(native(out)) = this
		return sOK
	})
	add := syscall.NewCallback(func(this uintptr) uintptr { return toastServerRef(this, 1) })
	drop := syscall.NewCallback(func(this uintptr) uintptr { return toastServerRef(this, -1) })
	toastActivationVtbl = [4]uintptr{query, add, drop, syscall.NewCallback(func(this, appID, args, data uintptr, count uint32) uintptr {
		toastServersMu.Lock()
		o := toastServers[this]
		toastServersMu.Unlock()
		if o == nil {
			return 0x80004005
		}
		app, ok := boundedToastString(appID, 256)
		if !ok || app != o.registration.appID {
			return 0x80070057
		}
		argument, ok := boundedToastString(args, 16384)
		if !ok || count > 128 {
			return 0x80070057
		}
		message := platform.NotificationActivationMessage{AppID: app, Arguments: argument, UserInput: map[string]string{}}
		type pair struct{ Key, Value uintptr }
		if count != 0 && data == 0 {
			return 0x80070057
		}
		for _, p := range unsafe.Slice((*pair)(native(data)), int(count)) {
			key, k := boundedToastString(p.Key, 256)
			value, v := boundedToastString(p.Value, 4096)
			if !k || !v {
				return 0x80070057
			}
			message.UserInput[key] = value
		}
		o.registration.mu.Lock()
		defer o.registration.mu.Unlock()
		if o.registration.closed {
			return 0x800401FD /*CO_E_OBJNOTCONNECTED*/
		}
		if o.registration.oneShot && o.registration.claimed {
			return 0x8001010A /*RPC_E_SERVERCALL_RETRYLATER*/
		}
		o.registration.claimed = true
		o.registration.onActivate(message)
		return sOK
	})}
	toastClassFactoryVtbl = [5]uintptr{query, add, drop, syscall.NewCallback(func(this, outer, riid, out uintptr) uintptr {
		if out == 0 {
			return 0x80004003
		}
		*(*uintptr)(native(out)) = 0
		if outer != 0 {
			return 0x80040110
		}
		toastServersMu.Lock()
		o := toastServers[this]
		toastServersMu.Unlock()
		if o == nil {
			return 0x80004005
		}
		o.registration.mu.Lock()
		defer o.registration.mu.Unlock()
		if o.registration.closed || o.registration.callback == 0 {
			return 0x800401FD /*CO_E_OBJNOTCONNECTED*/
		}
		return comCall(o.registration.callback, 0, riid, out)
	}), syscall.NewCallback(func(this uintptr, lock int32) uintptr { return sOK })}
}
func boundedToastString(p uintptr, max int) (string, bool) {
	if p == 0 {
		return "", true
	}
	n := 0
	for ; n <= max; n++ {
		if *(*uint16)(unsafe.Add(native(p), n*2)) == 0 {
			return string(utf16.Decode(unsafe.Slice((*uint16)(native(p)), n))), true
		}
	}
	return "", false
}
func toastServerRef(p uintptr, delta int) uintptr {
	toastServersMu.Lock()
	defer toastServersMu.Unlock()
	o := toastServers[p]
	if o == nil {
		return 0
	}
	if delta > 0 {
		o.refs++
	} else if o.refs > 0 {
		o.refs--
	}
	if o.refs == 0 {
		delete(toastServers, p)
	}
	return uintptr(o.refs)
}
func newToastServerObject(server *toastServer, factory bool) uintptr {
	o := &toastServerObject{refs: 1, registration: server, factory: factory, vtbl: &toastActivationVtbl[0]}
	if factory {
		o.vtbl = &toastClassFactoryVtbl[0]
	}
	p := uintptr(unsafe.Pointer(o))
	toastServersMu.Lock()
	toastServers[p] = o
	toastServersMu.Unlock()
	return p
}
func registerToastServer(clsid, appID string, activate func(platform.NotificationActivationMessage)) (*toastServer, error) {
	if len(clsid) != 38 || clsid[0] != '{' || clsid[37] != '}' || len(appID) == 0 || len(appID) > 256 || strings.ContainsRune(appID, 0) {
		return nil, errors.New("mygo: invalid toast activator CLSID/AppID")
	}
	var id GUID
	parse := ole32.NewProc("CLSIDFromString")
	hr, _, _ := parse.Call(uintptr(unsafe.Pointer(u16(clsid))), uintptr(unsafe.Pointer(&id)))
	if failed(hr) {
		return nil, hresultError("parsing toast activator CLSID", hr)
	}
	s := &toastServer{appID: appID, onActivate: activate}
	s.callback = newToastServerObject(s, false)
	s.factory = newToastServerObject(s, true)
	hr, _, _ = procCoRegisterClassObject.Call(uintptr(unsafe.Pointer(&id)), s.factory, 4, 1, uintptr(unsafe.Pointer(&s.cookie))) // LOCAL_SERVER, MULTIPLEUSE
	if failed(hr) {
		release(s.factory)
		release(s.callback)
		return nil, hresultError("registering toast COM server", hr)
	}
	return s, nil
}
func (s *toastServer) close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	cookie := s.cookie
	s.mu.Unlock()
	if cookie != 0 {
		if hr, _, _ := procCoRevokeClassObject.Call(uintptr(cookie)); failed(hr) {
			// Registration and its references remain owned when revoke fails;
			// retained clients still fail closed instead of calling zero memory.
			log.Print(hresultError("revoking toast COM server", hr))
			return
		}
	}
	s.mu.Lock()
	factory, callback := s.factory, s.callback
	s.cookie, s.factory, s.callback = 0, 0, 0
	s.mu.Unlock()
	release(factory)
	release(callback)
}

func (b *Backend) RegisterNotificationActivator(clsid, appID string) (func(), error) {
	s, err := b.toastService()
	if err != nil {
		return nil, err
	}
	if s.server != nil {
		return nil, errors.New("mygo: toast activator already registered")
	}
	server, err := registerToastServer(clsid, appID, func(m platform.NotificationActivationMessage) {
		s.enqueue(toastEvent{kind: "com-activated", message: m})
	})
	if err != nil {
		return nil, err
	}
	s.server = server
	return func() {
		if s.server == server {
			server.close()
			s.server = nil
		}
	}, nil
}
func (b *Backend) ReceiveNotificationActivation(clsid, appID string, timeout time.Duration) (platform.NotificationActivationMessage, error) {
	if timeout <= 0 || timeout > 30*time.Second {
		return platform.NotificationActivationMessage{}, errors.New("mygo: toast activation wait must be between 0 and 30 seconds")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hr, _, _ := procCoInitializeEx.Call(0, coinitApartmentThreaded)
	if failed(hr) {
		return platform.NotificationActivationMessage{}, hresultError("initializing toast server STA", hr)
	}
	defer ole32.NewProc("CoUninitialize").Call()
	ch := make(chan platform.NotificationActivationMessage, 1)
	thread := currentThreadID()
	server, err := registerToastServer(clsid, appID, func(m platform.NotificationActivationMessage) {
		select {
		case ch <- m:
		default:
		}
		user32.NewProc("PostThreadMessageW").Call(uintptr(thread), 0, 0, 0)
	})
	if err != nil {
		return platform.NotificationActivationMessage{}, err
	}
	server.mu.Lock()
	server.oneShot = true
	server.mu.Unlock()
	defer server.close()
	// Ensure the thread owns a message queue before COM wakes it.
	var message msg
	procPeekMessageW.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0, 0)
	deadline := time.Now().Add(timeout)
	for {
		select {
		case result := <-ch:
			return result, nil
		default:
		}
		left := time.Until(deadline)
		if left <= 0 {
			return platform.NotificationActivationMessage{}, fmt.Errorf("mygo: toast COM activation timed out after %s", timeout)
		}
		procMsgWaitForMultipleObjectsEx.Call(0, 0, uintptr(max(1, left.Milliseconds())), qsAllInput, mwmoInputAvailable)
		for {
			r, _, _ := procPeekMessageW.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0, pmRemove)
			if r == 0 {
				break
			}
			procTranslateMessage.Call(uintptr(unsafe.Pointer(&message)))
			procDispatchMessageW.Call(uintptr(unsafe.Pointer(&message)))
		}
	}
}
func (b *Backend) DispatchNotificationActivation(m platform.NotificationActivationMessage) {
	if b.toast == nil || b.toast.server == nil || m.AppID != b.toast.server.appID {
		return
	}
	b.toast.dispatchActivation(m)
}
func (s *toastService) activationSeen(id string) bool {
	if s.activated == nil {
		s.activated = map[string]bool{}
	}
	if s.activated[id] {
		return true
	}
	s.activated[id] = true
	return false
}
func (s *toastService) dispatchActivation(m platform.NotificationActivationMessage) {
	id, action := toastActivationParts(m.Arguments)
	if id == "" || s.activationSeen(id) {
		return
	}
	if n := s.active[id]; n != nil {
		delete(s.active, id)
		n.detach()
	}
	if h, ok := s.b.h.(platform.NotificationObserver); ok {
		h.NotificationActivated(id, platform.NotificationActivation{Arguments: action, UserInput: m.UserInput})
	} else {
		s.b.h.NotificationClicked(id)
	}
}
