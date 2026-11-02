//go:build windows

package processutil_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"syscall"
	"testing"
	"time"

	"whaleisle/internal/processutil"
)

const (
	createNewConsole = 0x00000010
	createNoWindow   = 0x08000000
	helperEnv        = "WHALEISLE_PROCESSUTIL_TEST_HELPER"
	helperStdout     = "processutil stdout"
	helperStderr     = "processutil stderr\n"
)

func TestBackgroundInitializesAttributes(t *testing.T) {
	cmd := exec.Command("unused-processutil-test-command")
	want := &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}

	processutil.Background(cmd)
	if !reflect.DeepEqual(cmd.SysProcAttr, want) {
		t.Fatalf("attributes = %+v, want %+v", cmd.SysProcAttr, want)
	}
	first := cmd.SysProcAttr
	processutil.Background(cmd)
	if cmd.SysProcAttr != first || !reflect.DeepEqual(cmd.SysProcAttr, want) {
		t.Fatalf("second call changed attributes or their pointer: %+v", cmd.SysProcAttr)
	}
}

func TestBackgroundPreservesAttributes(t *testing.T) {
	for _, alreadyHidden := range []bool{false, true} {
		t.Run(fmt.Sprintf("already_hidden_%t", alreadyHidden), func(t *testing.T) {
			processAttributes := &syscall.SecurityAttributes{Length: 24, InheritHandle: 1}
			threadAttributes := &syscall.SecurityAttributes{Length: 24, SecurityDescriptor: 0x1234}
			inheritedHandles := []syscall.Handle{0x2345, 0x3456}
			attrs := &syscall.SysProcAttr{
				HideWindow:                 alreadyHidden,
				CmdLine:                    "preserved custom command line",
				CreationFlags:              syscall.CREATE_NEW_PROCESS_GROUP | 0x04000000,
				Token:                      syscall.Token(0x4567),
				ProcessAttributes:          processAttributes,
				ThreadAttributes:           threadAttributes,
				NoInheritHandles:           true,
				AdditionalInheritedHandles: inheritedHandles,
				ParentProcess:              syscall.Handle(0x5678),
			}
			// These sentinel tokens/handles are only inspected, never passed to Start.
			cmd := exec.Command("unused-processutil-test-command")
			cmd.SysProcAttr = attrs
			want := *attrs
			want.HideWindow = true
			want.CreationFlags |= createNoWindow
			wantProcessAttributes, wantThreadAttributes := *processAttributes, *threadAttributes
			want.ProcessAttributes = &wantProcessAttributes
			want.ThreadAttributes = &wantThreadAttributes
			want.AdditionalInheritedHandles = append([]syscall.Handle(nil), inheritedHandles...)

			for call := 1; call <= 2; call++ {
				processutil.Background(cmd)
				if cmd.SysProcAttr != attrs {
					t.Fatalf("call %d replaced the caller's SysProcAttr pointer", call)
				}
				if !reflect.DeepEqual(*cmd.SysProcAttr, want) {
					t.Fatalf("call %d: attributes = %+v, want %+v", call, *cmd.SysProcAttr, want)
				}
				if attrs.ProcessAttributes != processAttributes || attrs.ThreadAttributes != threadAttributes {
					t.Fatalf("call %d replaced security attribute pointers", call)
				}
				if &attrs.AdditionalInheritedHandles[0] != &inheritedHandles[0] {
					t.Fatalf("call %d replaced inherited handles' backing array", call)
				}
			}
		})
	}
}

type consoleChildReport struct {
	ConsoleWindow uint64 `json:"console_window"`
	Visible       bool   `json:"visible"`
	PID           int    `json:"pid"`
	Payload       string `json:"payload"`
}

