package mygo

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/egoist/mygo/internal/platform"
)

// ToastOptions customizes a Windows WinRT toast. Other platforms return
// ErrUnsupported when this option is set; ordinary NotificationOptions work on
// every supported desktop. XML accepts the complete Windows toast schema,
// including adaptive groups, inputs, actions, progress, audio and attribution.
// When XML is empty, MyGo builds a ToastGeneric template from the notification.
// AppID overrides the process AppUserModelID and must identify a registered
// application. MyGo never creates or deletes Start-menu links implicitly.
type ToastOptions struct {
	AppID         string
	XML           string
	Icon          string // an absolute file URI or https URI, shown as appLogoOverride
	Hero          string // an absolute file URI or https URI
	Actions       []ToastAction
	Scenario      string // empty, default, reminder, alarm or incomingCall
	Duration      string // empty, short or long
	SuppressPopup bool   // deliver to notification history without a banner
}

// ToastAction is a foreground action button. Arguments reaches OnAction.
type ToastAction struct{ Label, Arguments string }

// NotificationActivation is the arguments of a toast body or action click.
// UserInput contains values submitted by inputs in a custom XML template.
type NotificationActivation struct {
	Arguments string
	UserInput map[string]string
}

// NotificationCloseReason distinguishes a user dismissal, an expired banner,
// and an explicit application removal. A timeout leaves the toast in Windows
// notification history and its click listener remains active.
type NotificationCloseReason string

const (
	NotificationDismissed NotificationCloseReason = "dismissed"
	NotificationTimedOut  NotificationCloseReason = "timed-out"
	NotificationRemoved   NotificationCloseReason = "removed"
	NotificationActivated NotificationCloseReason = "activated"
)

// NotificationRecord is a Windows notification still in notification history.
// ID is recovered from MyGo's launch argument; custom XML may leave it empty.
type NotificationRecord struct{ ID, Tag, Group, XML string }

// NotificationHistory reads the current app's delivered notifications without
// changing them. appID is the registered Windows identity; empty uses the
// process identity. Other platforms return ErrUnsupported.
func NotificationHistory(appID string) ([]NotificationRecord, error) {
	needsApp("NotificationHistory")
	type result struct {
		rows []NotificationRecord
		err  error
	}
	r := onMainValue(func() result {
		b, ok := backend().(platform.NotificationHistoryBackend)
		if !ok {
			return result{err: platform.ErrUnsupported}
		}
		rows, err := b.NotificationHistory(appID)
		out := make([]NotificationRecord, len(rows))
		for i, row := range rows {
			out[i] = NotificationRecord{row.ID, row.Tag, row.Group, row.XML}
		}
		return result{out, err}
	})
	return r.rows, r.err
}

// OnAction receives foreground activation arguments (also for a body click).
// OnClick follows it and retains its existing body/action click semantics.
func (n *Notification) OnAction(fn func(NotificationActivation)) (off func()) {
	return n.onAction.add(fn, false)
}

// OnClose receives native banner dismissal, expiry, activation or explicit
// Close. Windows can keep a timed-out notification in history afterwards.
func (n *Notification) OnClose(fn func(NotificationCloseReason)) (off func()) {
	return n.onClose.add(fn, false)
}

// OnFailed receives synchronous delivery errors and later native failures, on
// the UI thread. Show also returns its synchronous delivery error.
func (n *Notification) OnFailed(fn func(error)) (off func()) { return n.onFailed.add(fn, false) }

func (n *Notification) snapshot() (*platform.Notification, error) {
	out := &platform.Notification{ID: n.id, Title: n.opts.Title, Subtitle: n.opts.Subtitle, Body: n.opts.Body, Silent: n.opts.Silent, Group: n.opts.Group}
	if t := n.opts.Toast; t != nil {
		if strings.ContainsRune(t.AppID, 0) {
			return nil, errors.New("mygo: toast AppID contains NUL")
		}
		if t.Duration != "" && t.Duration != "short" && t.Duration != "long" {
			return nil, fmt.Errorf("mygo: invalid toast duration %q", t.Duration)
		}
		switch t.Scenario {
		case "", "default", "reminder", "alarm", "incomingCall":
		default:
			return nil, fmt.Errorf("mygo: invalid toast scenario %q", t.Scenario)
		}
		if len(t.Actions) > 5 {
			return nil, errors.New("mygo: Windows toasts allow at most five actions")
		}
		if t.XML != "" {
			if err := validateToastXML(t.XML); err != nil {
				return nil, err
			}
		}
		out.Toast = &platform.ToastOptions{AppID: t.AppID, XML: t.XML, Icon: t.Icon, Hero: t.Hero, Scenario: t.Scenario, Duration: t.Duration, SuppressPopup: t.SuppressPopup}
		for _, a := range t.Actions {
			if strings.TrimSpace(a.Label) == "" {
				return nil, errors.New("mygo: toast action label is empty")
			}
			out.Toast.Actions = append(out.Toast.Actions, platform.ToastAction{Label: a.Label, Arguments: a.Arguments})
		}
	}
	return out, nil
}

func validateToastXML(raw string) error {
	if len(raw) > 256<<10 {
		return errors.New("mygo: toast XML exceeds 256 KiB")
	}
	d := xml.NewDecoder(strings.NewReader(raw))
	depth, roots := 0, 0
	for {
		t, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("mygo: invalid toast XML: %w", err)
		}
		switch v := t.(type) {
		case xml.Directive:
			return errors.New("mygo: toast XML directives are not supported")
		case xml.StartElement:
			if depth == 0 {
				roots++
				if v.Name.Local != "toast" {
					return errors.New("mygo: toast XML root must be toast")
				}
			}
			depth++
		case xml.EndElement:
			depth--
		case xml.CharData:
			if depth == 0 && strings.TrimSpace(string(v)) != "" {
				return errors.New("mygo: text outside toast XML root")
			}
		}
	}
	if roots != 1 || depth != 0 {
		return errors.New("mygo: toast XML must have one root")
	}
	return nil
}

// notificationEvents is an optional platform observer. No backend invokes user
// code on a COM/OS worker: native events must first reach its UI event queue.
func (appHandler) NotificationActivated(id string, activation platform.NotificationActivation) {
	notifications.Lock()
	n := notifications.byID[id]
	notifications.Unlock()
	if n != nil {
		fire1(&n.onAction, NotificationActivation{activation.Arguments, activation.UserInput})
		fire(&n.onClick)
	}
	fire1(&App.onNotificationClick, id)
	// Listeners may have replaced the same ID while handling the click. Never
	// delete the replacement or attribute the old close event to the new one.
	if n != nil {
		notifications.Lock()
		if notifications.byID[id] == n {
			delete(notifications.byID, id)
		}
		notifications.Unlock()
		fire1(&n.onClose, NotificationActivated)
	}
}
func (appHandler) NotificationClosed(id, reason string) {
	notifications.Lock()
	n := notifications.byID[id]
	if reason == string(NotificationRemoved) || reason == string(NotificationActivated) || reason == string(NotificationDismissed) {
		delete(notifications.byID, id)
	}
	notifications.Unlock()
	if n != nil {
		fire1(&n.onClose, NotificationCloseReason(reason))
	}
}
func (appHandler) NotificationFailed(id string, err error) {
	notifications.Lock()
	n := notifications.byID[id]
	delete(notifications.byID, id)
	notifications.Unlock()
	if n != nil {
		fire1(&n.onFailed, err)
	}
}
