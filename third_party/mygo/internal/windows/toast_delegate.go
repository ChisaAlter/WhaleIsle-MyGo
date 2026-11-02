//go:build windows && (amd64 || arm64)

package windows

import (
	"sync"
	"syscall"
	"unsafe"

	"github.com/egoist/mygo/internal/platform"
)

// Unlike the permissive WebView2 handler, a WinRT delegate supports exactly
// IUnknown, IAgileObject and its parameterized delegate IID. It is safe to claim
// agility because it only copies arguments on the invoking apartment and posts
// plain Go data to the UI thread. It never retains an apartment-bound pointer.
type toastDelegate struct {
	vtbl *[4]uintptr
	refs uint32
	iid  GUID
	fn   func(sender, args uintptr)
}

var (
	toastDelegateIIDs = [3]GUID{guid("61c2402f-0ed0-5a18-ab69-59f4aa99a368"), guid("ab54de2d-97d9-5528-b6ad-105afe156530"), guid("95e3e803-c969-5e3a-9753-ea2ad22a9a33")}
	toastDelegateVtbl [4]uintptr
	toastDelegatesMu  sync.Mutex
	toastDelegates    = map[uintptr]*toastDelegate{}
)

func init() {
	toastDelegateVtbl = [4]uintptr{
		syscall.NewCallback(func(this, riid, out uintptr) uintptr {
			if out == 0 {
				return 0x80004003
			}
			*(*uintptr)(native(out)) = 0
			if riid == 0 {
				return 0x80004003
			}
			iid := *(*GUID)(native(riid))
			toastDelegatesMu.Lock()
			defer toastDelegatesMu.Unlock()
			d := toastDelegates[this]
			if d == nil {
				return eNoInterface
			}
			if iid != iunknownToastIID && iid != iidIAgileObject && iid != d.iid {
				return eNoInterface
			}
			d.refs++
			*(*uintptr)(native(out)) = this
			return sOK
		}),
		syscall.NewCallback(func(this uintptr) uintptr { return toastDelegateRef(this, true) }),
		syscall.NewCallback(func(this uintptr) uintptr { return toastDelegateRef(this, false) }),
		syscall.NewCallback(func(this, sender, args uintptr) uintptr {
			toastDelegatesMu.Lock()
			d := toastDelegates[this]
			toastDelegatesMu.Unlock()
			if d != nil {
				d.fn(sender, args)
			}
			return sOK
		}),
	}
}
func toastDelegateRef(this uintptr, add bool) uintptr {
	toastDelegatesMu.Lock()
	defer toastDelegatesMu.Unlock()
	d := toastDelegates[this]
	if d == nil {
		return 0
	}
	if add {
		d.refs++
	} else if d.refs > 0 {
		d.refs--
	}
	if d.refs == 0 {
		delete(toastDelegates, this)
	}
	return uintptr(d.refs)
}
func newToastDelegate(iid GUID, fn func(sender, args uintptr)) uintptr {
	d := &toastDelegate{vtbl: &toastDelegateVtbl, refs: 1, iid: iid, fn: fn}
	p := uintptr(unsafe.Pointer(d))
	toastDelegatesMu.Lock()
	toastDelegates[p] = d
	toastDelegatesMu.Unlock()
	return p
}

var (
	toastInputIterableIID = guid("fe2f3d47-5d47-5499-8374-430c7cda0204")
	toastPropertyValueIID = guid("4bd682dd-7554-40e9-9a9b-82654ede7e62")
)

func readToastActivation(args uintptr, n *platform.Notification) platform.NotificationActivation {
	result := platform.NotificationActivation{}
	a := queryInterface(args, &toastActivatedArgsIID)
	if a != 0 {
		var raw uintptr
		if hr := comCall(a, 6, uintptr(unsafe.Pointer(&raw))); !failed(hr) {
			result.Arguments = readHstring(raw)
			deleteHstring(raw)
		}
		release(a)
	}
	// The generated body launch argument is an internal ID, not a user action.
	if id, action := toastActivationParts(result.Arguments); id == n.ID {
		result.Arguments = action
	}
	a2 := queryInterface(args, &toastActivatedArgs2IID)
	if a2 == 0 {
		return result
	}
	defer release(a2)
	var inputs uintptr
	if hr := comCall(a2, 6, uintptr(unsafe.Pointer(&inputs))); failed(hr) || inputs == 0 {
		return result
	}
	defer release(inputs)
	iterable := queryInterface(inputs, &toastInputIterableIID)
	if iterable == 0 {
		return result
	}
	defer release(iterable)
	var iterator uintptr
	if hr := comCall(iterable, 6, uintptr(unsafe.Pointer(&iterator))); failed(hr) {
		return result
	}
	defer release(iterator)
	result.UserInput = map[string]string{}
	for count := 0; count < 128; count++ {
		var more byte
		if hr := comCall(iterator, 7, uintptr(unsafe.Pointer(&more))); failed(hr) || more == 0 {
			break
		}
		var pair uintptr
		if hr := comCall(iterator, 6, uintptr(unsafe.Pointer(&pair))); failed(hr) {
			break
		}
		var key, value uintptr
		keyHR := comCall(pair, 6, uintptr(unsafe.Pointer(&key)))
		valueHR := comCall(pair, 7, uintptr(unsafe.Pointer(&value)))
		if !failed(keyHR) && !failed(valueHR) && value != 0 {
			p := queryInterface(value, &toastPropertyValueIID)
			if p != 0 {
				var text uintptr
				if hr := comCall(p, 19, uintptr(unsafe.Pointer(&text))); !failed(hr) {
					result.UserInput[readHstring(key)] = readHstring(text)
					deleteHstring(text)
				}
				release(p)
			}
		}
		deleteHstring(key)
		release(value)
		release(pair)
		if hr := comCall(iterator, 8, uintptr(unsafe.Pointer(&more))); failed(hr) {
			break
		}
	}
	return result
}
