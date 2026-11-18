//go:build windows

// Package installhealth checks the promoted desktop before NSIS commits its
// registration or retires the previous payload. It uses a fresh synthetic
// profile and the candidate's own runtime, never a user's account or workspace.
package installhealth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"whaleisle/internal/desktop/terminalruntime"
	"whaleisle/internal/startuptrace"
)

type result struct {
	Schema          int                  `json:"schema"`
	Installation    string               `json:"installation"`
	Passed          bool                 `json:"passed"`
	Error           string               `json:"error,omitempty"`
	ElapsedMS       int64                `json:"elapsedMs"`
	Profile         string               `json:"profile,omitempty"`
	ProfileRemoved  bool                 `json:"profileRemoved"`
	ProcessesExited bool                 `json:"processesExited"`
	Startup         *startuptrace.Report `json:"startup,omitempty"`
}

// Check returns success only after the actual candidate has created its native
// window, connected to Host, opened its workspace and loaded Host inventory.
// All probe processes must exit before NSIS may roll directories back or commit.
func Check(installation, reportPath string) (failure error) {
	started := time.Now()
	r := result{Schema: 1, Installation: installation}
	defer func() {
		r.ElapsedMS = time.Since(started).Milliseconds()
		if failure != nil {
			r.Error = failure.Error()
		}
		r.Passed = failure == nil
		if !filepath.IsAbs(reportPath) {
			failure = errors.Join(failure, errors.New("health report must be absolute"))
			return
		}
		raw, err := json.MarshalIndent(r, "", "  ")
		if err == nil {
			err = os.MkdirAll(filepath.Dir(reportPath), 0700)
		}
		if err == nil {
			err = os.WriteFile(reportPath, append(raw, '\n'), 0600)
		}
		failure = errors.Join(failure, err)
	}()
	if !filepath.IsAbs(installation) {
		return errors.New("installation must be absolute")
	}
	installation = filepath.Clean(installation)
	resolved, err := filepath.EvalSymlinks(installation)
	if err != nil || !strings.EqualFold(resolved, installation) {
		return fmt.Errorf("installation must be an unlinked directory: %s", installation)
	}
	for _, name := range []string{"whaleisle.exe", "whaleisle-preview.exe", "runtime/node/node.exe"} {
		info, err := os.Lstat(filepath.Join(installation, filepath.FromSlash(name)))
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || info.Size() == 0 {
			return fmt.Errorf("invalid installed component: %s", name)
		}
	}
	spec, err := terminalruntime.Specification("windows-" + runtime.GOARCH)
	if err != nil {
		return err
	}
	if _, err = terminalruntime.Resolve(spec, filepath.Join(installation, spec.Name), nil); err != nil {
		return err
	}
	profile, err := os.MkdirTemp("", "whaleisle-install-health-")
	if err != nil {
		return err
	}
	r.Profile = profile
	// No process has started yet; once it has, cleanup is conditional on the
	// owned job's exit, not merely the desktop parent's exit.
	safeToRemove := true
	defer func() {
		if !safeToRemove {
			return
		}
		if err := removeOwnedProfile(profile); err != nil {
			failure = errors.Join(failure, err)
		} else {
			r.ProfileRemoved = true
		}
	}()
	workspace := filepath.Join(profile, "workspace")
	if err = os.Mkdir(workspace, 0700); err != nil {
		return err
	}
	if err = os.Mkdir(filepath.Join(profile, "tmp"), 0700); err != nil {
		return err
	}
	config, _ := json.Marshal(map[string]any{"autoDownloadUpdates": false, "autoStartDesktop": true, "remoteEnabled": false,
		"harnessAutoRestart": false, "closeToTray": false, "workspace": workspace})
	if err = os.WriteFile(filepath.Join(profile, "config.json"), config, 0600); err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(profile, "welcome-shown"), nil, 0600); err != nil {
		return err
	}
	trace := filepath.Join(profile, "startup.json")
	job, process, err := startCandidate(installation, profile, trace)
	if err != nil {
		return err
	}
	safeToRemove = false
	defer windows.CloseHandle(process)
	defer windows.CloseHandle(job)
	defer func() {
		exitErr := stopCandidate(job, process)
		r.ProcessesExited = exitErr == nil
		safeToRemove = exitErr == nil
		failure = errors.Join(failure, exitErr)
	}()
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		status, err := windows.WaitForSingleObject(process, 0)
		if err != nil {
			return err
		}
		if status == windows.WAIT_OBJECT_0 {
			return errors.New("candidate exited before installation health was confirmed")
		}
		if raw, err := os.ReadFile(trace); err == nil {
			var report startuptrace.Report
			if json.Unmarshal(raw, &report) == nil {
				r.Startup = &report
				if ready(report) {
					return nil
				}
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return errors.New("candidate did not confirm native window, Host, workspace and plugin inventory within 90 seconds")
}

func ready(r startuptrace.Report) bool {
	stages := map[string]bool{}
	for _, e := range r.Events {
		if e.Kind == "mark" {
			stages[e.Stage] = true
		}
	}
	return stages["window.created"] && stages["host.ready"] && stages["workspace.ready"] && stages["plugins.inventory_ready"]
}

func environment(installation, profile, trace string) []uint16 {
	values := map[string]string{}
	for _, entry := range os.Environ() {
		name, value, ok := strings.Cut(entry, "=")
		upper := strings.ToUpper(name)
		if !ok || name == "" || strings.HasPrefix(upper, "WHALEISLE_") || strings.HasPrefix(upper, "DSH_") || strings.HasPrefix(upper, "MYGO_") || strings.HasPrefix(upper, "NODE_") || upper == "ELECTRON_RUN_AS_NODE" {
			continue
		}
		values[upper] = value
	}
	for name, value := range map[string]string{
		"WHALEISLE_USER_DATA": profile, "WHALEISLE_ISOLATED": "1", "WHALEISLE_ISOLATED_VISIBLE": "0",
		"WHALEISLE_DISABLE_AUTO_UPDATE": "1", "WHALEISLE_STARTUP_TRACE": trace,
		"WHALEISLE_VENDOR": filepath.Join(installation, "runtime", "vendor"), "WHALEISLE_NODE_DIR": filepath.Join(installation, "runtime", "node"),
		"LOCALAPPDATA": profile, "DSH_HOME": filepath.Join(profile, "dsh-home"), "TEMP": filepath.Join(profile, "tmp"), "TMP": filepath.Join(profile, "tmp"),
	} {
		values[name] = value
	}
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	var block []uint16
	for _, name := range names {
		s, _ := windows.UTF16FromString(name + "=" + values[name])
		block = append(block, s...)
	}
	return append(block, 0)
}

func startCandidate(installation, profile, trace string) (job, process windows.Handle, failure error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return 0, 0, err
	}
	defer func() {
		if failure != nil {
			windows.CloseHandle(job)
		}
	}()
	var limits windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
	limits.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE | windows.JOB_OBJECT_LIMIT_ACTIVE_PROCESS
	limits.BasicLimitInformation.ActiveProcessLimit = 64
	if _, err = windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits))); err != nil {
		return job, 0, err
	}
	attrs, err := windows.NewProcThreadAttributeList(1)
	if err != nil {
		return job, 0, err
	}
	defer attrs.Delete()
	// Atomic assignment prevents children escaping between Start and Assign.
	if err = attrs.Update(0x0002000d, unsafe.Pointer(&job), unsafe.Sizeof(job)); err != nil {
		return job, 0, err
	}
	var startup windows.StartupInfoEx
	startup.Cb = uint32(unsafe.Sizeof(startup))
	startup.ProcThreadAttributeList = attrs.List()
	exe := filepath.Join(installation, "whaleisle.exe")
	app, _ := windows.UTF16PtrFromString(exe)
	command, _ := windows.UTF16PtrFromString(windows.ComposeCommandLine([]string{exe}))
	cwd, _ := windows.UTF16PtrFromString(installation)
	env := environment(installation, profile, trace)
	var pi windows.ProcessInformation
	err = windows.CreateProcess(app, command, nil, nil, false,
		windows.CREATE_UNICODE_ENVIRONMENT|windows.EXTENDED_STARTUPINFO_PRESENT|windows.CREATE_NO_WINDOW,
		&env[0], cwd, &startup.StartupInfo, &pi)
	runtime.KeepAlive(env)
	if err != nil {
		return job, 0, err
	}
	windows.CloseHandle(pi.Thread)
	return job, pi.Process, nil
}

