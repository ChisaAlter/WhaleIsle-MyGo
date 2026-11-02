//go:build windows

package processutil_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"whaleisle/internal/processutil"
)

type nodeLeafResult struct {
	PID      int    `json:"pid"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exitCode"`
}

type nodeTreeResult struct {
	ParentPID int            `json:"parentPid"`
	Direct    nodeLeafResult `json:"direct"`
	Nested    struct {
		NodePID int            `json:"nodePid"`
		Leaf    nodeLeafResult `json:"leaf"`
	} `json:"nested"`
	NestedProcess nodeLeafResult `json:"nestedProcess"`
}

// Opt-in selects the same real bundled Node that runs the desktop Host. Neither
// Node level requests windowsHide: this reproduces the previously unprotected
// descendant launch, without ever opening a visible positive-control window.
func TestBackgroundTreeKeepsNodeExecFileDescendantsHidden(t *testing.T) {
	node := os.Getenv("WHALEISLE_PROCESSUTIL_NODE")
	if node == "" {
		t.Skip("set WHALEISLE_PROCESSUTIL_NODE to the actual desktop Host Node")
	}
	script, err := filepath.Abs(filepath.Join("testdata", "node-tree.mjs"))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, node, script, os.Args[0])
	processutil.BackgroundTree(cmd)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	raw, err := cmd.Output()
	if err != nil {
		t.Fatalf("real Node tree failed: %v; stderr=%q", err, stderr.String())
	}
	var report nodeTreeResult
	if err = json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("real Node tree report: %v; output=%q", err, raw)
	}
	if report.ParentPID != cmd.Process.Pid || report.Nested.NodePID != report.NestedProcess.PID || report.NestedProcess.ExitCode != 0 || report.NestedProcess.Stderr != "" {
		t.Fatalf("Node ownership or nested IO changed: %+v", report)
	}
	var sharedConsole uint64
	for _, tc := range []struct {
		name   string
		result nodeLeafResult
	}{{"direct", report.Direct}, {"nested", report.Nested.Leaf}} {
		var child consoleChildReport
		if err = json.Unmarshal([]byte(tc.result.Stdout), &child); err != nil {
			t.Fatalf("%s leaf report: %v", tc.name, err)
		}
		if child.PID != tc.result.PID || child.Payload != helperStdout || child.ConsoleWindow == 0 || child.Visible || tc.result.ExitCode != 23 || tc.result.Stderr != helperStderr {
			t.Fatalf("%s Node descendant visibility, IO or status invalid: %+v; leaf=%+v", tc.name, tc.result, child)
		}
		if sharedConsole == 0 {
			sharedConsole = child.ConsoleWindow
		} else if child.ConsoleWindow != sharedConsole {
			t.Fatalf("Node child did not inherit console: %#x != %#x", child.ConsoleWindow, sharedConsole)
		}
		t.Logf("%s Node execFile leaf pid=%d console=%#x visible=%t stderr preserved exit=%d", tc.name, child.PID, child.ConsoleWindow, child.Visible, tc.result.ExitCode)
	}
}
