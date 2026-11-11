package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"whaleisle/internal/desktop/phoneremote"
	"whaleisle/internal/harness"
	"whaleisle/internal/processutil"
	"whaleisle/internal/store/session"
)

func exercisePhone(ctx context.Context, stage, root, userData string, endpoint harness.Endpoint, store *session.Store, sessionID string, pass func(string, string)) error {
	tools, chrome := os.Getenv("WHALEISLE_PHONE_TOOLS"), os.Getenv("WHALEISLE_PHONE_CHROME")
	if tools == "" || chrome == "" {
		return errors.New("phone check requires explicit tooling and headless Chrome paths")
	}
	_, source, _, _ := runtime.Caller(0)
	cmd := exec.CommandContext(ctx, filepath.Join(stage, "node/node.exe"), filepath.Join(filepath.Dir(source), "phone-relay-browser.cjs"), root, filepath.Join(stage, "vendor/dshd-remote"), tools, chrome)
	processutil.Background(cmd)
	in, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	errLog, err := os.Create(filepath.Join(root, "phone-helper-stderr.log"))
	if err != nil {
		return err
	}
	defer errLog.Close()
	cmd.Stderr = errLog
	if err = cmd.Start(); err != nil {
		return err
	}
	defer func() { in.Close(); _ = cmd.Wait() }()
	events := make(chan map[string]json.RawMessage, 16)
	go func() {
		defer close(events)
		scanner := bufio.NewScanner(out)
		scanner.Buffer(make([]byte, 4096), 1<<20)
		for scanner.Scan() {
			var v map[string]json.RawMessage
			if json.Unmarshal(scanner.Bytes(), &v) == nil {
				events <- v
			}
		}
	}()
	wait := func(kind string) (map[string]json.RawMessage, error) {
		for {
			select {
			case v, ok := <-events:
				if !ok {
					return nil, errors.New("phone helper closed before " + kind)
				}
				var got string
				_ = json.Unmarshal(v["kind"], &got)
				if got == "error" {
					var message string
					_ = json.Unmarshal(v["message"], &message)
					return nil, errors.New(message)
				}
				if got == kind {
					return v, nil
				}
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}
	ready, err := wait("ready")
	if err != nil {
		return err
	}
	var relay string
	_ = json.Unmarshal(ready["endpoint"], &relay)
	var pageOrigin string
	_ = json.Unmarshal(ready["pageOrigin"], &pageOrigin)
	if host, _, err := net.SplitHostPort(relay); err != nil || host != "127.0.0.1" {
		return errors.New("relay must be owned loopback")
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	remote := phoneremote.NewRuntime()
	defer remote.Stop()
	remote.Start(userData, filepath.Join(stage, "vendor"), map[string]any{"remoteEnabled": true, "remoteMode": "away", "remoteBindAddress": "127.0.0.1", "remotePort": port, "remoteRelayEndpoint": relay, "remoteRelayUseTls": false, "remoteAppBaseUrl": pageOrigin, "harnessOrigin": endpoint.Origin, "harnessCookie": endpoint.Cookie})
	if err = await(ctx, func() (bool, error) {
		s := remote.Snapshot()
		if s.Status == "offline" || s.Status == "missing" {
			return false, fmt.Errorf("remote bridge %s: %s", s.Status, s.Message)
		}
		return s.Status == "ready" && s.RelayConnected && s.PairingURL != "", nil
	}); err != nil {
		return err
	}
	pass("phone-real-daemon-relay-ready", "native Go bridge started the staged remote daemon and connected to an owned local relay")
	request, _ := json.Marshal(map[string]any{"action": "browse", "offer": remote.Snapshot().PairingURL, "sessionId": sessionID})
	if _, err = in.Write(append(request, '\n')); err != nil {
		return err
	}
	if _, err = wait("browser-complete"); err != nil {
		return err
	}
	if err = await(ctx, func() (bool, error) {
		for _, message := range store.Snapshot().Messages {
			if message.Role == "assistant" && strings.Contains(message.Body, "QA_NEXT_OK") && !message.Streaming {
				return true, nil
			}
		}
		return false, nil
	}); err != nil {
		return err
	}
	pass("phone-pair-send-reconnect", "real browser paired, restored Host history, sent through relay/daemon/Host to the local model, reconnected and restored history plus draft")
	remote.Stop()
	if !closedListener(fmt.Sprintf("http://127.0.0.1:%d", port)) {
		return errors.New("owned phone listeners survived stop")
	}
	if _, err = in.Write([]byte("{\"action\":\"stop\"}\n")); err != nil {
		return err
	}
	if _, err = wait("closed"); err != nil {
		return err
	}
	pass("phone-owned-resources-released", "native bridge and phone listeners stopped; headless browser and local relay closed")
	return nil
}
