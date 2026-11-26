package mygo

import (
	"encoding/json/v2"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"
)

var singleInstance struct {
	sync.Mutex
	listener net.Listener
	path     string
	socket   os.FileInfo // identifies our socket file at path
	platform singleInstancePlatform
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
	key := singleInstanceKey(a.Name())
	path := singleInstanceSocket(key)
	wd, _ := os.Getwd()
	msg, _ := json.Marshal(secondInstanceMessage{Args: os.Args[1:], WorkingDir: wd})
	plat, locked := acquireSingleInstance(key, path, msg, a.Name())
	if !locked {
		return false
	}
	singleInstance.platform = plat
	singleInstance.locked = true
	a.OnQuit(releaseSingleInstanceLock)
	return true
}

func startSingleInstanceListener(ln net.Listener, path string) {
	if ul, ok := ln.(*net.UnixListener); ok {
		// The socket file is removed on release, and only if it is still ours:
		// an instance that found it stale may have replaced it.
		ul.SetUnlinkOnClose(false)
	}
	singleInstance.socket, _ = os.Stat(path)
	singleInstance.listener, singleInstance.path = ln, path
	go acceptSecondInstances(ln)
	// Every key leaves its file behind when its owner dies without releasing
	// the lock, and a key is per profile: an isolated or test profile gets a
	// fresh name each time. Nothing else ever names those files again, so they
	// are swept here, out of the launch's way.
	go sweepStaleSockets(filepath.Dir(path), path, staleSocketGrace)
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
		singleInstance.listener, singleInstance.socket = nil, nil
	}
	singleInstance.platform.release()
	singleInstance.locked = false
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

// socketFileName matches the files singleInstanceSocket creates, so the sweep
// never looks at anything else in the runtime directory.
// Windows has no user id and os.Getuid reports -1, so the id is negative there.
var socketFileName = regexp.MustCompile(`^mygo--?[0-9]+-[0-9a-f]{12}\.sock$`)

// staleSocketGrace keeps the sweep away from a socket another instance is
// binding right now: a file that young may be a launch in progress, and a
// socket left by a crash is older than any launch.
const staleSocketGrace = time.Minute

// staleSocketDialTimeout bounds the probe that decides whether a socket still
// has an owner. A file without a listener refuses at once; a live one accepts
// at once.
const staleSocketDialTimeout = 500 * time.Millisecond

// socketSweep reports one pass over the directory that holds the socket files.
// The launch log carries it, so one run answers whether the stale files a
// crashed profile left behind are gone, without anyone counting files by hand.
type socketSweep struct {
	// Examined counts files named like this package's sockets that belong to
	// another instance or key; this instance's own file is never a candidate.
	Examined int
	// Removed counts files no instance answered for.
	Removed int
	// InUse counts files a live instance accepted the probe on.
	InUse int
}

// sweepStaleSockets removes the socket files in dir that no instance answers
// for. A key's owner removes its own file on a clean quit; an unclean exit
// leaves it for good, and because a key covers one profile, a machine that
// runs isolated profiles collects one file per profile that ever crashed.
//
// A file that accepts a connection has an owner and stays. The probe writes
// nothing and closes at once, so a live owner reads an empty line, rejects it
// as a command line, and runs no handlers.
func sweepStaleSockets(dir, own string, grace time.Duration) socketSweep {
	var sweep socketSweep
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("mygo: single-instance sockets in %s: %v", dir, err)
		return sweep
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !socketFileName.MatchString(name) {
			continue
		}
		path := filepath.Join(dir, name)
		if path == own {
			continue
		}
		observed, err := entry.Info()
		if err != nil {
			continue
		}
		sweep.Examined++
		if time.Since(observed.ModTime()) < grace {
			continue
		}
		if conn, err := net.DialTimeout("unix", path, staleSocketDialTimeout); err == nil {
			conn.Close()
			sweep.InUse++
			continue
		}
		// Remove only the file that was probed: another instance may have
		// replaced it since the dial.
		current, err := os.Lstat(path)
		if err != nil || !os.SameFile(observed, current) {
			continue
		}
		if os.Remove(path) == nil {
			sweep.Removed++
		}
	}
	log.Printf("mygo: single-instance sockets in %s: examined %d, removed %d, in use %d", dir, sweep.Examined, sweep.Removed, sweep.InUse)
	return sweep
}
