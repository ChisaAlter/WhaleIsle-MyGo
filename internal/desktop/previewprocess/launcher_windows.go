//go:build windows && (amd64 || arm64)

package previewprocess

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

const (
	attributeSecurityCapabilities = 0x00020009
	attributeJobList              = 0x0002000d
)

var (
	userenv           = windows.NewLazySystemDLL("userenv.dll")
	createProfileProc = userenv.NewProc("CreateAppContainerProfile")
	deleteProfileProc = userenv.NewProc("DeleteAppContainerProfile")
	profileFolderProc = userenv.NewProc("GetAppContainerFolderPath")
	coTaskMemFreeProc = windows.NewLazySystemDLL("ole32.dll").NewProc("CoTaskMemFree")
)

type securityCapabilities struct {
	SID          *windows.SID
	Capabilities *windows.SIDAndAttributes
	Count        uint32
	Reserved     uint32
}

type profile struct {
	name   string
	sid    *windows.SID
	folder string
}

func hresult(op string, hr uintptr) error {
	if int32(uint32(hr)) < 0 {
		return fmt.Errorf("%s failed (HRESULT 0x%08x)", op, uint32(hr))
	}
	return nil
}

func newProfile(nonce string) (_ *profile, err error) {
	p := &profile{name: "WhaleIsle.Preview." + nonce[:32]}
	name, err := windows.UTF16PtrFromString(p.name)
	if err != nil {
		return nil, err
	}
	// Every session creates a NEW random profile. ERROR_ALREADY_EXISTS is not
	// reused, because an old profile could carry loopback/network exemptions.
	hr, _, _ := createProfileProc.Call(uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(name)), 0, 0, uintptr(unsafe.Pointer(&p.sid)))
	if err := hresult("CreateAppContainerProfile", hr); err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = p.close()
		}
	}()
	if p.sid == nil {
		return nil, errors.New("CreateAppContainerProfile returned no SID")
	}
	sid, err := windows.UTF16PtrFromString(p.sid.String())
	if err != nil {
		return nil, err
	}
	var folder *uint16
	hr, _, _ = profileFolderProc.Call(uintptr(unsafe.Pointer(sid)), uintptr(unsafe.Pointer(&folder)))
	if err = hresult("GetAppContainerFolderPath", hr); err != nil {
		return nil, err
	}
	if folder == nil {
		return nil, errors.New("missing AppContainer profile directory")
	}
	p.folder = windows.UTF16PtrToString(folder)
	coTaskMemFreeProc.Call(uintptr(unsafe.Pointer(folder)))
	if !filepath.IsAbs(p.folder) {
		return nil, errors.New("invalid AppContainer profile directory")
	}
	return p, nil
}

func (p *profile) close() error {
	name, err := windows.UTF16PtrFromString(p.name)
	if err == nil {
		hr, _, _ := deleteProfileProc.Call(uintptr(unsafe.Pointer(name)))
		err = hresult("DeleteAppContainerProfile", hr)
	}
	if p.sid != nil {
		_ = windows.FreeSid(p.sid)
		p.sid = nil
	}
	return err
}

// The Windows network-isolation implementation depends on these services.
// Query only: this code never starts services, edits firewall rules, grants
// network capabilities, installs loopback exemptions or changes global ACLs.
func requireFilteringServices() error {
	manager, err := windows.OpenSCManager(nil, nil, windows.SC_MANAGER_CONNECT)
	if err != nil {
		return err
	}
	defer windows.CloseServiceHandle(manager)
	for _, name := range []string{"BFE", "MpsSvc"} {
		namePtr, err := windows.UTF16PtrFromString(name)
		if err != nil {
			return err
		}
		handle, err := windows.OpenService(manager, namePtr, windows.SERVICE_QUERY_STATUS)
		if err != nil {
			return err
		}
		s := &mgr.Service{Name: name, Handle: handle}
		status, queryErr := s.Query()
		s.Close()
		if queryErr != nil {
			return queryErr
		}
		if status.State != svc.Running {
			return fmt.Errorf("Windows filtering service %s is not running", name)
		}
	}
	return nil
}

