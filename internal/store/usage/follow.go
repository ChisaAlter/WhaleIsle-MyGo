package usage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/ui/i18n"
)

// AttachMux binds the shared remote.mux used for job/follow and account/watch.
func (s *Store) AttachMux(mux *dsh.Mux) {
	s.mu.Lock()
	if s.mux != mux {
		if s.jobFollowCancel != nil {
			s.jobFollowCancel()
			s.jobFollowCancel = nil
		}
		s.jobFollowEpoch++
	}
	s.mux = mux
	s.mu.Unlock()
	s.WatchAccount(context.Background())
	s.WatchExpiry(context.Background())
}

// WatchExpiry opens account/watchExpiry and stores the latest expiry token string.
func (s *Store) WatchExpiry(ctx context.Context) {
	s.mu.Lock()
	mux := s.mux
	if s.expiryWatchCancel != nil {
		s.expiryWatchCancel()
		s.expiryWatchCancel = nil
	}
	if mux == nil {
		s.mu.Unlock()
		return
	}
	fctx, cancel := context.WithCancel(ctx)
	s.expiryWatchCancel = cancel
	s.mu.Unlock()
	go s.runExpiryWatch(fctx, mux)
}

func (s *Store) runExpiryWatch(ctx context.Context, mux *dsh.Mux) {
	if err := mux.Dial(ctx); err != nil {
		return
	}
	stream, err := mux.Open(ctx, "account/watchExpiry", map[string]any{"args": map[string]any{}})
	if err != nil {
		return
	}
	defer stream.Cancel()

	for {
		raw, err := stream.Recv(ctx)
		if err != nil {
			if errors.Is(err, dsh.ErrStreamEnd) || errors.Is(err, context.Canceled) ||
				strings.Contains(err.Error(), "context canceled") {
				return
			}
			return
		}
		line := strings.TrimSpace(string(raw))
		if strings.HasPrefix(line, "\"") {
			var s string
			if json.Unmarshal(raw, &s) == nil {
				line = s
			}
		}
		if line == "" {
			continue
		}
		s.mu.Lock()
		if ctx.Err() != nil || s.mux != mux {
			s.mu.Unlock()
			return
		}
		s.expiryNote = truncateJSON(line, 120)
		s.mu.Unlock()
		s.fire()
	}
}

// WatchAccount opens account/watch and applies AccountView frames to the snapshot.
func (s *Store) WatchAccount(ctx context.Context) {
	s.mu.Lock()
	mux := s.mux
	if s.accountWatchCancel != nil {
		s.accountWatchCancel()
		s.accountWatchCancel = nil
	}
	if mux == nil {
		s.mu.Unlock()
		return
	}
	fctx, cancel := context.WithCancel(ctx)
	s.accountWatchCancel = cancel
	s.mu.Unlock()
	go s.runAccountWatch(fctx, mux)
}

func (s *Store) runAccountWatch(ctx context.Context, mux *dsh.Mux) {
	if err := mux.Dial(ctx); err != nil {
		s.failAccountWatch(ctx, mux, err)
		return
	}
	stream, err := mux.Open(ctx, "account/watch", map[string]any{"args": map[string]any{}})
	if err != nil {
		s.failAccountWatch(ctx, mux, err)
		return
	}
	defer stream.Cancel()

	for {
		raw, err := stream.Recv(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) ||
				strings.Contains(err.Error(), "context canceled") {
				return
			}
			s.failAccountWatch(ctx, mux, err)
			return
		}
		var view api.AccountView
		if json.Unmarshal(raw, &view) != nil {
			continue
		}
		s.mu.Lock()
		if ctx.Err() != nil || s.mux != mux {
			s.mu.Unlock()
			return
		}
		if view.Status != "credential-stored" || s.accountStatus != view.Status || view.Attempt.Phase == "succeeded" && view.Attempt.Id != s.attemptID {
			s.resetBonusNoticesLocked()
			s.bonuses = nil
		}
		s.resetAccountDetailsLocked()
		client, epoch := s.client, s.accountEpoch
		s.loading = false
		s.err = nil
		s.accountStreamErr = nil
		s.email, s.name, s.plan = "", "", ""
		if view.Status != "" {
			s.accountStatus = view.Status
		}
		if view.Links.UsageUrl != "" {
			s.usageURL = view.Links.UsageUrl
		}
		if view.Links.TopUpUrl != "" {
			s.topUpURL = view.Links.TopUpUrl
		}
		if view.Attempt.Phase != "" {
			s.attemptPhase = view.Attempt.Phase
		}
		if view.Attempt.Id != "" {
			s.attemptID = view.Attempt.Id
		}
		s.attemptPhase = view.Attempt.Phase
		s.attemptID = view.Attempt.Id
		s.attemptErrorCode = ""
		if view.Attempt.ErrorCode != nil {
			s.attemptErrorCode = *view.Attempt.ErrorCode
		}
		s.authorizeURL = ""
		if view.Attempt.AuthorizeUrl != nil {
			s.authorizeURL = *view.Attempt.AuthorizeUrl
		}
		s.mu.Unlock()
		s.fire()
		if client != nil {
			s.refreshAccountDetails(ctx, client, epoch, view.Status)
			s.refreshAccountBonuses(ctx, client, epoch, view.Status)
		}
	}
}

