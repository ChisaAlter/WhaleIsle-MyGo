package harness

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// EnsureWorkspace registers workspacePath with dsh via workspace/create.
func EnsureWorkspace(ctx context.Context, client *http.Client, ep Endpoint, workspacePath string) error {
	if workspacePath == "" {
		return fmt.Errorf("workspace path is empty")
	}
	if client == nil {
		client = http.DefaultClient
	}
	rpcID := newRPCID()
	body, err := json.Marshal(map[string]any{
		"type":   "client-request",
		"rpcId":  rpcID,
		"method": "workspace/create",
		"payload": map[string]any{
			"args": map[string]any{
				"request": map[string]any{"path": workspacePath},
			},
		},
	})
	if err != nil {
		return err
	}
	endpoint, err := url.JoinPath(ep.Origin, "/api/workspace/create")
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if ep.Cookie != "" {
		req.Header.Set("Cookie", ep.Cookie)
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	var parsed struct {
		Type   string `json:"type"`
		RPCID  string `json:"rpcId"`
		Result struct {
			OK    bool `json:"ok"`
			Error *struct {
				Message string `json:"message"`
			} `json:"error"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return fmt.Errorf("workspace/create: non-JSON HTTP %d: %s", res.StatusCode, truncate(string(raw), 240))
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("workspace/create: HTTP %d: %s", res.StatusCode, truncate(string(raw), 240))
	}
	if parsed.Type != "server-response" || parsed.RPCID != rpcID {
		return fmt.Errorf("workspace/create: invalid response envelope")
	}
	if !parsed.Result.OK {
		msg := "workspace/create failed"
		if parsed.Result.Error != nil && parsed.Result.Error.Message != "" {
			msg = parsed.Result.Error.Message
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func newRPCID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
