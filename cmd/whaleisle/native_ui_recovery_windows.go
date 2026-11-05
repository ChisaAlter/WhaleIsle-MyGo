//go:build windows

package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

type nativeRecoveryProcess struct {
	handle  windows.Handle
	pid     uint32
	created uint64
}

func openNativeRecoveryProcess(pid int, executable string) (*nativeRecoveryProcess, error) {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.SYNCHRONIZE|windows.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		return nil, err
	}
	p := &nativeRecoveryProcess{handle: h, pid: uint32(pid)}
	var name [32768]uint16
	size := uint32(len(name))
	if err = windows.QueryFullProcessImageName(h, 0, &name[0], &size); err != nil {
		p.close()
		return nil, err
	}
	actual, _ := filepath.Abs(windows.UTF16ToString(name[:size]))
	expected, _ := filepath.Abs(executable)
	self, _ := filepath.Abs(nativeRecoveryExecutable())
	if !strings.EqualFold(actual, expected) || !strings.EqualFold(actual, self) {
		p.close()
		return nil, fmt.Errorf("recovery target is not this executable")
	}
	var created, exit, kernel, user windows.Filetime
	if err = windows.GetProcessTimes(h, &created, &exit, &kernel, &user); err != nil {
		p.close()
		return nil, err
	}
	p.created = uint64(created.HighDateTime)<<32 | uint64(created.LowDateTime)
	return p, nil
}
func (p *nativeRecoveryProcess) alive() bool {
	result, err := windows.WaitForSingleObject(p.handle, 0)
	return err == nil && result == uint32(windows.WAIT_TIMEOUT)
}
func (p *nativeRecoveryProcess) close() {
	if p.handle != 0 {
		windows.CloseHandle(p.handle)
		p.handle = 0
	}
}

// Capture handles before stopping anything. They pin the actual process objects
// so a reused numeric PID cannot become a termination target. The independent
// watchdog branch is excluded; no unrelated process is selected by image name.
func (p *nativeRecoveryProcess) stopTree(watchdogPID int) error {
	if !p.alive() {
		return fmt.Errorf("the original desktop exited; no process was stopped")
	}
	// A handle opened after the snapshot must still identify the process
	// that existed when its parent relationship was recorded.
	snapshotAt := uint64(time.Now().UnixNano()/100 + 116444736000000000)
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(snapshot)
	parents := map[uint32]uint32{}
	entry := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	for err = windows.Process32First(snapshot, &entry); err == nil; err = windows.Process32Next(snapshot, &entry) {
		parents[entry.ProcessID] = entry.ParentProcessID
	}
	if err != windows.ERROR_NO_MORE_FILES {
		return err
	}
	depth := map[uint32]int{p.pid: 0}
	excluded := map[uint32]bool{uint32(watchdogPID): true}
	for changed := true; changed; {
		changed = false
		for pid, parent := range parents {
			if excluded[parent] && !excluded[pid] {
				excluded[pid] = true
				changed = true
			}
			if d, ok := depth[parent]; ok {
				if _, seen := depth[pid]; !seen && pid != p.pid {
					depth[pid] = d + 1
					changed = true
				}
			}
		}
	}
	type target struct {
		handle  windows.Handle
		depth   int
		pid     uint32
		created uint64
	}
	var targets []target
	createdByPID := map[uint32]uint64{p.pid: p.created}
	defer func() {
		for _, t := range targets {
			windows.CloseHandle(t.handle)
		}
	}()
	for pid, d := range depth {
		if pid == p.pid || excluded[pid] {
			continue
		}
		h, openErr := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.SYNCHRONIZE|windows.PROCESS_TERMINATE, false, pid)
		if openErr != nil {
			if openErr == windows.ERROR_INVALID_PARAMETER {
				continue
			}
			return fmt.Errorf("cannot stop owned child %d: %w", pid, openErr)
		}
		var created, exit, kernel, user windows.Filetime
		if err := windows.GetProcessTimes(h, &created, &exit, &kernel, &user); err != nil {
			windows.CloseHandle(h)
			return err
		}
		instant := uint64(created.HighDateTime)<<32 | uint64(created.LowDateTime)
		if instant < p.created || instant > snapshotAt {
			windows.CloseHandle(h)
			return fmt.Errorf("owned child identity changed")
		}
		targets = append(targets, target{handle: h, depth: d, pid: pid, created: instant})
		createdByPID[pid] = instant
	}
	for _, t := range targets {
		parentCreated, known := createdByPID[parents[t.pid]]
		if !known || parentCreated > t.created {
			return fmt.Errorf("owned child ancestry changed; no process was stopped")
		}
	}
	// Parent first freezes further child creation; terminate captured descendants
	// from leaves upward, then wait for every owned object before a restart.
	if err := windows.TerminateProcess(p.handle, 1); err != nil {
		return err
	}
	for maxDepth := len(depth); maxDepth > 0; maxDepth-- {
		for _, t := range targets {
			if t.depth == maxDepth {
				if result, _ := windows.WaitForSingleObject(t.handle, 0); result == uint32(windows.WAIT_TIMEOUT) {
					if err := windows.TerminateProcess(t.handle, 1); err != nil {
						return err
					}
				}
			}
		}
	}
	for _, t := range targets {
		if result, err := windows.WaitForSingleObject(t.handle, 5000); err != nil || result == uint32(windows.WAIT_TIMEOUT) {
			return fmt.Errorf("an owned runtime has not stopped")
		}
	}
	if result, err := windows.WaitForSingleObject(p.handle, 5000); err != nil || result == uint32(windows.WAIT_TIMEOUT) {
		return fmt.Errorf("the original desktop has not stopped")
	}
	return nil
}