type consoleChildResult struct {
	Name     string `json:"name"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
	Error    string `json:"error,omitempty"`
}

type consoleParentReport struct {
	ConsoleWindow uint64               `json:"console_window"`
	Children      []consoleChildResult `json:"children"`
}

func TestBackgroundDoesNotCreateConsole(t *testing.T) {
	parents := []struct {
		name       string
		flags      uint32
		hasConsole bool
	}{
		{name: "with_parent_console", flags: createNewConsole, hasConsole: true},
		{name: "without_parent_console", flags: createNoWindow, hasConsole: false},
	}
	for _, parent := range parents {
		t.Run(parent.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			cmd := backgroundHelperCommand(ctx, "parent")
			// Construct the parent independently of Background. SW_HIDE keeps the
			// positive-control console invisible without suppressing its allocation.
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: parent.flags}
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			stdout, err := cmd.Output()
			if err != nil {
				t.Fatalf("parent helper: %v; stdout=%q; stderr=%q", err, stdout, stderr.String())
			}
			var report consoleParentReport
			if err := json.Unmarshal(stdout, &report); err != nil {
				t.Fatalf("parent report: %v; stdout=%q; stderr=%q", err, stdout, stderr.String())
			}
			if got := report.ConsoleWindow != 0; got != parent.hasConsole {
				t.Fatalf("parent console handle = %#x; want console=%t", report.ConsoleWindow, parent.hasConsole)
			}
			t.Logf("parent GetConsoleWindow=%#x", report.ConsoleWindow)
			cases := []struct {
				name       string
				exitCode   int
				hasConsole bool
			}{
				{name: "hidden_new_console_control", exitCode: 0, hasConsole: true},
				{name: "background_nil_attributes", exitCode: 0, hasConsole: false},
				{name: "background_existing_flags_nonzero_exit", exitCode: 23, hasConsole: false},
			}
			if len(report.Children) != len(cases) {
				t.Fatalf("received %d child results, want %d: %+v", len(report.Children), len(cases), report)
			}
			for i, want := range cases {
				t.Run(want.name, func(t *testing.T) {
					got := report.Children[i]
					if got.Name != want.name || got.Error != "" {
						t.Fatalf("unexpected child result: %+v", got)
					}
					if got.ExitCode != want.exitCode {
						t.Fatalf("exit code = %d, want %d; stderr=%q", got.ExitCode, want.exitCode, got.Stderr)
					}
					if got.Stderr != helperStderr {
						t.Errorf("stderr = %q, want %q", got.Stderr, helperStderr)
					}
					var child consoleChildReport
					if err := json.Unmarshal([]byte(got.Stdout), &child); err != nil {
						t.Fatalf("child stdout is not readable JSON: %v; stdout=%q", err, got.Stdout)
					}
					if child.Payload != helperStdout || child.PID <= 0 {
						t.Errorf("unexpected stdout payload: %+v", child)
					}
					if hasConsole := child.ConsoleWindow != 0; hasConsole != want.hasConsole {
						t.Errorf("child GetConsoleWindow=%#x, want console=%t", child.ConsoleWindow, want.hasConsole)
					}
					t.Logf("child pid=%d GetConsoleWindow=%#x exit=%d; stdout and stderr read", child.PID, child.ConsoleWindow, got.ExitCode)
				})
			}
		})
	}
}

func backgroundHelperCommand(ctx context.Context, role string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestBackgroundHelperProcess$")
	// exec.Cmd removes duplicate environment keys, keeping the final value.
	cmd.Env = append(os.Environ(), helperEnv+"="+role)
	return cmd
}

// TestBackgroundHelperProcess runs only in explicitly gated subprocesses. The
// normal Go test executable is a real Windows console-subsystem application.
func TestBackgroundHelperProcess(t *testing.T) {
	role := os.Getenv(helperEnv)
	if role == "" {
		t.Skip("subprocess helper")
	}
	var exitCode int
	switch role {
	case "parent":
		exitCode = runConsoleParent()
	case "tree_parent":
		exitCode = runHiddenTreeParent()
	case "tree_child":
		exitCode = runHiddenTreeChild()
	case "child_success", "child_failure":
		exitCode = runConsoleChild(role == "child_failure")
	default:
		fmt.Fprintf(os.Stderr, "unknown processutil helper role %q\n", role)
		exitCode = 125
	}
	os.Exit(exitCode)
}

func runConsoleParent() int {
	handle, err := consoleWindow()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 125
	}
	report := consoleParentReport{ConsoleWindow: uint64(handle)}
	cases := []struct {
		name       string
		role       string
		background bool
		existing   bool
	}{
		{name: "hidden_new_console_control", role: "child_success"},
		{name: "background_nil_attributes", role: "child_success", background: true},
		{name: "background_existing_flags_nonzero_exit", role: "child_failure", background: true, existing: true},
	}
	for _, tc := range cases {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		cmd := backgroundHelperCommand(ctx, tc.role)
		if tc.existing {
			cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
		} else if !tc.background {
			// A console-less parent can pass that state to a normal child.
			// Request a fresh hidden console to make this a positive control.
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNewConsole}
		}
		if tc.background {
			processutil.Background(cmd)
		}
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		runErr := cmd.Run()
		result := consoleChildResult{Name: tc.name, Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: -1}
		if cmd.ProcessState != nil {
			result.ExitCode = cmd.ProcessState.ExitCode()
		}
		var exitErr *exec.ExitError
		if runErr != nil && !errors.As(runErr, &exitErr) {
			result.Error = runErr.Error()
		}
		if ctx.Err() != nil {
			result.Error = ctx.Err().Error()
		}
		cancel()
		report.Children = append(report.Children, result)
	}
	if err := json.NewEncoder(os.Stdout).Encode(report); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 125
	}
	return 0
}

func runConsoleChild(fail bool) int {
	handle, err := consoleWindow()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 125
	}
	visible, _, _ := syscall.NewLazyDLL("user32.dll").NewProc("IsWindowVisible").Call(handle)
	report := consoleChildReport{ConsoleWindow: uint64(handle), Visible: visible != 0, PID: os.Getpid(), Payload: helperStdout}
	if err := json.NewEncoder(os.Stdout).Encode(report); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 125
	}
	fmt.Fprint(os.Stderr, helperStderr)
	if fail {
		return 23
	}
	return 0
}

func consoleWindow() (uintptr, error) {
	proc := syscall.NewLazyDLL("kernel32.dll").NewProc("GetConsoleWindow")
	if err := proc.Find(); err != nil {
		return 0, err
	}
	// A zero HWND is the successful result when no console is attached;
	// GetConsoleWindow does not define GetLastError for that case.
	handle, _, _ := proc.Call()
	return handle, nil
}
