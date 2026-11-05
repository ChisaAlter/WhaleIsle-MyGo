package session

import (
	"context"
	"fmt"
	"os"
)

// AttachmentRead captures the canonical Host and Session before a worker is
// started. Its key is also the cache namespace; an opaque attachment ID alone
// is never an owner.
type AttachmentRead struct {
	Key   string
	Fetch func(context.Context, string) (string, error)
}

func (r AttachmentRead) Identity(id string) string {
	if r.Key == "" {
		return id
	} // explicitly supplied legacy standalone loader
	return r.Key + "\x00" + id
}
func (r AttachmentRead) Cached(id string) string {
	if r.Key == "" && r.Fetch == nil {
		return ""
	}
	return CachedAttachmentPath(r.Identity(id))
}
func (r AttachmentRead) Forget(id string) { ForgetCachedAttachment(r.Identity(id)) }

// CaptureAttachmentRead rejects a stale rendered root snapshot, rather than
// resolving it through whichever Session happens to be selected later.
func (s *Store) CaptureAttachmentRead(binding, sessionID string) AttachmentRead {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.client == nil || sessionID == "" || s.selectedID != sessionID || s.planHistoryBinding != binding {
		return AttachmentRead{}
	}
	client, endpoint, generation := s.client, s.client.Endpoint, s.followGeneration
	captured := *client
	key := fmt.Sprintf("%s\x00%d", turnFileHistoryBinding(client, sessionID), generation)
	current := func() bool {
		s.mu.Lock()
		defer s.mu.Unlock()
		return s.client == client && client.Endpoint == endpoint && s.selectedID == sessionID && s.followGeneration == generation && s.planHistoryBinding == binding
	}
	return AttachmentRead{Key: key, Fetch: func(ctx context.Context, id string) (string, error) {
		if !current() {
			return "", context.Canceled
		}
		if path := CachedAttachmentPath(key + "\x00" + id); path != "" {
			if _, err := os.Stat(path); err == nil {
				return path, nil
			}
		}
		path, err := fetchSessionAttachment(ctx, &captured, sessionID, id, key+"\x00"+id)
		if !current() {
			return "", context.Canceled
		}
		return path, err
	}}
}

func (s *Store) CaptureAgentAttachmentRead(token uint64, address AgentSessionAddress) AttachmentRead {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.agentResourceCurrentLocked(token, address) || s.agentNavigation.client == nil {
		return AttachmentRead{}
	}
	client, endpoint := s.agentNavigation.client, s.agentNavigation.client.Endpoint
	captured := *client
	key := fmt.Sprintf("%s\x00%v\x00%d", turnFileHistoryBinding(client, address.ChildSessionID), address, token)
	current := func() bool {
		s.mu.Lock()
		defer s.mu.Unlock()
		return s.agentResourceCurrentLocked(token, address) && s.agentNavigation.client == client && client.Endpoint == endpoint
	}
	return AttachmentRead{Key: key, Fetch: func(ctx context.Context, id string) (string, error) {
		if !current() {
			return "", context.Canceled
		}
		if path := CachedAttachmentPath(key + "\x00" + id); path != "" {
			if _, err := os.Stat(path); err == nil {
				return path, nil
			}
		}
		path, err := fetchSessionAttachment(ctx, &captured, address.ChildSessionID, id, key+"\x00"+id)
		if !current() {
			return "", context.Canceled
		}
		return path, err
	}}
}
