//go:build windows && (amd64 || arm64)

package windows

import (
	"errors"
	"log"
	"sync"
	"unicode/utf16"
	"unsafe"

	"github.com/egoist/mygo/internal/platform"
)

// ABI slots and IIDs are from Windows SDK windows.ui.notifications.h and
// windows.data.xml.dom.h. All WinRT objects are created/released on the UI STA.
// Delegates are agile: they copy callback arguments and enqueue Go values only;
// no user listener or backend mutable state is touched on a COM worker thread.
const wmAppToast = wmApp + 6

var (
	combase                       = systemDLL("combase.dll")
	procRoInitialize              = combase.NewProc("RoInitialize")
	procRoUninitialize            = combase.NewProc("RoUninitialize")
	procRoGetActivationFactory    = combase.NewProc("RoGetActivationFactory")
	procRoActivateInstance        = combase.NewProc("RoActivateInstance")
	procWindowsCreateString       = combase.NewProc("WindowsCreateString")
	procWindowsDeleteString       = combase.NewProc("WindowsDeleteString")
	procWindowsGetStringRawBuffer = combase.NewProc("WindowsGetStringRawBuffer")
	procGetProcessAppID           = shell32.NewProc("GetCurrentProcessExplicitAppUserModelID")
	toastManagerIID               = guid("50ac103f-d235-4598-bbef-98fe4d1a3ad4")
	toastManager2IID              = guid("7ab93c52-0e48-4750-ba9d-1a4113981847")
	toastFactoryIID               = guid("04124b20-82c6-4229-b109-fd9ed4662b53")
	toast2IID                     = guid("9dfb9fd1-143a-490e-90bf-b9fba7132de7")
	toastHistory2IID              = guid("3bc3d253-2f31-4092-9129-8ad5abf067da")
	toastActivatedArgsIID         = guid("e3bf92f3-c197-436f-8265-0625824f8dac")
	toastActivatedArgs2IID        = guid("ab7da512-cc61-568e-81be-304ac31038fa")
	xmlDocumentIID                = guid("f7f3a506-1e87-42d6-bcfb-b8c809fa5494")
	xmlDocumentIOIID              = guid("6cd0e74e-ee65-4489-9ebf-ca43e87ba637")
	xmlSerializerIID              = guid("5cc5b382-e6dd-4991-abef-06d8d2e7bd0c")
	iunknownToastIID              = guid("00000000-0000-0000-c000-000000000046")
)

type toastService struct {
	b                         *Backend
	initialized               bool
	manager, factory, history uintptr
	active                    map[string]*notification
	notifiers                 map[string]uintptr
	mu                        sync.Mutex
	queue                     []toastEvent
	stopped                   bool
	server                    *toastServer
	activated                 map[string]bool // deduplicates WinRT and COM callbacks of the same delivery
}

type notification struct {
	service               *toastService
	id, appID, tag, group string
	object                uintptr
	tokens                [3]int64 // dismissed, activated, failed
	registered            int
}

type toastEvent struct {
	n            *notification
	kind, reason string
	activation   platform.NotificationActivation
	message      platform.NotificationActivationMessage
	err          error
}