func jobPIDs(job windows.Handle) ([]uint32, error) {
	var data struct {
		Assigned, Listed uint32
		PIDs             [128]uintptr
	}
	var returned uint32
	if err := windows.QueryInformationJobObject(job, windows.JobObjectBasicProcessIdList, uintptr(unsafe.Pointer(&data)), uint32(unsafe.Sizeof(data)), &returned); err != nil {
		return nil, err
	}
	if data.Assigned != data.Listed || data.Listed > uint32(len(data.PIDs)) {
		return nil, errors.New("incomplete installation-health job enumeration")
	}
	pids := make([]uint32, 0, data.Listed)
	for _, pid := range data.PIDs[:data.Listed] {
		pids = append(pids, uint32(pid))
	}
	return pids, nil
}

func stopCandidate(job, process windows.Handle) error {
	pids, err := jobPIDs(job)
	if err != nil {
		_ = windows.TerminateJobObject(job, 1)
		return err
	}
	handles := []windows.Handle{process}
	for _, pid := range pids {
		h, err := windows.OpenProcess(windows.SYNCHRONIZE, false, pid)
		if err == windows.ERROR_INVALID_PARAMETER {
			continue
		}
		if err != nil {
			_ = windows.TerminateJobObject(job, 1)
			return err
		}
		handles = append(handles, h)
		defer windows.CloseHandle(h)
	}
	if err := windows.TerminateJobObject(job, 0); err != nil {
		return err
	}
	deadline := time.Now().Add(5 * time.Second)
	for _, h := range handles {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return errors.New("installation-health process exit deadline expired")
		}
		status, err := windows.WaitForSingleObject(h, uint32(remaining.Milliseconds()+1))
		if err != nil || status != windows.WAIT_OBJECT_0 {
			return fmt.Errorf("installation-health process exit not observed: status=%d error=%v", status, err)
		}
	}
	for time.Now().Before(deadline) {
		pids, err := jobPIDs(job)
		if err != nil {
			return err
		}
		if len(pids) == 0 {
			return nil
		}
		time.Sleep(20 * time.Millisecond)
	}
	return errors.New("installation-health job still contains processes; synthetic profile retained")
}

func removeOwnedProfile(root string) error {
	if filepath.Base(root) == "." || !strings.HasPrefix(filepath.Base(root), "whaleisle-install-health-") {
		return errors.New("invalid owned health profile")
	}
	// Explicit no-follow traversal: junctions are unlinked, never enumerated.
	var remove func(string) error
	remove = func(path string) error {
		p, err := windows.UTF16PtrFromString(path)
		if err != nil {
			return err
		}
		attrs, err := windows.GetFileAttributes(p)
		if err != nil {
			return err
		}
		if attrs&windows.FILE_ATTRIBUTE_DIRECTORY != 0 && attrs&windows.FILE_ATTRIBUTE_REPARSE_POINT == 0 {
			entries, err := os.ReadDir(path)
			if err != nil {
				return err
			}
			for _, entry := range entries {
				if err := remove(filepath.Join(path, entry.Name())); err != nil {
					return err
				}
			}
		}
		return os.Remove(path)
	}
	return remove(root)
}
