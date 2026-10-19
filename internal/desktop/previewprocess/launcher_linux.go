//go:build linux && (amd64 || arm64)

package previewprocess

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

var namespaceKinds = []string{"user", "mnt", "pid", "net", "ipc", "uts"}

func namespaceIdentity() (string, error) {
	ids := make([]string, 0, len(namespaceKinds))
	for _, kind := range namespaceKinds {
		id, err := os.Readlink("/proc/self/ns/" + kind)
		if err != nil {
			return "", err
		}
		ids = append(ids, id)
	}
	return strings.Join(ids, ","), nil
}

func isolatedNamespace(parent, current string) bool {
	before, after := strings.Split(parent, ","), strings.Split(current, ",")
	if len(before) != len(namespaceKinds) || len(after) != len(before) {
		return false
	}
	for i, kind := range namespaceKinds {
		// Reject malformed attestations as well as an unchanged namespace.
		prefix := kind + ":["
		if !strings.HasPrefix(before[i], prefix) || !strings.HasSuffix(before[i], "]") || !strings.HasPrefix(after[i], prefix) || !strings.HasSuffix(after[i], "]") || before[i] == after[i] {
			return false
		}
	}
	return true
}

func linuxSandboxArgs(helper, root, nonce, parent string) []string {
	return []string{"--die-with-parent", "--new-session", "--unshare-all", "--cap-drop", "ALL", "--clearenv",
		"--ro-bind", root, "/", "--ro-bind", helper, "/whaleisle-preview",
		"--proc", "/proc", "--dev", "/dev", "--tmpfs", "/tmp", "--tmpfs", "/run", "--tmpfs", "/home",
		"--dir", "/home/preview", "--dir", "/run/preview", "--chdir", "/home/preview",
		"--setenv", "HOME", "/home/preview", "--setenv", "XDG_RUNTIME_DIR", "/run/preview",
		"--setenv", "PATH", "/usr/bin:/bin", "--setenv", "LANG", "C.UTF-8",
		"--setenv", "GDK_BACKEND", "x11", "--setenv", "GDK_SCALE", "1", "--setenv", "GDK_DPI_SCALE", "1",
		"--setenv", "LIBGL_ALWAYS_SOFTWARE", "1", "--setenv", "NO_AT_BRIDGE", "1",
		"--", "/usr/bin/dbus-run-session", "--", "/whaleisle-preview", "--namespace-capture-v1", nonce, parent}
}

func openPlatform(ctx context.Context, opt Options, doc Document) (*Session, error) {
	if doc.capture && doc.Inline {
		return nil, errors.New("inline capture is not supported")
	}
	helper, err := filepath.Abs(opt.HelperPath)
	if err != nil {
		return nil, err
	}
	root := helper + ".runtime"
	required := []string{helper, filepath.Join(root, "usr/bin/Xvfb"), filepath.Join(root, "usr/bin/dbus-run-session")}
	if !doc.capture {
		required = append(required, filepath.Join(root, "usr/bin/xdotool"))
	}
	for _, path := range required {
		st, err := os.Stat(path)
		if err != nil || !st.Mode().IsRegular() || st.Mode()&0111 == 0 {
			return nil, fmt.Errorf("missing executable in trusted isolated preview package: %s", path)
		}
	}
	// Fixed system binary; do not trust PATH/environment overrides for containment.
	const bwrap = "/usr/bin/bwrap"
	if _, err := os.Stat(bwrap); err != nil {
		return nil, errors.New("Linux isolated capture requires system bubblewrap at /usr/bin/bwrap")
	}
	parent, err := namespaceIdentity()
	if err != nil {
		return nil, err
	}
	var random [32]byte
	if _, err := rand.Read(random[:]); err != nil {
		return nil, err
	}
	nonce := hex.EncodeToString(random[:])
	lifetime, cancel := context.WithTimeout(ctx, 30*time.Minute)
	cmd := exec.CommandContext(lifetime, bwrap, linuxSandboxArgs(helper, root, nonce, parent)...)
	cmd.Env = []string{"PATH=/usr/bin:/bin", "LANG=C.UTF-8"}
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGKILL}
	input, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	output, err := cmd.StdoutPipe()
	if err != nil {
		input.Close()
		cancel()
		return nil, err
	}
	// Diagnostic output must never enter the framed protocol.
	cmd.Stderr = os.Stderr
	if err = cmd.Start(); err != nil {
		input.Close()
		output.Close()
		cancel()
		return nil, err
	}
	exited := make(chan struct{})
	go func() { _ = cmd.Wait(); close(exited) }()
	session := &Session{done: make(chan struct{})}
	session.closeFn = func() error {
		_ = input.Close()
		cancel()
		_ = cmd.Process.Kill()
		<-exited
		_ = output.Close()
		return nil
	}
	var firstFrame []byte
	ready := make(chan error, 1)
	go func() {
		p, err := readPacket(output)
		if err == nil {
			err = expect(p, "ready", nonce)
		}
		if err == nil && p.SID != parent {
			err = errors.New("namespace helper attestation mismatch")
		}
		if err == nil {
			err = sendDocument(input, nonce, doc)
		}
		if err == nil {
			p, err = readPacket(output)
			if err == nil {
				err = expect(p, "loaded", nonce)
			}
		}
		if err == nil && !doc.capture {
			err = session.acceptImageDimensions(doc, p)
		}
		if err == nil && !doc.capture {
			firstFrame, err = receiveFrame(output, nonce, doc.Width)
		}
		if err == nil && doc.capture {
			result, captureErr := receiveCapture(output, nonce, doc.Width)
			err = captureErr
			if err == nil {
				session.capture = &result
			}
		}
		ready <- err
	}()
	timer := time.NewTimer(25 * time.Second)
	defer timer.Stop()
	select {
	case err := <-ready:
		if err != nil {
			session.Close()
			return nil, err
		}
		if !doc.capture {
			if err := startPrivateViewer(lifetime, session, input, output, nonce, doc, firstFrame); err != nil {
				session.Close()
				return nil, err
			}
		}
		go func() {
			select {
			case <-exited:
			case <-lifetime.Done():
			}
			_ = session.Close()
		}()
		return session, nil
	case <-lifetime.Done():
		session.Close()
		return nil, lifetime.Err()
	case <-timer.C:
		session.Close()
		return nil, errors.New("isolated Linux capture startup timed out")
	}
}
