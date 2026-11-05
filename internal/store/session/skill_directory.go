package session

import (
	"context"
	"fmt"
	"time"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/store/storelocale"
)

// SkillRemovalReview is a one-use, action-time confirmation, never a name-only command.
type SkillRemovalReview struct {
	ID             uint64
	Label, Message string
	Ready, Busy    bool
	target         SkillEditTarget
	endpoint       dsh.Endpoint
	epoch          uint64
	directory      string
}

func (s *Store) removalCurrentLocked(r SkillRemovalReview, version uint64) bool {
	return s.skillEditCurrentLocked(r.target, version) && s.client.Endpoint == r.endpoint && s.composerHostGeneration == r.epoch
}
func (s *Store) SkillRemovalReview() SkillRemovalReview {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.skillRemoval
	if r.Ready && !s.removalCurrentLocked(r, r.target.version) {
		r.Message = "Target or catalog changed; cancel and review again."
	}
	r.Ready = r.Ready && !s.skillCatalog.Busy && s.removalCurrentLocked(r, r.target.version)
	return r
}
func (s *Store) CancelSkillRemoval(id uint64) {
	s.mu.Lock()
	if s.skillRemoval.ID == id && !s.skillRemoval.Busy {
		s.skillRemoval = SkillRemovalReview{}
		s.skillRemovalSequence++
	}
	s.mu.Unlock()
	s.fire()
}
func (s *Store) ReviewSkillRemoval(ctx context.Context, name string) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	t, err := s.CaptureCatalogSkillEdit(name)
	if err != nil {
		s.skillInventoryTargetError(err)
		return
	}
	s.mu.Lock()
	if s.skillRemoval.Busy || !s.skillEditCurrentLocked(t, t.version) {
		s.mu.Unlock()
		return
	}
	s.skillRemovalSequence++
	r := SkillRemovalReview{ID: s.skillRemovalSequence, target: t, endpoint: s.client.Endpoint, epoch: s.composerHostGeneration, Message: "Reading deletion target…"}
	for _, row := range s.skillCatalog.Skills {
		if row.Name == t.name && row.Directory != nil {
			r.directory = *row.Directory
		}
	}
	r.Label = "Host: " + r.endpoint.Origin + "\n" + t.Label() + "\nDirectory: " + r.directory + "\nSession: " + t.scopeSession + "\nWorkspace: " + t.scopeCwd
	s.skillRemoval = r
	s.invalidateViewLocked()
	s.mu.Unlock()
	s.fire()
	loaded, err := s.LoadCatalogSkillEdit(ctx, t)
	s.mu.Lock()
	if s.skillRemoval.ID != r.ID || !s.removalCurrentLocked(r, t.version) {
		s.mu.Unlock()
		return
	}
	if err != nil {
		s.skillRemoval.Message = err.Error()
	} else if !loaded.Writable() {
		s.skillRemoval.Message = "This Skill is read-only."
	} else {
		s.skillRemoval.target = loaded
		s.skillRemoval.Ready = true
		s.skillRemoval.Message = storelocale.T(storelocale.KeyStoreSessionPermanentlyDeleteThisSkillDirectory)
	}
	s.mu.Unlock()
	s.fire()
}

