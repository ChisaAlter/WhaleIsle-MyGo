package proactive

import (
	"fmt"
)

// Notice is the durable in-app delivery. Desktop presentation is an additional
// receipt, never evidence that the user read or accepted the task result.
type Notice struct {
	ID            string `json:"id"`
	TaskID        string `json:"taskId,omitempty"`
	SourceID      string `json:"sourceId,omitempty"`
	SessionID     string `json:"sessionId,omitempty"`
	Kind          string `json:"kind"`
	Title         string `json:"title"`
	Body          string `json:"body"`
	CreatedAt     string `json:"createdAt"`
	ReadAt        string `json:"readAt,omitempty"`
	DesktopStatus string `json:"desktopStatus"`
	DesktopError  string `json:"desktopError,omitempty"`
}

func (s *Service) Notices() []Notice {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]Notice, 0, len(s.noticeOrder))
	for _, id := range s.noticeOrder {
		result = append(result, s.notices[id])
	}
	return result
}

func (s *Service) ClaimNotice(id string) (Notice, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n, ok := s.notices[id]
	if !ok {
		return Notice{}, ErrNotFound
	}
	if n.DesktopStatus != "pending" || n.ReadAt != "" {
		return Notice{}, ErrConflict
	}
	n.DesktopStatus = "sending"
	if err := s.commit(Event{Type: "notice.updated", Notice: &n}); err != nil {
		return Notice{}, err
	}
	return n, nil
}

func (s *Service) FinishNotice(id, status, message string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	n, ok := s.notices[id]
	if !ok {
		return ErrNotFound
	}
	if n.DesktopStatus != "sending" {
		return ErrConflict
	}
	switch status {
	case "shown", "in-app", "unavailable", "failed":
	default:
		return fmt.Errorf("unknown notice delivery status")
	}
	n.DesktopStatus, n.DesktopError = status, message
	return s.commit(Event{Type: "notice.updated", Notice: &n})
}

func (s *Service) ReadNotice(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	n, ok := s.notices[id]
	if !ok {
		return ErrNotFound
	}
	if n.ReadAt != "" {
		return nil
	}
	n.ReadAt = timestamp(s.now())
	if n.DesktopStatus == "pending" {
		n.DesktopStatus = "in-app"
	}
	return s.commit(Event{Type: "notice.updated", Notice: &n})
}

func (s *Service) RetryNotice(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	n, ok := s.notices[id]
	if !ok {
		return ErrNotFound
	}
	if n.DesktopStatus != "failed" && n.DesktopStatus != "unknown" && n.DesktopStatus != "unavailable" {
		return ErrConflict
	}
	n.DesktopStatus, n.DesktopError, n.ReadAt = "pending", "", ""
	return s.commit(Event{Type: "notice.updated", Notice: &n})
}

func (s *Service) recoverNotices() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var events []Event
	for _, id := range s.noticeOrder {
		n := s.notices[id]
		if n.DesktopStatus == "sending" {
			n.DesktopStatus = "unknown"
			events = append(events, Event{Type: "notice.updated", Notice: &n})
		}
	}
	return s.commit(events...)
}
