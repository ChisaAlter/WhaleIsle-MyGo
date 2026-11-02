//go:build windows

package processutil_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"syscall"
	"testing"
	"time"

	"whaleisle/internal/processutil"
)

type hiddenTreeReport struct {
	PID           int                 `json:"pid"`
	ConsoleWindow uint64              `json:"console_window"`
	Visible       bool                `json:"visible"`
	Child         *hiddenTreeReport   `json:"child,omitempty"`
	Leaf          *consoleChildReport `json:"leaf,omitempty"`
	ChildStderr   string              `json:"child_stderr"`
	ChildExitCode int                 `json:"child_exit_code"`
}

func TestBackgroundTreePreservesAttributesAndClearsConflictingFlags(t *testing.T) {
	cmd := exec.Command("unused-hidden-tree-command")
	attrs := &syscall.SysProcAttr{
		CreationFlags:              syscall.CREATE_NEW_PROCESS_GROUP | 0x04000000 | createNoWindow | 0x8,
		Token:                      syscall.Token(0x4567),
		NoInheritHandles:           true,
		AdditionalInheritedHandles: []syscall.Handle{0x2345},
		ParentProcess:              syscall.Handle(0x5678),
	}
	cmd.SysProcAttr = attrs
	want := *attrs
	want.HideWindow = true
	want.CreationFlags = syscall.CREATE_NEW_PROCESS_GROUP | 0x04000000 | createNewConsole
	for call := 0; call < 2; call++ {
		processutil.BackgroundTree(cmd)
		if cmd.SysProcAttr != attrs || !reflect.DeepEqual(*attrs, want) {
			t.Fatalf("background tree changed unrelated attributes: got %+v want %+v", attrs, want)
		}
	}
}

// Both descendants are ordinary console programs without any HideWindow flag.
// The root owns the hidden console; the descendants must inherit it rather than
// creating a visible terminal. Pipes, stderr, and failure status must still work.
func TestBackgroundTreeKeepsDefaultDescendantsHidden(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := backgroundHelperCommand(ctx, "tree_parent")
	processutil.BackgroundTree(cmd)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	raw, err := cmd.Output()
	if err != nil {
		t.Fatalf("hidden runtime tree: %v; stderr=%q", err, stderr.String())
	}
	var report hiddenTreeReport
	if err = json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("runtime report: %v; output=%q", err, raw)
	}
	if report.ConsoleWindow == 0 || report.Visible || report.Child == nil {
		t.Fatalf("root did not own a hidden console: %+v", report)
	}
	child := report.Child
	if child.ConsoleWindow != report.ConsoleWindow || child.Visible || child.Leaf == nil {
		t.Fatalf("ordinary child did not inherit the hidden console: %+v", child)
	}
	if child.Leaf.ConsoleWindow != report.ConsoleWindow || child.ChildExitCode != 23 || child.ChildStderr != helperStderr {
		t.Fatalf("ordinary grandchild IO/exit/console changed: %+v", child)
	}
	if report.ChildExitCode != 0 || report.PID == child.PID || child.PID == child.Leaf.PID {
		t.Fatalf("runtime descendants or exit status invalid: %+v", report)
	}
	t.Logf("hidden console=%#x inherited by runtime pid=%d, child=%d and grandchild=%d; stderr and exit 23 preserved", report.ConsoleWindow, report.PID, child.PID, child.Leaf.PID)
}

func hiddenConsoleState() (uint64, bool, error) {
	handle, err := consoleWindow()
	if err != nil {
		return 0, false, err
	}
	visible, _, _ := syscall.NewLazyDLL("user32.dll").NewProc("IsWindowVisible").Call(handle)
	return uint64(handle), visible != 0, nil
}

func runHiddenTreeParent() int { return runHiddenTree(false) }
func runHiddenTreeChild() int  { return runHiddenTree(true) }

func runHiddenTree(leaf bool) int {
	handle, visible, err := hiddenConsoleState()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 125
	}
	report := hiddenTreeReport{PID: os.Getpid(), ConsoleWindow: handle, Visible: visible}
	role := "tree_child"
	if leaf {
		role = "child_failure"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := backgroundHelperCommand(ctx, role)
	// Deliberately no process attributes: exercise the descendant package that
	// forgot windowsHide, rather than repeating the parent's configuration.
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	raw, runErr := cmd.Output()
	if runErr != nil {
		if _, ok := runErr.(*exec.ExitError); !ok {
			fmt.Fprintln(os.Stderr, runErr)
			return 125
		}
	}
	report.ChildExitCode = cmd.ProcessState.ExitCode()
	report.ChildStderr = stderr.String()
	if leaf {
		report.Leaf = &consoleChildReport{}
		err = json.Unmarshal(raw, report.Leaf)
	} else {
		report.Child = &hiddenTreeReport{}
		err = json.Unmarshal(raw, report.Child)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 125
	}
	if err = json.NewEncoder(os.Stdout).Encode(report); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 125
	}
	return 0
}
