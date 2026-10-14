// Package taskcontrol speaks the bundled dsh-task-control Host contract.
// It never invokes the cancel endpoint or interprets missing coverage as idle.
package taskcontrol

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var ErrCancelled = errors.New("task protection cancelled")

type Work struct {
	Kind  string `json:"kind"`
	ID    string `json:"id"`
	Count int    `json:"count"`
}
type Inspection struct {
	OK             bool              `json:"ok"`
	HostGeneration string            `json:"hostGeneration"`
	ActiveWork     []Work            `json:"activeWork"`
	ScheduledWork  []Work            `json:"scheduledWork"`
	Coverage       map[string]string `json:"coverage"`
	Unknown        string            `json:"-"`
	LocalRunning   int               `json:"-"`
}

func (i Inspection) NeedsConsent() bool {
	if i.Unknown != "" || !i.OK || i.LocalRunning > 0 || len(i.ActiveWork) > 0 || len(i.ScheduledWork) > 0 {
		return true
	}
	for _, key := range []string{"agents", "jobs", "schedule", "bots"} {
		v := i.Coverage[key]
		if v != "ok" && v != "intentional-disabled" {
			return true
		}
	}
	for _, v := range i.Coverage {
		if v != "ok" && v != "intentional-disabled" {
			return true
		}
	}
	return false
}

type Client struct {
	origin, token string
	http          *http.Client
	LocalRunning  func() int
}

func New(origin, token string) *Client {
	u, err := url.Parse(origin)
	clean := ""
	if err == nil && u.User == nil && (u.Scheme == "http" || u.Scheme == "https") {
		ip := net.ParseIP(u.Hostname())
		if u.Hostname() == "localhost" || ip != nil && ip.IsLoopback() {
			clean = u.Scheme + "://" + u.Host
		}
	}
	return &Client{origin: clean, token: token, http: &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}
func (c *Client) call(ctx context.Context, op string, body any, out any, timeout time.Duration) error {
	if c.origin == "" || c.token == "" {
		return errors.New("Host task-control endpoint or token unavailable")
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.origin+"/dshd-task-control/"+op, bytes.NewReader(payload))
	if err != nil {
		return errors.New("invalid task-control request")
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("Host task-control %s unavailable", op)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("Host task-control %s HTTP %d", op, res.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("task-control %s response unreadable", op)
	}
	var verdict struct {
		OK   bool   `json:"ok"`
		Code string `json:"code"`
	}
	if json.Unmarshal(raw, &verdict) != nil {
		return fmt.Errorf("invalid task-control %s response", op)
	}
	if !verdict.OK {
		return fmt.Errorf("task-control %s: %s", op, strings.ReplaceAll(verdict.Code, c.token, "[redacted]"))
	}
	if err = json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("invalid task-control %s response", op)
	}
	return nil
}
func (c *Client) inspect(ctx context.Context) (Inspection, error) {
	var i Inspection
	err := c.call(ctx, "inspect", map[string]any{}, &i, 5*time.Second)
	if err == nil && (i.HostGeneration == "" || i.ActiveWork == nil || i.ScheduledWork == nil || i.Coverage == nil) {
		err = errors.New("Host task inspection is incomplete")
	}
	if c.LocalRunning != nil {
		i.LocalRunning = c.LocalRunning()
	}
	return i, err
}

// Coordinate performs inspect -> acquire (including drain) -> inspect -> renew
// before commit. Only a failed initial inspection can use the explicitly unknown
// fallback confirmation. Lock/drain/second-inspect failures always block commit.
func (c *Client) Coordinate(ctx context.Context, operation string, confirm func(Inspection) (bool, error), commit func() error) (proceeded bool, err error) {
	first, inspectErr := c.inspect(ctx)
	if inspectErr != nil {
		first.OK = false
		first.Unknown = inspectErr.Error()
		approved, e := confirm(first)
		if e != nil {
			return false, e
		}
		if !approved {
			return false, ErrCancelled
		}
		if e = commit(); e != nil {
			return false, e
		}
		return true, nil
	}
	if first.NeedsConsent() {
		approved, e := confirm(first)
		if e != nil {
			return false, e
		}
		if !approved {
			return false, ErrCancelled
		}
	}
	owner := "mygo-" + operation
	var lock struct {
		LockID         string `json:"lockId"`
		HostGeneration string `json:"hostGeneration"`
		ExpiresAt      int64  `json:"expiresAt"`
	}
	if err = c.call(ctx, "acquire", map[string]any{"owner": owner, "kind": operation, "ttlMs": 600000, "drainTimeoutMs": 30000}, &lock, 40*time.Second); err != nil {
		return false, err
	}
	if lock.LockID == "" {
		return false, errors.New("Host returned no task admission lock")
	}
	identity := map[string]any{"owner": owner, "lockId": lock.LockID}
	defer func() {
		if proceeded {
			return
		} // Successful quit/restart ends the old Host; never release against its replacement.
		var released map[string]any
		if e := c.call(context.Background(), "release", identity, &released, 5*time.Second); e != nil {
			err = errors.Join(err, fmt.Errorf("admission lock release failed (expires automatically): %w", e))
		}
	}()
	if lock.HostGeneration != first.HostGeneration {
		return false, errors.New("Host changed during task protection")
	}
	second, e := c.inspect(ctx)
	if e != nil {
		return false, e
	}
	if second.HostGeneration != lock.HostGeneration {
		return false, errors.New("Host changed while checking tasks")
	}
	// Show the post-drain picture whenever work/coverage changed. Existing work
	// already approved in the first picture does not produce a duplicate prompt.
	firstJSON, _ := json.Marshal(first)
	secondJSON, _ := json.Marshal(second)
	if second.NeedsConsent() && (!bytes.Equal(firstJSON, secondJSON) || second.LocalRunning != first.LocalRunning) {
		approved, e := confirm(second)
		if e != nil {
			return false, e
		}
		if !approved {
			return false, ErrCancelled
		}
	}
	// renew does not itself reject an expired-but-unpruned lock in the Host
	// contract, so check its original deadline before renewing it.
	if lock.ExpiresAt <= time.Now().UnixMilli() {
		return false, errors.New("task admission lock expired; retry the action")
	}
	var renewed map[string]any
	if e = c.call(ctx, "renew", map[string]any{"owner": owner, "lockId": lock.LockID, "ttlMs": 600000}, &renewed, 5*time.Second); e != nil {
		return false, e
	}
	if e = commit(); e != nil {
		return false, e
	}
	return true, nil
}
