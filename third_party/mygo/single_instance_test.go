package mygo

import (
	"io"
	"net"
	"os"
	"path/filepath"
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
