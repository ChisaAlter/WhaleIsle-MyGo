package main

import (
	"github.com/egoist/mygo"
	"whaleisle/internal/ui/projects"
)

// projectController reuses native Project operations with one stable state owner.
// Embedding shares the real Host, sessions, composer and mutex; it does not copy
// mainState or temporarily replace any global state around asynchronous work.
type projectController struct {
	*mainState
	projectFlow *projectFlowState
}

func (s *mainState) primaryProjectController() *projectController {
	return &projectController{mainState: s, projectFlow: &s.projectFlow}
}
func (s *mainState) projectModel(win *mygo.Window) projects.Model {
	m := s.primaryProjectController().projectModel(win)
	// Opening an observer does not make the sidebar own its controller lifetime.
	m.OnObserver = nil
	m.OnObserverSection = nil
	m.ObserverModel = func(o *projects.ObserverState) projects.Model {
		s.mu.Lock()
		controller := s.observerControllerLocked(o)
		s.mu.Unlock()
		return controller.projectModel(win)
	}
	return m
}
func (s *mainState) observerControllerLocked(o *projects.ObserverState) *projectController {
	f := &s.projectFlow
	if f.observers == nil {
		f.observers = make(map[*projects.ObserverState]*projectController)
	}
	controller := f.observers[o]
	if controller == nil {
		controller = &projectController{mainState: s, projectFlow: &projectFlowState{client: s.dshClient, listing: f.listing, loaded: f.loaded, observerOpen: true, observerSection: o.Section}}
		f.observers[o] = controller
		o.Initialized = false
	}
	return controller
}

func (s *projectController) disposeLocked() {
	s.stopProjectProcessLocked()
	s.stopProjectRefreshLocked()
	s.closeProjectHistoricalLocked()
	s.closeProjectActivityLocked()
	s.closeProjectDeliveriesLocked()
	s.closeProjectDocumentsLocked()
	s.closeProjectMaterialCollectionLocked()
	s.projectFlow.generation++
	s.projectFlow.client = nil
}

// Existing application lifecycle hooks stop every retained observer, not only
// the sidebar's current Project. Their independent read state remains retained.
func (s *mainState) stopProjectProcessLocked() {
	s.primaryProjectController().stopProjectProcessLocked()
	for _, c := range s.projectFlow.observers {
		c.stopProjectProcessLocked()
	}
}
func (s *mainState) stopProjectRefreshLocked() {
	s.primaryProjectController().stopProjectRefreshLocked()
	for _, c := range s.projectFlow.observers {
		c.stopProjectRefreshLocked()
	}
}
func (s *mainState) closeProjectHistoricalLocked() {
	s.primaryProjectController().closeProjectHistoricalLocked()
	for _, c := range s.projectFlow.observers {
		c.closeProjectHistoricalLocked()
	}
}