func hstring(text string) (uintptr, error) {
	units := utf16.Encode([]rune(text))
	var data *uint16
	if len(units) > 0 {
		data = &units[0]
	}
	var out uintptr
	hr, _, _ := procWindowsCreateString.Call(uintptr(unsafe.Pointer(data)), uintptr(len(units)), uintptr(unsafe.Pointer(&out)))
	if failed(hr) {
		return 0, hresultError("creating WinRT string", hr)
	}
	return out, nil
}
func deleteHstring(s uintptr) {
	if s != 0 {
		procWindowsDeleteString.Call(s)
	}
}
func readHstring(s uintptr) string {
	if s == 0 {
		return ""
	}
	var n uint32
	p, _, _ := procWindowsGetStringRawBuffer.Call(s, uintptr(unsafe.Pointer(&n)))
	if p == 0 || n == 0 {
		return ""
	}
	return string(utf16.Decode(unsafe.Slice((*uint16)(native(p)), n)))
}
func winrtFactory(name string, iid *GUID) (uintptr, error) {
	s, err := hstring(name)
	if err != nil {
		return 0, err
	}
	defer deleteHstring(s)
	var out uintptr
	hr, _, _ := procRoGetActivationFactory.Call(s, uintptr(unsafe.Pointer(iid)), uintptr(unsafe.Pointer(&out)))
	if failed(hr) {
		return 0, hresultError("activating "+name, hr)
	}
	return out, nil
}
func (b *Backend) toastService() (*toastService, error) {
	if b.toast != nil {
		return b.toast, nil
	}
	if !has(procRoInitialize) {
		return nil, platform.ErrUnsupported
	}
	hr, _, _ := procRoInitialize.Call(0) // RO_INIT_SINGLETHREADED, balanced at shutdown
	if failed(hr) {
		return nil, hresultError("initializing WinRT STA", hr)
	}
	s := &toastService{b: b, initialized: true, active: map[string]*notification{}, notifiers: map[string]uintptr{}}
	var err error
	s.manager, err = winrtFactory("Windows.UI.Notifications.ToastNotificationManager", &toastManagerIID)
	if err == nil {
		s.factory, err = winrtFactory("Windows.UI.Notifications.ToastNotification", &toastFactoryIID)
	}
	if err == nil {
		m2 := queryInterface(s.manager, &toastManager2IID)
		if m2 == 0 {
			err = platform.ErrUnsupported
		} else {
			hr = comCall(m2, 6, uintptr(unsafe.Pointer(&s.history)))
			release(m2)
			if failed(hr) {
				err = hresultError("getting toast history", hr)
			}
		}
	}
	if err != nil {
		s.close()
		return nil, err
	}
	b.toast = s
	return s, nil
}
func processAppID() (string, error) {
	if !has(procGetProcessAppID) {
		return "", platform.ErrUnsupported
	}
	var p uintptr
	hr, _, _ := procGetProcessAppID.Call(uintptr(unsafe.Pointer(&p)))
	if failed(hr) {
		return "", errors.New("mygo: WinRT notifications require a registered process AppUserModelID or Toast.AppID")
	}
	defer procCoTaskMemFree.Call(p)
	id := wstr(p)
	if id == "" {
		return "", errors.New("mygo: empty AppUserModelID")
	}
	return id, nil
}
func (s *toastService) notifier(appID string) (uintptr, error) {
	if n := s.notifiers[appID]; n != 0 {
		return n, nil
	}
	a, err := hstring(appID)
	if err != nil {
		return 0, err
	}
	defer deleteHstring(a)
	var n uintptr
	if hr := comCall(s.manager, 7, a, uintptr(unsafe.Pointer(&n))); failed(hr) {
		return 0, hresultError("creating toast notifier", hr)
	}
	s.notifiers[appID] = n
	return n, nil
}
func (b *Backend) NotificationsSupported() bool { _, err := b.toastService(); return err == nil }
func (b *Backend) ShowNotification(n *platform.Notification, done func(error)) {
	done(b.showNotification(n))
}
func (b *Backend) showNotification(n *platform.Notification) error {
	s, err := b.toastService()
	if err != nil {
		return err
	}
	appID := ""
	if n.Toast != nil {
		appID = n.Toast.AppID
	}
	if appID == "" {
		appID, err = processAppID()
		if err != nil {
			return err
		}
	}
	notifier, err := s.notifier(appID)
	if err != nil {
		return err
	}
	var setting int32
	if hr := comCall(notifier, 8, uintptr(unsafe.Pointer(&setting))); failed(hr) && uint32(hr) != 0x80070490 {
		return hresultError("reading notification setting", hr)
	}
	// A freshly registered unpackaged AUMID has no per-app setting until its
	// first delivery (ERROR_NOT_FOUND). Show is authoritative in that state;
	// it is still called once and its HRESULT/native Failed event is returned.
	if setting != 0 {
		return platform.ErrNotificationsDenied
	}
	doc, err := newToastDocument(toastXML(n))
	if err != nil {
		return err
	}
	defer release(doc)
	record := &notification{service: s, id: n.ID, appID: appID, tag: toastKey(n.ID)}
	if n.Group != "" {
		record.group = toastKey(n.Group)
	}
	if hr := comCall(s.factory, 6, doc, uintptr(unsafe.Pointer(&record.object))); failed(hr) {
		return hresultError("creating toast", hr)
	}
	ok := false
	defer func() {
		if !ok {
			record.detach()
		}
	}()
	t2 := queryInterface(record.object, &toast2IID)
	if t2 == 0 {
		return platform.ErrUnsupported
	}
	defer release(t2)
	for _, p := range []struct {
		slot  int
		value string
	}{{6, record.tag}, {8, record.group}} {
		v, e := hstring(p.value)
		if e != nil {
			return e
		}
		hr := comCall(t2, p.slot, v)
		deleteHstring(v)
		if failed(hr) {
			return hresultError("setting toast tag/group", hr)
		}
	}
	if n.Toast != nil && n.Toast.SuppressPopup {
		if hr := comCall(t2, 10, 1); failed(hr) {
			return hresultError("suppressing toast banner", hr)
		}
	}
	for i, kind := range []string{"dismissed", "activated", "failed"} {
		i, kind := i, kind
		h := newToastDelegate(toastDelegateIIDs[i], func(_, args uintptr) {
			e := toastEvent{n: record, kind: kind}
			switch kind {
			case "activated":
				e.activation = readToastActivation(args, n)
			case "dismissed":
				var reason int32
				if hr := comCall(args, 6, uintptr(unsafe.Pointer(&reason))); failed(hr) {
					e.kind = "failed"
					e.err = hresultError("reading toast dismissal", hr)
				} else {
					e.reason = map[int32]string{0: "dismissed", 1: "removed", 2: "timed-out"}[reason]
				}
			case "failed":
				var code int32
				hr := comCall(args, 6, uintptr(unsafe.Pointer(&code)))
				if failed(hr) {
					e.err = hresultError("reading toast failure", hr)
				} else {
					e.err = hresultError("displaying toast", uintptr(uint32(code)))
				}
			}
			s.enqueue(e)
		})
		hr := comCall(record.object, 9+i*2, h, uintptr(unsafe.Pointer(&record.tokens[i])))
		release(h)
		if failed(hr) {
			return hresultError("registering toast "+kind, hr)
		}
		record.registered++
	}
	// Show must succeed before replacing our live record. On a failure the old
	// notification and its listeners continue to represent the delivered toast.
	if hr := comCall(notifier, 6, record.object); failed(hr) {
		return hresultError("showing toast", hr)
	}
	if old := s.active[n.ID]; old != nil {
		old.detach()
	}
	delete(s.activated, n.ID)
	s.active[n.ID] = record
	ok = true
	return nil
}
func newToastDocument(raw string) (uintptr, error) {
	s, err := hstring("Windows.Data.Xml.Dom.XmlDocument")
	if err != nil {
		return 0, err
	}
	defer deleteHstring(s)
	var instance uintptr
	hr, _, _ := procRoActivateInstance.Call(s, uintptr(unsafe.Pointer(&instance)))
	if failed(hr) {
		return 0, hresultError("creating toast XML document", hr)
	}
	defer release(instance)
	doc := queryInterface(instance, &xmlDocumentIID)
	if doc == 0 {
		return 0, platform.ErrUnsupported
	}
	io := queryInterface(instance, &xmlDocumentIOIID)
	if io == 0 {
		release(doc)
		return 0, platform.ErrUnsupported
	}
	defer release(io)
	x, err := hstring(raw)
	if err != nil {
		release(doc)
		return 0, err
	}
	defer deleteHstring(x)
	if hr := comCall(io, 6, x); failed(hr) {
		release(doc)
		return 0, hresultError("parsing toast XML", hr)
	}
	return doc, nil
}
func (n *notification) detach() {
	if n.object == 0 {
		return
	}
	for i := 0; i < n.registered; i++ {
		comCall(n.object, 10+i*2, uintptr(n.tokens[i]))
	}
	release(n.object)
	n.object = 0
	n.registered = 0
}
func (s *toastService) enqueue(e toastEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return
	}
	s.queue = append(s.queue, e)
	postMessage(s.b.appHwnd, wmAppToast, 0, 0)
}
func (s *toastService) dispatch() {
	s.mu.Lock()
	q := s.queue
	s.queue = nil
	s.mu.Unlock()
	for _, e := range q {
		if e.kind == "com-activated" {
			s.dispatchActivation(e.message)
			continue
		}
		n := e.n
		if s.active[n.id] != n {
			continue
		}
		observer, _ := s.b.h.(platform.NotificationObserver)
		switch e.kind {
		case "activated":
			if s.activationSeen(n.id) {
				continue
			}
			delete(s.active, n.id)
			n.detach()
			if observer != nil {
				observer.NotificationActivated(n.id, e.activation)
			} else {
				s.b.h.NotificationClicked(n.id)
			}
		case "dismissed":
			// A timeout only hides the banner: keep activation registered for
			// notification-center clicks until explicit removal or shutdown.
			if e.reason != "timed-out" {
				delete(s.active, n.id)
				n.detach()
			}
			if observer != nil {
				observer.NotificationClosed(n.id, e.reason)
			}
		case "failed":
			delete(s.active, n.id)
			n.detach()
			if observer != nil {
				observer.NotificationFailed(n.id, e.err)
			}
		}
	}
}
func (b *Backend) RemoveNotification(id string) {
	if b.toast == nil {
		return
	}
	s := b.toast
	n := s.active[id]
	if n == nil {
		return
	}
	delete(s.active, id)
	if notifier := s.notifiers[n.appID]; notifier != 0 {
		comCall(notifier, 7, n.object)
	}
	if err := s.removeHistory(n.tag, n.group, n.appID); err != nil {
		log.Print(err)
	}
	n.detach()
}
func (s *toastService) removeHistory(tag, group, appID string) error {
	values := make([]uintptr, 3)
	for i, v := range []string{tag, group, appID} {
		h, e := hstring(v)
		if e != nil {
			for _, x := range values {
				deleteHstring(x)
			}
			return e
		}
		values[i] = h
	}
	defer func() {
		for _, h := range values {
			deleteHstring(h)
		}
	}()
	if hr := comCall(s.history, 8, values...); failed(hr) {
		return hresultError("removing toast from history", hr)
	}
	return nil
}
func (b *Backend) RemoveAllNotifications() {
	if b.toast == nil {
		return
	}
	s := b.toast
	for id := range s.active {
		b.RemoveNotification(id)
	}
	// Explicit ClearNotifications is the only operation that clears earlier
	// runs; ordinary startup and shutdown never alter notification history.
	if appID, err := processAppID(); err == nil {
		if _, err = s.notifier(appID); err != nil {
			log.Print(err)
		}
	}
	for appID := range s.notifiers {
		h, err := hstring(appID)
		if err != nil {
			continue
		}
		hr := comCall(s.history, 12, h)
		deleteHstring(h)
		if failed(hr) {
			log.Print(hresultError("clearing toast history", hr))
		}
	}
}
func toastDocumentXML(doc uintptr) (string, error) {
	serializer := queryInterface(doc, &xmlSerializerIID)
	if serializer == 0 {
		return "", platform.ErrUnsupported
	}
	defer release(serializer)
	var out uintptr
	if hr := comCall(serializer, 6, uintptr(unsafe.Pointer(&out))); failed(hr) {
		return "", hresultError("serializing toast XML", hr)
	}
	defer deleteHstring(out)
	return readHstring(out), nil
}
func (b *Backend) NotificationHistory(appID string) ([]platform.NotificationRecord, error) {
	s, err := b.toastService()
	if err != nil {
		return nil, err
	}
	if appID == "" {
		appID, err = processAppID()
		if err != nil {
			return nil, err
		}
	}
	h2 := queryInterface(s.history, &toastHistory2IID)
	if h2 == 0 {
		return nil, platform.ErrUnsupported
	}
	defer release(h2)
	a, err := hstring(appID)
	if err != nil {
		return nil, err
	}
	defer deleteHstring(a)
	var view uintptr
	if hr := comCall(h2, 7, a, uintptr(unsafe.Pointer(&view))); failed(hr) {
		return nil, hresultError("reading toast history", hr)
	}
	defer release(view)
	var size uint32
	if hr := comCall(view, 7, uintptr(unsafe.Pointer(&size))); failed(hr) {
		return nil, hresultError("reading toast history size", hr)
	}
	rows := make([]platform.NotificationRecord, 0, size)
	for i := uint32(0); i < size; i++ {
		var toast, doc uintptr
		if hr := comCall(view, 6, uintptr(i), uintptr(unsafe.Pointer(&toast))); failed(hr) {
			return nil, hresultError("reading history toast", hr)
		}
		row := platform.NotificationRecord{}
		hr := comCall(toast, 6, uintptr(unsafe.Pointer(&doc)))
		if !failed(hr) {
			row.XML, err = toastDocumentXML(doc)
			release(doc)
		} else {
			err = hresultError("reading history XML", hr)
		}
		if err == nil {
			t2 := queryInterface(toast, &toast2IID)
			if t2 != 0 {
				for _, p := range []struct {
					slot int
					out  *string
				}{{7, &row.Tag}, {9, &row.Group}} {
					var h uintptr
					if hr := comCall(t2, p.slot, uintptr(unsafe.Pointer(&h))); !failed(hr) {
						*p.out = readHstring(h)
						deleteHstring(h)
					}
				}
				release(t2)
			}
			row.ID = toastIDFromXML(row.XML)
		}
		release(toast)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	return rows, nil
}
func (s *toastService) close() {
	s.mu.Lock()
	s.stopped = true
	s.queue = nil
	s.mu.Unlock()
	if s.server != nil {
		s.server.close()
		s.server = nil
	}
	for _, n := range s.active {
		n.detach()
	}
	clear(s.active)
	for _, n := range s.notifiers {
		release(n)
	}
	clear(s.notifiers)
	release(s.history)
	release(s.factory)
	release(s.manager)
	s.history, s.factory, s.manager = 0, 0, 0
	if s.initialized {
		procRoUninitialize.Call()
		s.initialized = false
	}
}
