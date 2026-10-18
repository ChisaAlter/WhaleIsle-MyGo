package main

import (
	"context"
	"fmt"
	"github.com/egoist/mygo"
	"time"
	"whaleisle/internal/desktop/pet"
	"whaleisle/internal/harness"
	"whaleisle/internal/petwatch"
	"whaleisle/internal/store/session"
)

type petEventTarget struct {
	window         *mygo.Window
	pet            *pet.Companion
	supervisor     *harness.Supervisor
	sessions       *session.Store
	source         harness.PetEventSource
	origin, cookie string
}
type petEventController struct {
	target petEventTarget
	serial uint64
	owner  string
	cancel context.CancelFunc
	done   chan struct{}
	timer  *time.Timer
}

func (s *mainState) petEventTarget() (petEventTarget, bool) {
	s.mu.Lock()
	t := petEventTarget{window: s.win, pet: s.desktopPet, supervisor: s.sup, sessions: s.sessions, origin: s.bootStatus.Endpoint.Origin, cookie: s.bootStatus.Endpoint.Cookie}
	ready := !s.quitting && s.dshMuxReady && s.bootStatus.State == harness.Ready
	generation := s.bootStatus.Generation
	s.mu.Unlock()
	if !ready || t.window == nil || t.window.IsDestroyed() || t.pet == nil || !t.pet.NotificationsActive() || t.supervisor == nil || t.sessions == nil {
		return t, false
	}
	source, ok := t.supervisor.PetEventSource()
	if !ok || source.Generation != generation || source.Origin != t.origin {
		return t, false
	}
	t.source = source
	return t, true
}
func (s *mainState) cancelPetEvents() {
	m := &s.petEvents
	if m.cancel != nil {
		m.cancel()
	}
	if m.timer != nil {
		m.timer.Stop()
	}
	if m.target.pet != nil {
		m.target.pet.ClearNotificationSource()
	}
	// Retain the exit barrier: the next generation must not overlap checkpoint
	// writers, even when a queued old callback arrives after its pet closed.
	*m = petEventController{serial: m.serial + 1, done: m.done}
}
func (s *mainState) syncPetEvents() {
	target, ready := s.petEventTarget()
	m := &s.petEvents
	if !ready {
		s.cancelPetEvents()
		return
	}
	if m.target == target && (m.cancel != nil || m.timer != nil) {
		return
	}
	s.cancelPetEvents()
	m = &s.petEvents
	previous := m.done
	ctx, cancel := context.WithCancel(context.Background())
	m.serial++
	owner := fmt.Sprint(m.serial)
	done := make(chan struct{})
	m.target = target
	m.owner = owner
	m.cancel = cancel
	m.done = done
	acknowledgements := make(chan string, 256)
	acknowledge := func(id string) {
		select {
		case acknowledgements <- id:
		case <-ctx.Done():
		default:
		}
	}
	current := func() bool {
		actual, ok := s.petEventTarget()
		return ok && actual == target && s.petEvents.owner == owner && ctx.Err() == nil
	}
	go func() {
		defer close(done)
		if previous != nil {
			select {
			case <-previous:
			case <-ctx.Done():
				return
			}
		}
		err := harness.WatchPetEvents(ctx, target.source, acknowledgements, func(snapshot petwatch.Snapshot) {
			target.window.Update(func() {
				if current() {
					target.pet.UpdateNotifications(owner, snapshot, acknowledge, target.source.Home, target.source.StickerRoot)
				}
			})
		})
		if ctx.Err() != nil {
			return
		}
		target.window.Update(func() {
			if !current() {
				return
			}
			_ = err // Do not expose filesystem/runtime errors or source paths to the page.
			target.pet.UpdateNotifications(owner, petwatch.Snapshot{State: "unavailable"}, acknowledge, target.source.Home, target.source.StickerRoot)
			s.petEvents.cancel = nil
			s.petEvents.timer = time.AfterFunc(30*time.Second, func() {
				target.window.Update(func() {
					if s.petEvents.owner != owner {
						return
					}
					s.petEvents.timer = nil
					s.syncPetEvents()
				})
			})
		})
	}()
}

func (s *mainState) openPetNotificationSession(owner, id string) {
	target, ready := s.petEventTarget()
	if !ready || owner == "" || owner != s.petEvents.owner || target != s.petEvents.target || id == "" {
		return
	}
	// session/list is the authenticated full live roster. Never reinterpret the
	// receipt as the currently selected chat, restore archives or invent an RPC.
	snap := target.sessions.Snapshot()
	if !snap.ClientReady || snap.ListLoading || snap.ListErr != "" {
		return
	}
	for _, archived := range snap.ArchivedSessionIDs {
		if archived == id {
			return
		}
	}
	for _, item := range snap.Items {
		if item.SessionID == id {
			s.activatePetConversation(target.sessions, id)
			return
		}
	}
}