func copyTrustedFile(source, destination string, limit int64) (err error) {
	src, err := os.Open(source)
	if err != nil {
		return err
	}
	defer src.Close()
	info, err := src.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() < 1 || info.Size() > limit {
		return errors.New("invalid trusted helper binary")
	}
	dst, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := dst.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			_ = os.Remove(destination)
		}
	}()
	n, err := io.Copy(dst, io.LimitReader(src, limit+1))
	if err != nil {
		return err
	}
	if n != info.Size() || n > limit {
		return errors.New("trusted helper changed while staging")
	}
	return dst.Sync()
}

func helperEnvironment(folder string) ([]uint16, error) {
	root := os.Getenv("SystemRoot")
	if !filepath.IsAbs(root) || strings.ContainsRune(root, 0) {
		return nil, errors.New("invalid SystemRoot")
	}
	// Never propagate credentials, proxies, WEBVIEW2_* overrides, debug pipes,
	// MYGO_GENERATE, Go runtime injection knobs, or the parent's general PATH.
	entries := []string{
		"LOCALAPPDATA=" + folder,
		"PATH=" + filepath.Join(root, "System32") + ";" + root,
		"SystemRoot=" + root,
		"TEMP=" + filepath.Join(folder, "tmp"),
		"TMP=" + filepath.Join(folder, "tmp"),
		"USERPROFILE=" + folder,
		"WINDIR=" + root,
	}
	return utf16.Encode([]rune(strings.Join(entries, "\x00") + "\x00\x00")), nil
}

