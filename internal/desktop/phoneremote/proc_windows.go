//go:build windows

package phoneremote

import (
	"context"
	"os/exec"
	"strconv"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"whaleisle/internal/processutil"
)

type bridgeJob struct {
	mu     sync.Mutex
	handle windows.Handle
}

func prepareBridgeCmd(cmd *exec.Cmd) {
	processutil.BackgroundTree(cmd)
}

func bindBridgeJob(cmd *exec.Cmd) *bridgeJob {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return nil
	}
	var limits windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
	limits.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err := windows.SetInformationJobObject(
		job,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&limits)),
		uint32(unsafe.Sizeof(limits)),
	); err != nil {
		_ = windows.CloseHandle(job)
		return nil
	}
	proc, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(cmd.Process.Pid))
	if err != nil {
		_ = windows.CloseHandle(job)
		return nil
	}
	err = windows.AssignProcessToJobObject(job, proc)
	_ = windows.CloseHandle(proc)
	if err != nil {
		_ = windows.CloseHandle(job)
		return nil
	}
	return &bridgeJob{handle: job}
}

func (j *bridgeJob) terminate() error {
	if j == nil {
		return nil
	}
	j.mu.Lock()
	h := j.handle
	j.mu.Unlock()
	if h == 0 {
		return nil
	}
	return windows.TerminateJobObject(h, 1)
}

func (j *bridgeJob) close() {
	if j == nil {
		return
	}
	j.mu.Lock()
	h := j.handle
	j.handle = 0
	j.mu.Unlock()
	if h != 0 {
		_ = windows.CloseHandle(h)
	}
}

func killBridgeTree(cmd *exec.Cmd, job *bridgeJob) {
	if job != nil && job.terminate() == nil {
		return
	}
	if cmd == nil || cmd.Process == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	kill := processutil.CommandContext(ctx, "taskkill", "/PID", strconv.Itoa(cmd.Process.Pid), "/T", "/F")
	processutil.Background(kill)
	_ = kill.Run()
	_ = cmd.Process.Kill()
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	h, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h)
	status, err := windows.WaitForSingleObject(h, 0)
	return err != nil || status != windows.WAIT_OBJECT_0
}