// ConfirmSkillRemoval consumes the review before I/O. Unknown results are read
// back in the captured scope and are never retried as writes. The Host API has
// no atomic path/revision CAS; the immediate read-before-delete is best effort.
func (s *Store) ConfirmSkillRemoval(ctx context.Context, id uint64) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	s.mu.Lock()
	r := s.skillRemoval
	if r.ID != id || !r.Ready || r.Busy || s.skillCatalog.Busy || !s.removalCurrentLocked(r, r.target.version) {
		s.mu.Unlock()
		return
	}
	s.skillRemoval.Ready = false
	s.skillRemoval.Busy = true
	s.skillCatalog.Busy = true
	s.skillCatalogVersion++
	version := s.skillCatalogVersion
	s.invalidateViewLocked()
	rpc := &dsh.Client{Endpoint: r.endpoint, HTTP: r.target.client.HTTP}
	s.mu.Unlock()
	s.fire()
	// Notification callbacks may have rebound the same Host before I/O.
	s.mu.Lock()
	valid := ctx.Err() == nil && s.removalCurrentLocked(r, version)
	if !valid {
		s.retireSkillCatalogMutationLocked(version)
		if s.skillRemoval.ID == id {
			s.skillRemoval.Busy = false
			s.skillRemoval.Message = "Host, account, session or catalog changed; deletion cancelled"
		}
	}
	s.mu.Unlock()
	if !valid {
		s.fire()
		return
	}
	var doc api.SkillInventoryDetail
	err := rpc.Call(ctx, "skillInventory/get", map[string]any{"request": r.target.request()}, &doc)
	attempted := false
	if err == nil && (!r.target.matchesFile(doc) || skillEditFingerprint(doc) != r.target.fingerprint) {
		err = fmt.Errorf("Skill changed after review; cancel and review again")
	}
	s.mu.Lock()
	valid = ctx.Err() == nil && s.removalCurrentLocked(r, version)
	s.mu.Unlock()
	if err == nil && !valid {
		err = fmt.Errorf("Host, account, session or catalog changed; deletion cancelled")
	}
	if err == nil {
		attempted = true
		var out api.DeleteResult2
		err = rpc.Call(ctx, "skillInventory/delete", map[string]any{"request": r.target.request()}, &out)
	}
	message := ""
	var inv api.SkillInventorySnapshot
	var readErr error
	if attempted {
		// A timed-out write must not prevent a fresh read-only reconciliation.
		readCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		scope := r.target.request()
		delete(scope, "name")
		readErr = rpc.Call(readCtx, "skillInventory/list", map[string]any{"request": scope}, &inv)
		cancel()
		if readErr == nil && inv.Skills == nil {
			readErr = fmt.Errorf("Host returned no valid skills array")
		}
		if readErr != nil {
			message = "Deletion result unknown; refresh the original scope before any new review. " + readErr.Error()
		} else {
			present := false
			for _, row := range inv.Skills {
				if row.Name == r.target.name && row.Source == r.target.source && row.Path != nil && *row.Path == r.target.path {
					present = true
				}
			}
			if present {
				message = "The reviewed Skill is still present. No automatic retry was made; cancel and review again."
			} else {
				message = "The reviewed Skill is absent from the original catalog after deletion."
			}
		}
	} else if err != nil {
		message = err.Error()
	}
	s.mu.Lock()
	if s.skillRemoval.ID == id {
		s.skillRemoval.Busy = false
		s.skillRemoval.Message = message
		s.invalidateViewLocked()
	}
	refresh := false
	if s.removalCurrentLocked(r, version) {
		s.invalidateViewLocked()
		s.skillCatalog.Busy = false
		s.skillCatalog.Action = message
		if attempted && readErr == nil {
			s.skillCatalog.Skills = inv.Skills
			// Deleting a winner can also expose a lower-priority skill with the
			// same name. Retire the old picker and reload its actual Session view.
			s.skillOptionsEpoch++
			s.skillOptions = nil
			s.skillOptionsOwner = skillOptionsScope{}
			refresh = true
		}
	} else {
		s.retireSkillCatalogMutationLocked(version)
	}
	s.mu.Unlock()
	s.fire()
	if refresh {
		s.ListSkills(ctx)
	}
}

// SkillDirectoryTarget is captured synchronously by the click handler. Its
// fields are private so a worker cannot substitute a different directory/Host.
type SkillDirectoryTarget struct {
	owner     SkillRemovalReview
	directory string
}

func (s *Store) CaptureSkillDirectory(name string) (SkillDirectoryTarget, error) {
	t, err := s.CaptureCatalogSkillEdit(name)
	if err != nil {
		return SkillDirectoryTarget{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.skillEditCurrentLocked(t, t.version) {
		return SkillDirectoryTarget{}, fmt.Errorf("Skill directory target changed")
	}
	r := SkillRemovalReview{target: t, endpoint: t.client.Endpoint, epoch: s.composerHostGeneration}
	for _, row := range s.skillCatalog.Skills {
		if row.Name == name && row.Directory != nil && *row.Directory != "" {
			return SkillDirectoryTarget{owner: r, directory: *row.Directory}, nil
		}
	}
	return SkillDirectoryTarget{}, fmt.Errorf("Skill directory unavailable")
}

// OpenCatalogSkillDirectory is the synchronous convenience entrypoint.
func (s *Store) OpenCatalogSkillDirectory(ctx context.Context, name string) {
	ticket, err := s.CaptureSkillDirectory(name)
	if err != nil {
		s.skillInventoryTargetError(err)
		return
	}
	s.OpenCapturedSkillDirectory(ctx, ticket)
}

// OpenCapturedSkillDirectory uses the original opaque Host path unchanged.
// It never resolves the name again or invokes a local OS shell.
func (s *Store) OpenCapturedSkillDirectory(ctx context.Context, ticket SkillDirectoryTarget) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	r, directory := ticket.owner, ticket.directory
	t := r.target
	s.mu.Lock()
	valid := directory != "" && s.removalCurrentLocked(r, t.version)
	s.mu.Unlock()
	if !valid {
		return
	}
	rpc := &dsh.Client{Endpoint: r.endpoint, HTTP: t.client.HTTP}
	var can api.CanOpenWorkspacePathResult
	err := rpc.Call(ctx, "session/canOpenWorkspacePath", map[string]any{}, &can)
	if err == nil && !can {
		err = fmt.Errorf("Directory opening is unavailable on this Host")
	}
	s.mu.Lock()
	valid = s.removalCurrentLocked(r, t.version)
	s.mu.Unlock()
	if err == nil && !valid {
		err = fmt.Errorf("Skill directory target changed; refresh before opening")
	}
	if err == nil {
		var out api.SessionOpenWorkspacePathValue
		err = rpc.Call(ctx, "session/openWorkspacePath", map[string]any{"request": api.SessionOpenWorkspacePathRequest{Path: directory}}, &out)
		if err == nil && !out.Opened {
			err = fmt.Errorf("Host did not open the Skill directory")
		}
	}
	s.mu.Lock()
	if s.removalCurrentLocked(r, t.version) {
		if err != nil {
			s.lastAction = err.Error()
		} else {
			s.lastAction = "Skill directory opened on its Host"
		}
	}
	s.mu.Unlock()
	s.fire()
}
