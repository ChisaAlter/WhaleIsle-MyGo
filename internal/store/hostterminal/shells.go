package hostterminal

import (
	"context"
	"errors"
	"time"
)

// ShellView contains only choices returned by the current Host for this Session.
// Selecting or remembering one never creates a terminal.
type ShellView struct {
	Loading, Loaded                      bool
	Items                                []Shell
	SelectedPath, Error, PreferenceError string
}
type shellState struct {
	view     ShellView
	explicit bool
	revision uint64
	cancel   context.CancelFunc
}
type shellLaunch struct {
	path, preferred string
	explicit        bool
}

// ConfigureShellPreference installs a client-local preference, once at startup.
// Persistence is best-effort; it never becomes a launch prerequisite.
func (s *Store) ConfigureShellPreference(path string, save func(string) error) {
	s.preferenceMu.Lock()
	defer s.preferenceMu.Unlock()
	s.mu.Lock()
	s.shellPreference, s.saveShellPreference = path, save
	s.mu.Unlock()
}
func cloneShells(items []Shell) []Shell {
	out := append([]Shell(nil), items...)
	for i := range out {
		out[i].Args = append([]string(nil), out[i].Args...)
	}
	return out
}
func (s *Store) ShellSnapshot(owner uint64, session string) ShellView {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.epoch != owner {
		return ShellView{}
	}
	if st := s.sessions[session]; st != nil {
		v := st.shells.view
		v.Items = cloneShells(v.Items)
		return v
	}
	return ShellView{}
}
func shellOffered(items []Shell, path string) bool {
	if path == "" {
		return false
	}
	for _, v := range items {
		if v.Path == path {
			return true
		}
	}
	return false
}
func availableShell(items []Shell, preferred string) string {
	if shellOffered(items, preferred) {
		return preferred
	}
	if len(items) > 0 {
		return items[0].Path
	}
	return ""
}
func (s *Store) cancelShellDiscoveryLocked(st *sessionState) {
	st.shells.revision++
	if st.shells.cancel != nil {
		st.shells.cancel()
		st.shells.cancel = nil
	}
	st.shells.view.Loading = false
}

// DiscoverShells is an explicit, cancellable metadata lookup. No create, retain,
// follow or environment call is sent by this operation.
func (s *Store) DiscoverShells(owner uint64, session string) {
	s.mu.Lock()
	if s.epoch != owner || s.remote == nil || !s.ready || session == "" || s.visible != session {
		s.mu.Unlock()
		return
	}
	st := s.sessionLocked(session)
	if st.busy || st.shells.view.Loading {
		s.mu.Unlock()
		return
	}
	s.cancelShellDiscoveryLocked(st)
	revision := st.shells.revision
	ctx, cancel := context.WithTimeout(s.ctx, 15*time.Second)
	st.shells.cancel = cancel
	st.shells.view.Loading, st.shells.view.Error = true, ""
	r, epoch := s.remote, s.epoch
	s.mu.Unlock()
	s.fire()
	go func() {
		defer cancel()
		var items []Shell
		err := r.Call(ctx, "terminal/shells", map[string]any{"agentId": session}, &items)
		s.mu.Lock()
		if s.epoch != epoch || st.shells.revision != revision {
			s.mu.Unlock()
			return
		}
		st.shells.cancel = nil
		st.shells.view.Loading = false
		if err != nil {
			st.shells.view.Error = err.Error()
		} else {
			// An explicit still-available selection survives refreshing its menu.
			selected := s.shellPreference
			if st.shells.explicit && shellOffered(items, st.shells.view.SelectedPath) {
				selected = st.shells.view.SelectedPath
			} else {
				st.shells.explicit = false
			}
			st.shells.view.Items = cloneShells(items)
			st.shells.view.SelectedPath = availableShell(items, selected)
			st.shells.view.Loaded = true
		}
		s.mu.Unlock()
		s.fire()
	}()
}

func (s *Store) SelectShell(owner uint64, session, path string) {
	// Serialize config writes so a slower older write cannot replace a newer pick.
	s.preferenceMu.Lock()
	s.mu.Lock()
	st := s.sessions[session]
	if s.epoch != owner || !s.ready || s.visible != session || st == nil || st.busy || st.shells.view.Loading || !st.shells.view.Loaded || !shellOffered(st.shells.view.Items, path) {
		s.mu.Unlock()
		s.preferenceMu.Unlock()
		return
	}
	st.shells.view.SelectedPath = path
	st.shells.explicit = true
	st.shells.view.Error = ""
	st.shells.view.PreferenceError = ""
	s.shellPreference = path
	save := s.saveShellPreference
	s.mu.Unlock()
	var err error
	if save != nil {
		err = save(path)
	}
	s.mu.Lock()
	if s.epoch == owner && err != nil {
		st.shells.view.PreferenceError = "Shell choice is usable for this run, but could not be saved. " + err.Error()
	}
	s.mu.Unlock()
	s.preferenceMu.Unlock()
	s.fire()
}
func (s *Store) shellLaunchLocked(st *sessionState) shellLaunch {
	s.cancelShellDiscoveryLocked(st)
	return shellLaunch{st.shells.view.SelectedPath, s.shellPreference, st.shells.explicit}
}

// resolveLaunchShell revalidates preferences against this Host at explicit New.
// An explicit unavailable pick fails instead of silently starting another shell.
func (s *Store) resolveLaunchShell(ctx context.Context, r Remote, epoch uint64, session string, st *sessionState, op uint64, launch shellLaunch) (string, error) {
	var items []Shell
	if err := r.Call(ctx, "terminal/shells", map[string]any{"agentId": session}, &items); err != nil {
		return "", err
	}
	path := availableShell(items, launch.preferred)
	if launch.explicit {
		if !shellOffered(items, launch.path) {
			return "", errors.New("Selected shell is no longer available; refresh shell choices and select again")
		}
		path = launch.path
	}
	s.preferenceMu.Lock()
	s.mu.Lock()
	if s.epoch != epoch || st.operation != op || !s.ready || s.visible != session || ctx.Err() != nil {
		s.mu.Unlock()
		s.preferenceMu.Unlock()
		return "", context.Canceled
	}
	st.shells.view.Items = cloneShells(items)
	st.shells.view.Loaded = true
	st.shells.view.SelectedPath = path
	st.shells.view.Error = ""
	st.shells.view.PreferenceError = ""
	save := s.saveShellPreference
	if path != "" {
		s.shellPreference = path
	}
	s.mu.Unlock()
	var err error
	if path != "" && save != nil {
		err = save(path)
	}
	s.mu.Lock()
	if s.epoch == epoch && st.operation == op && err != nil {
		st.shells.view.PreferenceError = "Shell choice is usable for this run, but could not be saved. " + err.Error()
	}
	s.mu.Unlock()
	s.preferenceMu.Unlock()
	return path, nil
}
