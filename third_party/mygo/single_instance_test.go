package mygo

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// unixSocketDir returns a short temp dir for a unix socket, which has a ~100
// byte path limit on macOS.
func unixSocketDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "mygo-si")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func listenUnix(t *testing.T, path string) *net.UnixListener {
	t.Helper()
	ln, err := net.Listen("unix", path)
	if err != nil {
		t.Skipf("unix sockets are unavailable here: %v", err)
	}
	return ln.(*net.UnixListener)
}

// TestSecondInstanceHandshakeAcknowledged: a live owner answers, so the second
// instance hands its command line over and exits.
func TestSecondInstanceHandshakeAcknowledged(t *testing.T) {
	path := filepath.Join(unixSocketDir(t), "s.sock")
	ln := listenUnix(t, path)
	defer ln.Close()
	received := make(chan []byte, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		data, _ := io.ReadAll(io.LimitReader(conn, 1<<20))
		received <- data
		_, _ = conn.Write([]byte(secondInstanceAck))
	}()

	conn, err := net.DialTimeout("unix", path, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	msg := []byte(`{"args":["--from-second"],"workingDir":"C:\\work"}`)
	if !handOverSecondInstance(conn, msg, secondInstanceHandshakeTimeout) {
		t.Fatal("a live owner that acknowledged was not treated as running")
	}
	select {
	case got := <-received:
		if string(got) != string(msg) {
			t.Fatalf("owner read %q, want %q", got, msg)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the owner never read the command line")
	}
}

// TestSecondInstanceHandshakeRejectsSilentOwner is the round145 defect: a socket
// that accepts but never answers (a crash left it, or a surviving child
// inherited the listening handle) must not be mistaken for a running app.
func TestSecondInstanceHandshakeRejectsSilentOwner(t *testing.T) {
	path := filepath.Join(unixSocketDir(t), "s.sock")
	ln := listenUnix(t, path)
	defer ln.Close()
	// Never accept: the connection is queued and no one answers.

	conn, err := net.DialTimeout("unix", path, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	answered := handOverSecondInstance(conn, []byte("{}"), 150*time.Millisecond)
	waited := time.Since(start)
	if answered {
		t.Fatal("a silent listener was accepted as a live owner")
	}
	if waited < 100*time.Millisecond {
		t.Fatalf("the handshake returned after %v, before its deadline", waited)
	}
	if waited > 2*time.Second {
		t.Fatalf("the handshake waited %v, want about its 150ms deadline", waited)
	}

	// A listener that reads the message but sends nothing is just as silent.
	ln2 := listenUnix(t, filepath.Join(unixSocketDir(t), "s.sock"))
	defer ln2.Close()
	go func() {
		if c, err := ln2.Accept(); err == nil {
			_, _ = io.ReadAll(io.LimitReader(c, 1<<20))
			c.Close()
		}
	}()
	conn2, err := net.DialTimeout("unix", ln2.Addr().String(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if handOverSecondInstance(conn2, []byte("{}"), 150*time.Millisecond) {
		t.Fatal("a listener that closes without acknowledging was accepted as an owner")
	}
}

// TestSilentListenerNeverClaimsTheLock calls the whole lock path against a
// socket whose owner never answers: the app must still start (true), never exit
// silently as a second instance.
func TestSilentListenerNeverClaimsTheLock(t *testing.T) {
	name := "MyGoSilentOwnerTest"
	App.SetName(name)
	path := singleInstanceSocket(singleInstanceKey(name))
	_ = os.Remove(path)
	silent := listenUnix(t, path)
	defer func() {
		silent.Close()
		releaseSingleInstanceLock()
		_ = os.Remove(path)
	}()
	// Never accept, so the owner never acknowledges.

	start := time.Now()
	locked := App.RequestSingleInstanceLock()
	waited := time.Since(start)
	if !locked {
		t.Fatal("a socket without a live owner made the launch exit as a second instance")
	}
	if waited > 10*time.Second {
		t.Fatalf("taking the lock over waited %v", waited)
	}
	// Handlers registered by the app still receive forwarded arguments once a
	// real owner exists; here we only assert the launch was not blocked.
	t.Cleanup(func() {
		singleInstance.Lock()
		singleInstance.locked = false
		singleInstance.Unlock()
	})
}

// TestSweepStaleSocketsRemovesOnlyUnownedFiles is the AYA-30 accumulation: the
// socket of a key whose owner died stays forever, because only its owner ever
// unlinks it. The sweep has to remove those and nothing else.
func TestSweepStaleSocketsRemovesOnlyUnownedFiles(t *testing.T) {
	dir := unixSocketDir(t)
	uid := strconv.Itoa(os.Getuid())
	socket := func(tag string) string { return filepath.Join(dir, "mygo-"+uid+"-"+tag+".sock") }
	live, stale, own := socket("aaaaaaaaaaaa"), socket("bbbbbbbbbbbb"), socket("cccccccccccc")

	liveListener := listenUnix(t, live)
	defer liveListener.Close()
	ownListener := listenUnix(t, own)
	defer ownListener.Close()
	// An unclean exit: the file stays, and nobody answers it any more.
	dead := listenUnix(t, stale)
	dead.SetUnlinkOnClose(false)
	if err := dead.Close(); err != nil {
		t.Fatal(err)
	}
	foreign := filepath.Join(dir, "not-a-mygo-socket.sock")
	if err := os.WriteFile(foreign, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	var logged bytes.Buffer
	previous, flags := log.Writer(), log.Flags()
	log.SetOutput(&logged)
	log.SetFlags(0)
	t.Cleanup(func() { log.SetOutput(previous); log.SetFlags(flags) })

	if sweep := sweepStaleSockets(dir, own, 0); sweep.Removed != 1 || sweep.Examined != 2 || sweep.InUse != 1 {
		t.Fatalf("sweep = %+v, want one removed out of two examined with one in use", sweep)
	}
	if line := logged.String(); !strings.Contains(line, "examined 2, removed 1, in use 1") {
		t.Fatalf("the launch log does not report the sweep: %q", line)
	}
	for _, kept := range []string{live, own, foreign} {
		if _, err := os.Lstat(kept); err != nil {
			t.Fatalf("sweep removed %s, which it does not own: %v", filepath.Base(kept), err)
		}
	}
	if _, err := os.Lstat(stale); !os.IsNotExist(err) {
		t.Fatalf("the stale socket survived the sweep: %v", err)
	}
}

// TestSweepStaleSocketsSparesAFreshFile keeps the sweep away from a socket a
// launch may be binding right now: only files older than the grace are probed.
func TestSweepStaleSocketsSparesAFreshFile(t *testing.T) {
	dir := unixSocketDir(t)
	path := filepath.Join(dir, "mygo-"+strconv.Itoa(os.Getuid())+"-dddddddddddd.sock")
	ln := listenUnix(t, path)
	ln.SetUnlinkOnClose(false)
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}
	if sweep := sweepStaleSockets(dir, "", time.Minute); sweep.Removed != 0 || sweep.InUse != 0 {
		t.Fatalf("sweep = %+v, want a fresh file left alone", sweep)
	}
	if _, err := os.Lstat(path); err != nil {
		t.Fatalf("a fresh socket file was removed: %v", err)
	}
}

// TestSingleInstanceConcurrencyHelper is invoked as a subprocess by
// TestSingleInstanceConcurrency and TestSingleInstanceLongTempDir.
func TestSingleInstanceConcurrencyHelper(t *testing.T) {
	name := os.Getenv("MYGO_SI_TEST_NAME")
	if name == "" {
		return
	}
	// Wait until the parent removes the start gate (or closes stdin for non-gated).
	if gate := os.Getenv("MYGO_SI_TEST_GATE"); gate != "" {
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			if _, err := os.Stat(gate); os.IsNotExist(err) {
				break
			}
			time.Sleep(2 * time.Millisecond)
		}
	}
	App.SetName(name)
	if App.RequestSingleInstanceLock() {
		fmt.Println("RESULT:WINNER")
		// Hold the lock until parent closes stdin, so all competitors finish
		// handing over before the winner exits.
		_, _ = io.ReadAll(os.Stdin)
		releaseSingleInstanceLock()
		return
	}
	fmt.Println("RESULT:FORWARDED")
}

// TestSingleInstanceConcurrency launches 5 competing OS processes at the same
// time and asserts that exactly 1 wins the single-instance lock while the
// other 4 complete handover to the winner.
func TestSingleInstanceConcurrency(t *testing.T) {
	const competitors = 5
	appName := fmt.Sprintf("MyGoConcurrencyTest-%d-%d", os.Getpid(), time.Now().UnixNano())
	gate := filepath.Join(t.TempDir(), "start.gate")
	if err := os.WriteFile(gate, []byte("wait"), 0o600); err != nil {
		t.Fatal(err)
	}

	type proc struct {
		cmd   *exec.Cmd
		stdin io.WriteCloser
		out   bytes.Buffer
		done  chan error
	}
	procs := make([]*proc, competitors)
	for i := range competitors {
		cmd := exec.Command(os.Args[0], "-test.run=^TestSingleInstanceConcurrencyHelper$")
		cmd.Env = append(os.Environ(),
			"MYGO_SI_TEST_NAME="+appName,
			"MYGO_SI_TEST_GATE="+gate,
		)
		stdin, err := cmd.StdinPipe()
		if err != nil {
			t.Fatal(err)
		}
		p := &proc{cmd: cmd, stdin: stdin, done: make(chan error, 1)}
		cmd.Stdout = &p.out
		cmd.Stderr = &p.out
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		procs[i] = p
		go func(p *proc) { p.done <- p.cmd.Wait() }(p)
	}

	// Give children a moment to reach the gate, then release all 5 together.
	time.Sleep(150 * time.Millisecond)
	_ = os.Remove(gate)

	// Wait for the 4 losers to exit on their own, then close all stdins so the
	// winner releases and exits cleanly.
	var mu sync.Mutex
	exited := 0
	deadline := time.After(15 * time.Second)
	for exited < competitors-1 {
		select {
		case <-deadline:
			for _, p := range procs {
				_ = p.stdin.Close()
			}
			t.Fatalf("timed out waiting for competitors to finish (only %d/%d exited)", exited, competitors-1)
		default:
		}
		progressed := false
		for _, p := range procs {
			select {
			case err := <-p.done:
				if err != nil {
					for _, q := range procs {
						_ = q.stdin.Close()
					}
					t.Fatalf("competitor failed: %v\noutput:\n%s", err, p.out.String())
				}
				p.done = nil
				mu.Lock()
				exited++
				mu.Unlock()
				progressed = true
			default:
			}
		}
		if !progressed {
			time.Sleep(10 * time.Millisecond)
		}
	}

	for _, p := range procs {
		_ = p.stdin.Close()
	}
	for _, p := range procs {
		if p.done == nil {
			continue
		}
		select {
		case err := <-p.done:
			if err != nil {
				t.Fatalf("winner failed: %v\noutput:\n%s", err, p.out.String())
			}
		case <-time.After(5 * time.Second):
			_ = p.cmd.Process.Kill()
			t.Fatalf("winner did not exit cleanly\noutput:\n%s", p.out.String())
		}
	}

	winners, forwarded := 0, 0
	for i, p := range procs {
		out := p.out.String()
		switch {
		case strings.Contains(out, "RESULT:WINNER"):
			winners++
		case strings.Contains(out, "RESULT:FORWARDED"):
			forwarded++
		default:
			t.Fatalf("competitor %d produced unexpected output:\n%s", i, out)
		}
	}
	if winners != 1 || forwarded != competitors-1 {
		t.Fatalf("winners=%d forwarded=%d, want 1 and %d", winners, forwarded, competitors-1)
	}
}

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// TestSingleInstanceLongTempDir verifies that a long (>90 char) TEMP directory
// does not silently disable the single-instance lock (e.g. by falling back to
// non-existent "/tmp" on Windows).
func TestSingleInstanceLongTempDir(t *testing.T) {
	base := t.TempDir()
	longDir := filepath.Join(base, strings.Repeat("a", 32), strings.Repeat("b", 32), strings.Repeat("c", 32))
	if err := os.MkdirAll(longDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if len(longDir) < 90 {
		t.Fatalf("longDir len = %d, want >= 90", len(longDir))
	}

	t.Setenv("TEMP", longDir)
	t.Setenv("TMP", longDir)
	t.Setenv("TMPDIR", longDir)
	t.Setenv("XDG_RUNTIME_DIR", "")

	appName := fmt.Sprintf("MyGoLongTempTest-%d-%d", os.Getpid(), time.Now().UnixNano())
	sockPath := singleInstanceSocket(singleInstanceKey(appName))
	if runtime.GOOS == "windows" {
		if strings.HasPrefix(sockPath, "/tmp") || strings.HasPrefix(sockPath, `\tmp`) {
			t.Fatalf("Windows socket path fell back to /tmp: %q", sockPath)
		}
		if len(sockPath) >= 107 {
			t.Fatalf("Windows socket path too long (%d bytes): %q", len(sockPath), sockPath)
		}
	}

	// Launch a primary process and a second process both using longDir as TEMP
	// and assert the second process forwards to the primary instead of winning.
	makeCmd := func() (*exec.Cmd, io.WriteCloser, *syncBuffer) {
		cmd := exec.Command(os.Args[0], "-test.run=^TestSingleInstanceConcurrencyHelper$")
		var env []string
		for _, kv := range os.Environ() {
			k, _, _ := strings.Cut(kv, "=")
			switch strings.ToUpper(k) {
			case "TEMP", "TMP", "TMPDIR", "XDG_RUNTIME_DIR":
				continue
			}
			env = append(env, kv)
		}
		cmd.Env = append(env,
			"TEMP="+longDir,
			"TMP="+longDir,
			"TMPDIR="+longDir,
			"XDG_RUNTIME_DIR=",
			"MYGO_SI_TEST_NAME="+appName,
		)
		stdin, err := cmd.StdinPipe()
		if err != nil {
			t.Fatal(err)
		}
		out := &syncBuffer{}
		cmd.Stdout = out
		cmd.Stderr = out
		return cmd, stdin, out
	}

	cmd1, stdin1, out1 := makeCmd()
	if err := cmd1.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = stdin1.Close()
		_ = cmd1.Wait()
	}()

	// Wait until primary prints RESULT:WINNER.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(out1.String(), "RESULT:WINNER") {
			break
		}
		time.Sleep(15 * time.Millisecond)
	}
	if !strings.Contains(out1.String(), "RESULT:WINNER") {
		t.Fatalf("primary did not acquire lock under long TEMP %q:\n%s", longDir, out1.String())
	}

	cmd2, stdin2, out2 := makeCmd()
	_ = stdin2.Close()
	if err := cmd2.Run(); err != nil {
		t.Fatalf("second instance failed: %v\noutput:\n%s", err, out2.String())
	}
	if !strings.Contains(out2.String(), "RESULT:FORWARDED") {
		t.Fatalf("second instance did not forward under long TEMP %q (got %q)", longDir, out2.String())
	}
}
