package mygo

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json/v2"
	"errors"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"
)

var singleInstance struct {
	sync.Mutex
	listener net.Listener
	path     string
	socket   os.FileInfo // identifies our socket file at path
	locked   bool
	handlers listeners[func([]string, string)]
}

type secondInstanceMessage struct {
	Args       []string `json:"args"`
	WorkingDir string   `json:"workingDir"`
}

// secondInstanceAck is what a live owner writes once it has taken a second
// instance's command line. It is the proof that the socket has a running owner:
// a socket file left behind by a crash, or a listening handle that a surviving
// child process inherited, accepts the connection but never answers, and must
// not be mistaken for a running app. Treating that as "already running" made
// the launch exit with no window (round145).
const secondInstanceAck = "ok\n"

// secondInstanceHandshakeTimeout bounds the wait for that proof. A live owner
// answers at once; only a stale listener runs into it.
const secondInstanceHandshakeTimeout = 2 * time.Second

// handOverSecondInstance gives msg to the owner on conn and reports whether it
// acknowledged. conn is closed either way.
func handOverSecondInstance(conn net.Conn, msg []byte, timeout time.Duration) bool {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))
	if _, err := conn.Write(msg); err != nil {
		return false
	}
	if uc, ok := conn.(*net.UnixConn); ok {
		if err := uc.CloseWrite(); err != nil {
			return false
		}
	}
	ack := make([]byte, len(secondInstanceAck))
	if _, err := io.ReadFull(conn, ack); err != nil {
		return false
	}
	return string(ack) == secondInstanceAck
}

// RequestSingleInstanceLock makes sure a single instance of the app runs.
// It returns true in the first instance. In any later instance it returns
// false after forwarding its command line arguments and working directory to
// the first one (see OnSecondInstance); that instance should then exit:
//
//	if !mygo.App.RequestSingleInstanceLock() {
//		return
//	}
//
// The lock is identified by the application name and, in a packaged app,
// its identifier, and released on quit.
func (a *Application) RequestSingleInstanceLock() bool {
	if os.Getenv("MYGO_GENERATE") != "" {
		return true // Run will only write the TypeScript client.
	}
	singleInstance.Lock()
	defer singleInstance.Unlock()
	if singleInstance.locked {
		return true
	}
	path := singleInstanceSocket(singleInstanceKey(a.Name()))
	wd, _ := os.Getwd()
	msg, _ := json.Marshal(secondInstanceMessage{Args: os.Args[1:], WorkingDir: wd})
	var lastErr error
	for attempt := 0; attempt < 5; attempt++ {
		if conn, err := net.DialTimeout("unix", path, time.Second); err == nil {
			if handOverSecondInstance(conn, msg, secondInstanceHandshakeTimeout) {
				if launchedByDev() {
					// mygo dev would only see the app exit.
					log.Printf("mygo: %s is already running: handed the command line over to it (RequestSingleInstanceLock)", a.Name())
				}
				return false
			}
			// The socket answered but no owner acknowledged: a crash left it
			// behind, or a surviving child inherited the listening handle. Fall
			// through and take the lock over, so the launch still shows a window.
			log.Printf("mygo: %s: the single-instance socket has no live owner; taking it over", a.Name())
		}
		// Nobody answers: remove a stale socket left by a crash and take
		// over. If another instance wins the race, Listen fails and the next
		// attempt connects to it.
		_ = os.Remove(path)
		ln, err := net.Listen("unix", path)
		if err == nil {
			// The socket file is removed on release, and only if it is
			// still ours: an instance that found it stale may have
			// replaced it.
			ln.(*net.UnixListener).SetUnlinkOnClose(false)
			singleInstance.socket, _ = os.Stat(path)
			singleInstance.listener, singleInstance.path, singleInstance.locked = ln, path, true
			go acceptSecondInstances(ln)
			a.OnQuit(releaseSingleInstanceLock)
			return true
		}
		// Windows reports WSAEADDRINUSE.
		if !errors.Is(err, syscall.EADDRINUSE) && !errors.Is(err, syscall.Errno(10048)) {
			// Locking is not possible here (e.g. read-only temp dir):
			// behave as if this were the only instance.
			return true
		}
		lastErr = err
		time.Sleep(time.Duration(attempt+1) * 20 * time.Millisecond)
	}
	// A stale listener that a surviving process still holds keeps the path
	// bound. The app must still start: forwarding is unavailable until that
	// process is gone, but a silent exit with no window is worse.
	log.Printf("mygo: %s starts without the single-instance socket: %v", a.Name(), lastErr)
	return true
}

// OnSecondInstance is called in the first instance when another instance
// was started and called RequestSingleInstanceLock. Apps usually focus
// their main window here.
func (a *Application) OnSecondInstance(fn func(args []string, workingDir string)) (off func()) {
	return singleInstance.handlers.add(fn, false)
}

func acceptSecondInstances(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		go func() {
			defer conn.Close()
			_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
			data, err := io.ReadAll(io.LimitReader(conn, 1<<20))
			if err != nil {
				return
			}
			var msg secondInstanceMessage
			if json.Unmarshal(data, &msg) != nil {
				return
			}
			// Prove to the second instance that this owner is alive before the
			// main thread runs its handlers: a slow app must not look dead.
			_, _ = conn.Write([]byte(secondInstanceAck))
			postMain(func() {
				for _, fn := range singleInstance.handlers.snapshot() {
					fn(msg.Args, msg.WorkingDir)
				}
				deliverArgs(msg.Args, msg.WorkingDir)
			})
		}()
	}
}

func releaseSingleInstanceLock() {
	singleInstance.Lock()
	defer singleInstance.Unlock()
	if singleInstance.listener != nil {
		singleInstance.listener.Close()
		if fi, err := os.Stat(singleInstance.path); err == nil && singleInstance.socket != nil && os.SameFile(fi, singleInstance.socket) {
			_ = os.Remove(singleInstance.path)
		}
		singleInstance.listener, singleInstance.socket, singleInstance.locked = nil, nil, false
	}
}

// singleInstanceKey identifies the lock of the app named name. The
// identifier keeps the development app of mygo dev ("<identifier>.dev")
// apart from the installed app when both call SetName with the same name.
func singleInstanceKey(name string) string {
	if info, ok := packageInfo(); ok && info.Identifier != "" {
		return info.Identifier + "\n" + name
	}
	return name
}

// singleInstanceSocket returns a short per-user socket path; Unix socket
// paths are limited to about 100 bytes.
func singleInstanceSocket(name string) string {
	sum := sha256.Sum256([]byte(name))
	file := "mygo-" + strconv.Itoa(os.Getuid()) + "-" + hex.EncodeToString(sum[:6]) + ".sock"
	for _, dir := range []string{os.Getenv("XDG_RUNTIME_DIR"), os.TempDir(), "/tmp"} {
		if dir != "" && len(filepath.Join(dir, file)) < 100 {
			return filepath.Join(dir, file)
		}
	}
	return filepath.Join("/tmp", file)
}
