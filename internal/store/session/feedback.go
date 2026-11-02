package session

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	MessageFeedbackUp   = "positive"
	MessageFeedbackDown = "negative"
)

type FeedbackItem struct {
	MessageID string `json:"messageId"`
	Rating    string `json:"rating"`
	Note      string `json:"note,omitempty"`
	Category  string `json:"category,omitempty"`
	Version   string `json:"version"`
}
type FeedbackView struct {
	Ready, Loading                   bool
	BusyID, Error                    string
	Items                            map[string]FeedbackItem
	EditorID, Rating, Note, Category string
	// Acknowledgement counts committed actions, independently of draft dismissal.
	Acknowledgement uint64
}
type feedbackState struct {
	FeedbackView
	epoch              uint64
	draftGeneration    uint64
	readGeneration     uint64
	readHostGeneration uint64
}

func (s *Store) Feedback() FeedbackView {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := s.feedback.FeedbackView
	v.Items = map[string]FeedbackItem{}
	for id, item := range s.feedback.Items {
		v.Items[id] = item
	}
	return v
}
func (s *Store) clearFeedbackLocked() {
	s.invalidateViewLocked()
	s.feedback = feedbackState{epoch: s.feedback.epoch + 1}
	s.messageRatings = nil
	s.feedbackBusyID = ""
}
func (s *Store) BeginFeedback(id, rating string) {
	s.mu.Lock()
	if !s.feedback.Ready || s.feedback.Loading || id == "" || (rating != MessageFeedbackUp && rating != MessageFeedbackDown) {
		s.mu.Unlock()
		return
	}
	f := &s.feedback
	// Original dialog.ts open() replaces the draft independently of any
	// in-flight action; that action still owns BusyID until it settles.
	f.draftGeneration++
	f.Note = ""
	f.Category = ""
	f.EditorID = id
	f.Rating = rating
	f.Error = ""
	s.mu.Unlock()
	s.fire()
}
func (s *Store) EditFeedback(note, category string) {
	s.mu.Lock()
	if s.feedback.BusyID == "" {
		s.feedback.Note = note
		s.feedback.Category = category
	}
	s.mu.Unlock()
}
func (s *Store) CancelFeedback() {
	s.mu.Lock()
	s.feedback.draftGeneration++
	s.feedback.EditorID = ""
	s.feedback.Rating = ""
	s.feedback.Note = ""
	s.feedback.Category = ""
	s.feedback.Error = ""
	s.mu.Unlock()
	s.fire()
}
func (s *Store) SubmitFeedback(ctx context.Context) {
	s.mu.Lock()
	id, rating, note := s.feedback.EditorID, s.feedback.Rating, s.feedback.Note
	s.mu.Unlock()
	s.RateMessage(ctx, id, rating, note)
}

type feedbackReply struct {
	OK    *bool           `json:"ok"`
	Value json.RawMessage `json:"value"`
	Error struct {
		Code    string        `json:"code"`
		Current *FeedbackItem `json:"current"`
	} `json:"error"`
}

func decodeFeedback(raw json.RawMessage) (feedbackReply, error) {
	var r feedbackReply
	if json.Unmarshal(raw, &r) != nil || r.OK == nil {
		return r, fmt.Errorf("feedback: incompatible Host response")
	}
	if !*r.OK {
		return r, fmt.Errorf("feedback: %s", r.Error.Code)
	}
	return r, nil
}
func (s *Store) RefreshFeedback(ctx context.Context) {
	s.refreshFeedbackForOwner(ctx, nil)
}

func (s *Store) refreshFeedbackForOwner(ctx context.Context, owner *workspaceSelectedOwner) {
	s.mu.Lock()
	if ctx.Err() != nil || (owner != nil && !owner.currentLocked(s)) {
		s.mu.Unlock()
		return
	}
	f := &s.feedback
	if s.client == nil || s.selectedID == "" || (f.Loading && f.readHostGeneration == s.eventsGeneration) || f.BusyID != "" {
		s.mu.Unlock()
		return
	}
	client, id, epoch := s.client, s.selectedID, f.epoch
	f.readGeneration++
	request, hostGeneration := f.readGeneration, s.eventsGeneration
	f.readHostGeneration = hostGeneration
	f.Loading = true
	f.Ready = false
	f.Error = ""
	s.mu.Unlock()
	s.fire()
	// A synchronous notification can itself navigate or replace the Host.
	// Recheck before launching an RPC for a revoked owner.
	if owner != nil {
		s.mu.Lock()
		valid := ctx.Err() == nil && owner.currentLocked(s) && s.feedback.epoch == epoch && s.feedback.readGeneration == request
		retired := !valid && s.feedback.epoch == epoch && s.feedback.readGeneration == request && s.feedback.Loading
		if retired {
			s.feedback.Loading = false
			s.invalidateViewLocked()
		}
		s.mu.Unlock()
		if !valid {
			if retired {
				s.fire()
			}
			return
		}
	}
	go func() {
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		var raw json.RawMessage
		err := client.Call(ctx, "messageFeedback/list", map[string]any{"request": map[string]any{"sessionId": id}}, &raw)
		var value struct {
			Items []FeedbackItem `json:"items"`
		}
		if err == nil {
			var r feedbackReply
			r, err = decodeFeedback(raw)
			if err == nil {
				err = json.Unmarshal(r.Value, &value)
				if err == nil && value.Items == nil {
					err = fmt.Errorf("feedback: missing items")
				}
			}
		}
		s.mu.Lock()
		if s.client != client || s.selectedID != id || s.feedback.epoch != epoch || s.feedback.readGeneration != request {
			s.mu.Unlock()
			return
		}
		if ctx.Err() != nil || s.eventsGeneration != hostGeneration || (owner != nil && !owner.currentLocked(s)) {
			// This request still owns Loading. Retire only that flag so a new
			// Host generation can retry without accepting the stale response.
			s.feedback.Loading = false
			s.mu.Unlock()
			s.fire()
			return
		}
		f := &s.feedback
		f.Loading = false
		if err != nil {
			f.Error = "Feedback unavailable: " + err.Error()
		} else {
			f.Ready = true
			f.Items = map[string]FeedbackItem{}
			s.messageRatings = map[string]string{}
			for _, item := range value.Items {
				if item.MessageID != "" && item.Version != "" {
					f.Items[item.MessageID] = item
					s.messageRatings[item.MessageID] = item.Rating
				}
			}
		}
		s.mu.Unlock()
		s.fire()
	}()
}

