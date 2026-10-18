package main

import (
	"context"
	"time"

	"github.com/egoist/mygo"
	"whaleisle/internal/config"
	"whaleisle/internal/desktop/pet"
	"whaleisle/internal/harness"
)

const petPersonalityRetry = 30 * time.Second
const petPersonalityMaxRejectRetries = 3

type petPersonalityTarget struct {
	window     *mygo.Window
	companion  *pet.Companion
	store      *config.Store
	generation int
	origin     string
	cookie     string
	ready      bool
}

type petPersonalityAttempt struct {
	target petPersonalityTarget
	cancel context.CancelFunc
}

// petPersonalityMirror belongs to the UI thread. Only an explicit saved
// personality choice (including one restored on startup) queues a value.
// Constructing a companion without that choice never mirrors a default.
type petPersonalityMirror struct {
	restoredOwner  petPersonalityTarget
	desired        string
	serial         uint64
	rejectRetries  int
	owner          petPersonalityTarget
	attempt        *petPersonalityAttempt
	timer          *time.Timer
	timerSerial    uint64
	next           time.Time
	waitingForHost bool
}

func (s *mainState) petPersonalityTarget() petPersonalityTarget {
	s.mu.Lock()
	host := s.bootStatus
	target := petPersonalityTarget{
		window: s.win, companion: s.desktopPet, store: s.cfgStore,
		generation: host.Generation, origin: host.Endpoint.Origin, cookie: host.Endpoint.Cookie,
		ready: !s.quitting && s.dshMuxReady && host.State == harness.Ready,
	}
	quitting := s.quitting
	s.mu.Unlock()
	if quitting || target.window == nil || target.window.IsDestroyed() || target.companion == nil || target.store == nil {
		target.window = nil
		target.ready = false
		return target
	}
	target.ready = target.ready && target.store.Get().WhaleAssistantEnabled && target.origin != ""
	return target
}

func (t petPersonalityTarget) sameOwner(other petPersonalityTarget) bool {
	return t.window != nil && t.window == other.window && t.companion == other.companion && t.store == other.store
}

func (t petPersonalityTarget) sameHost(other petPersonalityTarget) bool {
	return t.sameOwner(other) && t.generation == other.generation && t.origin == other.origin && t.cookie == other.cookie
}

// queuePetPersonality is wired to Companion.OnPersonality, which runs only
// after the personality is explicitly persisted. Draft edits and Save
// failures cannot modify the assistant. A newer saved value cancels old work.
func (s *mainState) queuePetPersonality(personality string) {
	if !validPetPersonality(personality) {
		return
	}
	target := s.petPersonalityTarget()
	if target.window == nil || target.store.DesktopPet().Behavior.Personality != personality {
		return
	}
	s.cancelPetPersonality()
	m := &s.petPersonality
	m.restoredOwner = target
	m.desired = personality
	m.owner = target
	s.syncPetPersonality()
}

// cancelPetPersonality is called when the main window closes. Hiding the pet
// does not cancel a saved setting; its mirror also works with the pet disabled.
func (s *mainState) cancelPetPersonality() {
	m := &s.petPersonality
	if m.attempt != nil {
		m.attempt.cancel()
	}
	if m.timer != nil {
		m.timer.Stop()
	}
	serial, restoredOwner := m.serial+1, m.restoredOwner
	*m = petPersonalityMirror{serial: serial, restoredOwner: restoredOwner}
}

func (s *mainState) schedulePetPersonality() {
	m := &s.petPersonality
	if m.desired == "" || m.timer != nil || m.owner.window == nil {
		return
	}
	if m.next.IsZero() {
		m.next = time.Now().Add(petPersonalityRetry)
	}
	delay := time.Until(m.next)
	if delay < 0 {
		delay = 0
	}
	m.timerSerial++
	serial, timerSerial, window := m.serial, m.timerSerial, m.owner.window
	m.timer = time.AfterFunc(delay, func() {
		window.Update(func() {
			if s.petPersonality.serial != serial || s.petPersonality.timerSerial != timerSerial {
				return
			}
			s.petPersonality.timer = nil
			s.petPersonality.next = time.Time{}
			s.syncPetPersonality()
		})
	})
}

// syncPetPersonality runs with refreshPetTask on the UI thread, so a Host or
// mux transition immediately invalidates the previous request. The latest
// explicit choice remains queued for the currently configured, enabled Host.
func (s *mainState) syncPetPersonality() {
	m := &s.petPersonality
	target := s.petPersonalityTarget()
	if target.window != nil && !m.restoredOwner.sameOwner(target) {
		m.restoredOwner = target
		if personality, ok := target.store.SavedPetPersonality(); ok {
			s.queuePetPersonality(personality)
			return
		}
	}
	if m.desired == "" {
		return
	}
	if !m.owner.sameOwner(target) || target.store.DesktopPet().Behavior.Personality != m.desired {
		s.cancelPetPersonality()
		return
	}
	if m.attempt != nil {
		if target.ready && m.attempt.target.sameHost(target) {
			return
		}
		m.attempt.cancel()
		m.attempt = nil
		m.next = time.Time{}
		m.waitingForHost = true
	}
	if !target.ready {
		m.waitingForHost = true
		s.schedulePetPersonality()
		return
	}
	if m.waitingForHost {
		// Connection recovery can flush immediately without waiting for the
		// parked timer; a mounted-route transport error still backs off 30s.
		m.waitingForHost = false
		m.next = time.Time{}
	}
	if !m.next.IsZero() && time.Now().Before(m.next) {
		s.schedulePetPersonality()
		return
	}
	if m.timer != nil {
		m.timer.Stop()
		m.timer = nil
		m.timerSerial++
	}
	m.next = time.Time{}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	attempt := &petPersonalityAttempt{target: target, cancel: cancel}
	m.attempt = attempt
	serial, desired := m.serial, m.desired
	go func() {
		// Recheck the authenticated target immediately before sending. The
		// goroutine never mutates UI-owned state or uses a renderer URL.
		now := s.petPersonalityTarget()
		result := petPersonalityTransport
		if ctx.Err() == nil && now.ready && target.sameHost(now) && target.store.DesktopPet().Behavior.Personality == desired {
			result = postPetPersonality(ctx, target.origin, target.cookie, desired)
		}
		target.window.Update(func() {
			defer cancel()
			m := &s.petPersonality
			if m.serial != serial || m.attempt != attempt || m.desired != desired {
				return
			}
			now := s.petPersonalityTarget()
			if !now.ready || !target.sameHost(now) || now.store.DesktopPet().Behavior.Personality != desired {
				s.syncPetPersonality()
				return
			}
			m.attempt = nil
			switch result {
			case petPersonalityAccepted:
				s.cancelPetPersonality()
				return
			case petPersonalityRejected:
				// Original contract: initial answered attempt plus three retries.
				if m.rejectRetries >= petPersonalityMaxRejectRetries {
					s.cancelPetPersonality()
					return
				}
				m.rejectRetries++
			}
			m.next = time.Now().Add(petPersonalityRetry)
			s.schedulePetPersonality()
		})
	}()
}
