//go:build windows

package harness

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

const taskkillTimeout = 5 * time.Second

// childJob owns a Windows Job Object configured with KILL_ON_JOB_CLOSE so the
// Host process tree (including grandchildren whose parent exited early) dies
// together on stop, failure, or unexpected desktop crash.
type childJob struct {
	mu     sync.Mutex
	handle windows.Handle
}

func bindChildJob(cmd *exec.Cmd) *childJob {
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
	return &childJob{handle: job}
}

func (j *childJob) terminate() error {
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

func (j *childJob) close() {
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

func detach(cmd *exec.Cmd) {
	processutil.BackgroundTree(cmd)
}

func terminateWithJob(cmd *exec.Cmd, job *childJob, force bool) error {
	if job != nil {
		if err := job.terminate(); err == nil {
			return nil
		}
	}
	return terminate(cmd, force)
}

// terminate force-ends the child's process tree; force is ignored. The Host
// runs in a hidden console with no window to receive taskkill's close request,
// so a plain taskkill never ends it and would only hold Stop for stopGrace.
// Electron's killTree (reference/electron/src/main/dsh.js) also uses /T /F.
func terminate(cmd *exec.Cmd, _ bool) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), taskkillTimeout)
	defer cancel()
	kill := processutil.CommandContext(ctx, "taskkill", "/PID", strconv.Itoa(cmd.Process.Pid), "/T", "/F")
	processutil.Background(kill)
	return kill.Run()
}