// RateMessage sends only the actual per-message command. It never records a
// second session-level remark or fabricates offline success.
func (s *Store) RateMessage(ctx context.Context, id, rating, note string) {
	s.mutateFeedback(ctx, id, rating, note, false)
}
func (s *Store) ClearMessageFeedback(ctx context.Context, id string) {
	s.mutateFeedback(ctx, id, "", "", true)
}
func (s *Store) mutateFeedback(ctx context.Context, id, rating, note string, remove bool) {
	s.mu.Lock()
	f := &s.feedback
	if s.client == nil || s.selectedID == "" || !f.Ready || f.Loading || f.BusyID != "" || id == "" {
		s.mu.Unlock()
		return
	}
	if !remove && rating != MessageFeedbackUp && rating != MessageFeedbackDown {
		s.mu.Unlock()
		return
	}
	client, sid, epoch, draftGeneration := s.client, s.selectedID, f.epoch, f.draftGeneration
	item, exists := f.Items[id]
	if remove && !exists {
		s.mu.Unlock()
		return
	}
	var version any
	if exists {
		version = item.Version
	}
	req := map[string]any{"sessionId": sid, "messageId": id, "ifVersion": version}
	method := "messageFeedback/put"
	if remove {
		method = "messageFeedback/delete"
	} else {
		req["rating"] = rating
		if strings.TrimSpace(note) != "" {
			req["note"] = note
		}
		if f.EditorID == id {
			if f.Category != "" {
				req["category"] = f.Category
			}
		} else if exists && item.Category != "" {
			req["category"] = item.Category
		}
	}
	f.BusyID = id
	s.feedbackBusyID = id
	f.Error = ""
	s.mu.Unlock()
	s.fire()
	go func() {
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		var raw json.RawMessage
		err := client.Call(ctx, method, map[string]any{"request": req}, &raw)
		var r feedbackReply
		var saved FeedbackItem
		if err == nil {
			r, err = decodeFeedback(raw)
		}
		if err == nil {
			if remove {
				var v struct {
					Absent bool `json:"absent"`
				}
				if json.Unmarshal(r.Value, &v) != nil || !v.Absent {
					err = fmt.Errorf("feedback: deletion not acknowledged")
				}
			} else {
				err = json.Unmarshal(r.Value, &saved)
				if err == nil && (saved.MessageID != id || saved.Version == "" || saved.Rating != rating) {
					err = fmt.Errorf("feedback: returned item does not match")
				}
			}
		}
		s.mu.Lock()
		if s.client != client || s.selectedID != sid || s.feedback.epoch != epoch {
			s.mu.Unlock()
			return
		}
		f := &s.feedback
		currentDraft := f.draftGeneration == draftGeneration
		f.BusyID = ""
		s.feedbackBusyID = ""
		if err != nil {
			if currentDraft {
				f.Error = err.Error()
				s.lastAction = "feedback: " + err.Error()
			}
			if r.Error.Code == "version-conflict" {
				if r.Error.Current == nil {
					delete(f.Items, id)
					delete(s.messageRatings, id)
				} else if r.Error.Current.MessageID == id && r.Error.Current.Version != "" {
					f.Items[id] = *r.Error.Current
					s.messageRatings[id] = r.Error.Current.Rating
				}
			}
		} else {
			f.Acknowledgement++
			if f.Items == nil {
				f.Items = map[string]FeedbackItem{}
			}
			if s.messageRatings == nil {
				s.messageRatings = map[string]string{}
			}
			if remove {
				delete(f.Items, id)
				delete(s.messageRatings, id)
			} else {
				f.Items[id] = saved
				s.messageRatings[id] = saved.Rating
			}
			if currentDraft && f.EditorID == id {
				f.EditorID = ""
				f.Rating = ""
				f.Note = ""
				f.Category = ""
			}
			s.lastAction = "feedback saved"
		}
		s.mu.Unlock()
		s.fire()
	}()
}
