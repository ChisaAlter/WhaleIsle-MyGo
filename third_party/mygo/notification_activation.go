package mygo

import (
	"github.com/egoist/mygo/internal/platform"
	"time"
)

// NotificationActivatorOptions identifies an unpackaged Windows COM activator.
// The installer registers its CLSID and AppID; these calls never write registry
// or shortcuts. Every caller must use its own CLSID instead of another app's.
type NotificationActivatorOptions struct{ CLSID, AppID string }

// RegisterNotificationActivator serves notification-center activations while
// the app runs. It is safe from any goroutine. App.OnNotificationClick receives
// the original generated notification ID, including notifications of older runs.
// The returned unregister function is idempotent; app shutdown also unregisters.
func RegisterNotificationActivator(o NotificationActivatorOptions) (func(), error) {
	needsApp("RegisterNotificationActivator")
	type result struct {
		off func()
		err error
	}
	r := onMainValue(func() result {
		b, ok := backend().(platform.NotificationActivatorBackend)
		if !ok {
			return result{err: platform.ErrUnsupported}
		}
		off, err := b.RegisterNotificationActivator(o.CLSID, o.AppID)
		return result{off, err}
	})
	if r.err != nil {
		return nil, r.err
	}
	return func() { onMain(r.off) }, nil
}

// ReceiveNotificationActivation runs a bounded COM message pump before Run in
// a dedicated --toast-server invocation. Call it only when the installer-owned
// server flag is present, before taking the app's single-instance lock. It does
// no filesystem/registry work. On timeout it returns an error, not a fabricated
// click. The result can be passed through the app's existing single-instance IPC.
func ReceiveNotificationActivation(o NotificationActivatorOptions, timeout time.Duration) (NotificationActivationMessage, error) {
	b, ok := backend().(platform.NotificationActivationReceiver)
	if !ok {
		return NotificationActivationMessage{}, platform.ErrUnsupported
	}
	m, err := b.ReceiveNotificationActivation(o.CLSID, o.AppID, timeout)
	return NotificationActivationMessage{AppID: m.AppID, Arguments: m.Arguments, UserInput: m.UserInput}, err
}

// NotificationActivationMessage is inert activation data suitable for an app's
// single-instance transport. It never executes a URI or command.
type NotificationActivationMessage struct {
	AppID, Arguments string
	UserInput        map[string]string
}

// DispatchNotificationActivation dispatches a previously received and validated
// activation on the main thread. The app must check its identity before calling.
func DispatchNotificationActivation(m NotificationActivationMessage) {
	needsApp("DispatchNotificationActivation")
	onMain(func() {
		b, ok := backend().(platform.NotificationActivationDispatcher)
		if ok {
			b.DispatchNotificationActivation(platform.NotificationActivationMessage{AppID: m.AppID, Arguments: m.Arguments, UserInput: m.UserInput})
		}
	})
}
