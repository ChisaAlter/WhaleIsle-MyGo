//go:build windows

package networkstatus

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var ipHelper = windows.NewLazySystemDLL("iphlpapi.dll")
var notifyHints = ipHelper.NewProc("NotifyNetworkConnectivityHintChange")
var getHint = ipHelper.NewProc("GetNetworkConnectivityHint")
var cancelHints = ipHelper.NewProc("CancelMibChangeNotify2")
var subscriptions sync.Map
var nextSubscription atomic.Uintptr

// Only CallerContext is interpreted. On 386 the by-value 12-byte hint occupies
// three stack slots and WINAPI must pop all four arguments. The 64-bit ABI
// leaves aggregate transport to the caller; its unused hint is opaque here.
var hintCallback = func() uintptr {
	if runtime.GOARCH == "386" {
		return windows.NewCallback(func(token, _, _, _ uintptr) uintptr { signalHint(token); return 0 })
	}
	return windows.NewCallback(func(token, _ uintptr) uintptr { signalHint(token); return 0 })
}()

func signalHint(token uintptr) {
	if value, ok := subscriptions.Load(token); ok {
		select {
		case value.(chan struct{}) <- struct{}{}:
		default:
		}
	}
}

type connectivityHint struct {
	Level, Cost                                   uint32
	ApproachingLimit, OverLimit, Roaming, Padding byte
}

func readAvailability() (Availability, error) {
	var hint connectivityHint
	code, _, _ := getHint.Call(uintptr(unsafe.Pointer(&hint)))
	runtime.KeepAlive(&hint)
	if code != 0 {
		return Unknown, fmt.Errorf("network availability: GetNetworkConnectivityHint: %w", syscall.Errno(code))
	}
	switch hint.Level {
	case 1, 5:
		return Offline, nil // None / Hidden, as Chromium's modern Windows path.
	case 2, 3, 4:
		return Online, nil // Local / Internet / Constrained Internet access.
	default:
		return Unknown, nil
	}
}

// Watch subscribes before reading the initial local snapshot, so the caller
// receives an initial value before starting its connection loop. Later reads
// occur only on OS notifications. stop cancels and joins the OS subscription.
// Requires Windows 10 2004; absent APIs report Unknown instead of guessing.
func Watch(ctx context.Context, emit func(Availability)) (stop func() error, err error) {
	noop := func() error { return nil }
	if ctx.Err() != nil {
		return noop, ctx.Err()
	}
	for _, proc := range []*windows.LazyProc{notifyHints, getHint, cancelHints} {
		if err := proc.Find(); err != nil {
			emit(Unknown)
			return noop, fmt.Errorf("network availability notifications unavailable: %w", err)
		}
	}
	wake := make(chan struct{}, 1)
	token := nextSubscription.Add(1)
	subscriptions.Store(token, wake)
	var handle uintptr
	code, _, _ := notifyHints.Call(hintCallback, token, 0, uintptr(unsafe.Pointer(&handle)))
	runtime.KeepAlive(&handle)
	if code != 0 {
		subscriptions.Delete(token)
		emit(Unknown)
		return noop, fmt.Errorf("network availability: NotifyNetworkConnectivityHintChange: %w", syscall.Errno(code))
	}
	state, readErr := readAvailability()
	emit(state)
	watchCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	var closeErr error
	go func() {
		defer close(done)
		defer subscriptions.Delete(token)
		defer func() {
			// Not a native callback thread, and no resource its callback awaits is
			// held here: CancelMibChangeNotify2 can safely join native delivery.
			code, _, _ := cancelHints.Call(handle)
			if code != 0 {
				closeErr = fmt.Errorf("network availability: CancelMibChangeNotify2: %w", syscall.Errno(code))
			}
		}()
		for {
			select {
			case <-watchCtx.Done():
				return
			case <-wake:
				if watchCtx.Err() != nil {
					return
				}
				next, _ := readAvailability()
				if next != state {
					state = next
					emit(state)
				}
			}
		}
	}()
	return func() error { cancel(); <-done; return closeErr }, readErr
}