func openPlatform(ctx context.Context, opt Options, doc Document) (_ *Session, err error) {
	for _, proc := range []*windows.LazyProc{createProfileProc, deleteProfileProc, profileFolderProc, coTaskMemFreeProc, isProcessInJobProc} {
		if err = proc.Find(); err != nil {
			return nil, err
		}
	}
	if err = requireFilteringServices(); err != nil {
		return nil, err
	}
	helper, err := filepath.Abs(opt.HelperPath)
	if err != nil {
		return nil, err
	}
	var random [32]byte
	if _, err = rand.Read(random[:]); err != nil {
		return nil, err
	}
	nonce := hex.EncodeToString(random[:])
	p, err := newProfile(nonce)
	if err != nil {
		return nil, err
	}
	ownedProfile := true
	defer func() {
		if ownedProfile {
			_ = p.close()
		}
	}()
	sid := p.sid.String()
	folder := filepath.Join(p.folder, "WhaleIslePreview")
	if err = os.Mkdir(folder, 0700); err != nil {
		return nil, err
	}
	if err = os.Mkdir(filepath.Join(folder, "tmp"), 0700); err != nil {
		return nil, err
	}
	staged := filepath.Join(folder, "whaleisle-preview.exe")
	if err = copyTrustedFile(helper, staged, 256*1024*1024); err != nil {
		return nil, err
	}
	// The normal MyGo loader prefers the SDK's sibling loader, if shipped.
	loader := filepath.Join(filepath.Dir(helper), "WebView2Loader.dll")
	if _, statErr := os.Stat(loader); statErr == nil {
		if err = copyTrustedFile(loader, filepath.Join(folder, "WebView2Loader.dll"), 8*1024*1024); err != nil {
			return nil, err
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return nil, statErr
	}
	env, err := helperEnvironment(folder)
	if err != nil {
		return nil, err
	}
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return nil, err
	}
	ownedJob := true
	defer func() {
		if ownedJob {
			windows.CloseHandle(job)
		}
	}()
	var limits windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
	limits.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE | windows.JOB_OBJECT_LIMIT_ACTIVE_PROCESS | windows.JOB_OBJECT_LIMIT_JOB_MEMORY
	limits.BasicLimitInformation.ActiveProcessLimit = 32
	limits.JobMemoryLimit = 1024 * 1024 * 1024
	if _, err = windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits))); err != nil {
		return nil, err
	}
	// No BREAKAWAY_OK, SILENT_BREAKAWAY_OK, UI restriction or shared job handle.
	// JOB_LIST assigns this job atomically at creation, before any helper code.
	inputRead, inputWrite, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	defer inputRead.Close()
	outputRead, outputWrite, err := os.Pipe()
	if err != nil {
		inputWrite.Close()
		return nil, err
	}
	defer outputWrite.Close()
	errorRead, errorWrite, err := os.Pipe()
	if err != nil {
		inputWrite.Close()
		outputRead.Close()
		return nil, err
	}
	defer errorWrite.Close()
	ownedPipes := true
	defer func() {
		if ownedPipes {
			inputWrite.Close()
			outputRead.Close()
			errorRead.Close()
		}
	}()
	handles := []windows.Handle{windows.Handle(inputRead.Fd()), windows.Handle(outputWrite.Fd()), windows.Handle(errorWrite.Fd())}
	for _, h := range handles {
		if err = windows.SetHandleInformation(h, windows.HANDLE_FLAG_INHERIT, windows.HANDLE_FLAG_INHERIT); err != nil {
			return nil, err
		}
	}
	attrs, err := windows.NewProcThreadAttributeList(3)
	if err != nil {
		return nil, err
	}
	defer attrs.Delete()
	caps := securityCapabilities{SID: p.sid} // pCapabilities=nil, CapabilityCount=0.
	if err = attrs.Update(attributeSecurityCapabilities, unsafe.Pointer(&caps), unsafe.Sizeof(caps)); err != nil {
		return nil, err
	}
	if err = attrs.Update(attributeJobList, unsafe.Pointer(&job), unsafe.Sizeof(job)); err != nil {
		return nil, err
	}
	if err = attrs.Update(windows.PROC_THREAD_ATTRIBUTE_HANDLE_LIST, unsafe.Pointer(&handles[0]), uintptr(len(handles))*unsafe.Sizeof(handles[0])); err != nil {
		return nil, err
	}
	var startup windows.StartupInfoEx
	startup.Cb = uint32(unsafe.Sizeof(startup))
	startup.Flags = windows.STARTF_USESTDHANDLES
	startup.StdInput, startup.StdOutput, startup.StdErr = handles[0], handles[1], handles[2]
	startup.ProcThreadAttributeList = attrs.List()
	app, err := windows.UTF16PtrFromString(staged)
	if err != nil {
		return nil, err
	}
	command, err := windows.UTF16PtrFromString(windows.ComposeCommandLine([]string{staged, "--appcontainer-helper-v1", nonce, sid}))
	if err != nil {
		return nil, err
	}
	cwd, err := windows.UTF16PtrFromString(folder)
	if err != nil {
		return nil, err
	}
	var process windows.ProcessInformation
	flags := uint32(windows.CREATE_SUSPENDED | windows.CREATE_UNICODE_ENVIRONMENT | windows.EXTENDED_STARTUPINFO_PRESENT | windows.CREATE_NO_WINDOW)
	err = windows.CreateProcess(app, command, nil, nil, true, flags, &env[0], cwd, &startup.StartupInfo, &process)
	runtime.KeepAlive(caps)
	runtime.KeepAlive(handles)
	runtime.KeepAlive(env)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(process.Thread)
	// Parent copies of the child's pipe ends must close before waiting for EOF.
	inputRead.Close()
	outputWrite.Close()
	errorWrite.Close()
	ownedProfile, ownedJob, ownedPipes = false, false, false
	lifetime, cancelLifetime := context.WithTimeout(ctx, 30*time.Minute)
	s := &Session{done: make(chan struct{})}
	s.initOpenUIEvents(doc)
	s.closeFn = func() error {
		cancelLifetime()
		inputWrite.Close()
		outputRead.Close()
		errorRead.Close()
		terminateErr := windows.TerminateJobObject(job, 1)
		// Keep the job handle alive until every member has exited. If cleanup
		// cannot be observed, retain profile data instead of deleting live UDFs.
		deadline := time.Now().Add(5 * time.Second)
		empty := false
		for time.Now().Before(deadline) {
			pids, queryErr := jobPIDs(job)
			if queryErr == nil && len(pids) == 0 {
				empty = true
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
		windows.CloseHandle(process.Process)
		windows.CloseHandle(job)
		if !empty {
			_ = windows.FreeSid(p.sid)
			p.sid = nil
			return errors.New("preview job exit was not observed; profile retained")
		}
		return errors.Join(terminateErr, p.close())
	}
	defer func() {
		if err != nil {
			_ = s.Close()
		}
	}()
	go func() { _, _ = io.Copy(io.Discard, errorRead) }()
	stopCancellation := context.AfterFunc(lifetime, func() { _ = s.Close() })
	go func() { <-s.Done(); stopCancellation() }()
	if err = verifyInJob(process.Process, job); err != nil {
		return nil, err
	}
	if err = verifyProcessToken(process.Process, sid); err != nil {
		return nil, err
	}
	if _, err = windows.ResumeThread(process.Thread); err != nil {
		return nil, err
	}
	// Bound startup and all IPC. Timeout closes the job and pipe, unblocking
	// both readers and writers without exposing the untrusted snapshot.
	startupTimer := time.AfterFunc(30*time.Second, func() { _ = s.Close() })
	defer startupTimer.Stop()
	ready, err := readPacket(outputRead)
	if err != nil {
		return nil, fmt.Errorf("preview helper handshake: %w", err)
	}
	if err = expect(ready, "ready", nonce); err != nil {
		return nil, err
	}
	if err = requireOpenUIBridge(doc, ready); err != nil {
		return nil, err
	}
	if ready.SID != sid || ready.HelperPID != process.ProcessId {
		return nil, errors.New("unexpected helper identity")
	}
	if err = verifyTree(job, sid, process.ProcessId, ready.BrowserPID); err != nil {
		return nil, err
	}
	// This is the first point at which any document bytes enter the helper.
	if err = sendDocument(inputWrite, nonce, doc); err != nil {
		return nil, err
	}
	loaded, err := readPacket(outputRead)
	if err != nil {
		return nil, err
	}
	if err = expect(loaded, "loaded", nonce); err != nil {
		return nil, err
	}
	if loaded.BrowserPID != ready.BrowserPID {
		return nil, errors.New("preview browser identity changed")
	}
	if err = verifyTree(job, sid, process.ProcessId, ready.BrowserPID); err != nil {
		return nil, err
	}
	var commandMu sync.Mutex
	s.enableOpenUIUpdates(ready.OpenUIUpdate)
	if !doc.capture {
		s.surfaceHandle, s.surfacePID = uintptr(loaded.WindowHandle), process.ProcessId
		if !validSurfaceWindow(s.surfaceHandle, s.surfacePID) {
			return nil, errors.New("invalid preview surface owner")
		}
		s.theme = ""
		s.sendTheme = func(appearance string, variables map[string]string) error {
			commandMu.Lock()
			defer commandMu.Unlock()
			return writePacket(inputWrite, packet{Version: protocolVersion, Kind: "theme", Nonce: nonce, Appearance: appearance, Variables: variables})
		}
	}
	if doc.capture {
		result, captureErr := receiveCapture(outputRead, nonce, doc.Width)
		if captureErr != nil {
			return nil, captureErr
		}
		if err = verifyTree(job, sid, process.ProcessId, ready.BrowserPID); err != nil {
			return nil, err
		}
		s.capture = &result
	}
	if !startupTimer.Stop() {
		return nil, errors.New("preview startup deadline expired")
	}
	if err = lifetime.Err(); err != nil {
		return nil, err
	}
	startOpenUIUpdatePump(lifetime, s, func(raw []byte) error {
		commandMu.Lock()
		defer commandMu.Unlock()
		return sendOpenUIUpdate(inputWrite, nonce, raw)
	})
	// A heartbeat is a drift detector, not the network security boundary.
	// AppContainer token inheritance and the non-breakaway job are the boundary.
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-s.Done():
				return
			case <-ticker.C:
				if verifyTree(job, sid, process.ProcessId, ready.BrowserPID) != nil {
					_ = s.Close()
					return
				}
			}
		}
	}()
	go func() {
		defer s.Close()
		if doc.capture {
			_, _ = io.Copy(io.Discard, outputRead)
			return
		}
		for {
			p, err := readPacket(outputRead)
			if err != nil {
				return
			}
			if p.Kind == "openui-event" {
				if s.receiveOpenUIEvent(outputRead, p, nonce) != nil {
					return
				}
				continue
			}
			if expect(p, "size", nonce) != nil || p.Height < 80 || p.Height > 2000 {
				return
			}
			s.height.Store(int64(p.Height))
		}
	}()
	return s, nil
}