func (s *Store) failAccountWatch(ctx context.Context, mux *dsh.Mux, err error) {
	if ctx.Err() != nil {
		return
	}
	s.mu.Lock()
	if s.mux != mux || ctx.Err() != nil {
		s.mu.Unlock()
		return
	}
	label := "Account connection lost. Refresh the account to reconnect."
	if i18n.Locale() == i18n.LocaleZH {
		label = "账户连接已中断，请刷新账户重新连接。"
	}
	s.accountStreamErr = fmt.Errorf("%s %w", label, err)
	s.mu.Unlock()
	s.fire()
}

// FollowJob opens job/follow for a few frames and appends a preview into Recent.
func (s *Store) FollowJob(ctx context.Context, sessionID, jobID string) {
	sessionID = strings.TrimSpace(sessionID)
	jobID = strings.TrimSpace(jobID)
	s.mu.Lock()
	client, mux, epoch := s.client, s.mux, s.statsEpoch
	if s.jobFollowCancel != nil {
		s.jobFollowCancel()
		s.jobFollowCancel = nil
	}
	s.jobFollowEpoch++
	followEpoch := s.jobFollowEpoch
	if mux == nil || sessionID == "" || jobID == "" {
		s.mu.Unlock()
		return
	}
	fctx, cancel := context.WithCancel(ctx)
	s.jobFollowCancel = cancel
	s.mu.Unlock()

	go s.runJobFollow(fctx, client, mux, epoch, followEpoch, sessionID, jobID)
}

func (s *Store) jobFollowCurrentLocked(ctx context.Context, client *dsh.Client, mux *dsh.Mux, epoch, followEpoch uint64) bool {
	return ctx.Err() == nil && s.client == client && s.mux == mux && s.statsEpoch == epoch && s.jobFollowEpoch == followEpoch
}

func (s *Store) runJobFollow(ctx context.Context, client *dsh.Client, mux *dsh.Mux, epoch, followEpoch uint64, sessionID, jobID string) {
	if err := mux.Dial(ctx); err != nil {
		if errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "context canceled") || ctx.Err() != nil {
			return
		}
		s.mu.Lock()
		if !s.jobFollowCurrentLocked(ctx, client, mux, epoch, followEpoch) {
			s.mu.Unlock()
			return
		}
		s.err = err
		s.mu.Unlock()
		s.fire()
		return
	}
	payload := map[string]any{
		"args": map[string]any{
			"request": map[string]any{
				"sessionId": sessionID,
				"jobId":     jobID,
			},
		},
	}
	stream, err := mux.Open(ctx, "job/follow", payload)
	if err != nil {
		if errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "context canceled") || ctx.Err() != nil {
			return
		}
		s.mu.Lock()
		if !s.jobFollowCurrentLocked(ctx, client, mux, epoch, followEpoch) {
			s.mu.Unlock()
			return
		}
		s.err = err
		s.mu.Unlock()
		s.fire()
		return
	}
	defer stream.Cancel()

	previews := make([]string, 0, 8)
	for len(previews) < 8 {
		raw, err := stream.Recv(ctx)
		if err != nil {
			if errors.Is(err, dsh.ErrStreamEnd) {
				break
			}
			if errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "context canceled") || ctx.Err() != nil {
				return
			}
			s.mu.Lock()
			if !s.jobFollowCurrentLocked(ctx, client, mux, epoch, followEpoch) {
				s.mu.Unlock()
				return
			}
			s.err = err
			s.mu.Unlock()
			s.fire()
			return
		}
		line := truncateJSON(string(raw), 160)
		if line != "" {
			previews = append(previews, line)
		}
	}

	s.mu.Lock()
	if !s.jobFollowCurrentLocked(ctx, client, mux, epoch, followEpoch) {
		s.mu.Unlock()
		return
	}
	if len(previews) > 0 {
		detail := strings.Join(previews, "\n")
		found := false
		for i := range s.recent {
			if s.recent[i].JobID == jobID {
				s.recent[i].Detail = detail
				found = true
				break
			}
		}
		if !found {
			s.recent = append(s.recent, Entry{
				Label:  "job/follow " + jobID,
				Detail: detail,
				JobID:  jobID,
			})
		}
		s.err = nil
	}
	s.mu.Unlock()
	s.fire()
}
