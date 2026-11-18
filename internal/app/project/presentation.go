package projectctl

import (
	"fmt"

	"whaleisle/internal/dsh"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/projects"
)

// observerReadingState retains editor and navigation data, not subscriptions,
// browser handles or a Controller. Host state is refreshed when the view returns.
type observerReadingState struct {
	materials                                         map[string]*projectflow.MaterialDraft
	materialKey, teamQuery, searchQuery, section, err string
	teamHistory, history                              bool
	reading                                           projects.ObserverReading
}

type observerIdentity struct{ host, project, coordinator string }

func readingIdentity(o *projects.ObserverState, client *dsh.Client) observerIdentity {
	return observerIdentity{host: fmt.Sprintf("%p", client), project: o.ProjectID, coordinator: o.CoordinatorSessionID}
}

func (s *observerReadingState) restore(f *flowState) {
	f.materials, f.materialKey = s.materials, s.materialKey
	f.teamQuery, f.searchQuery = s.teamQuery, s.searchQuery
	f.teamHistory, f.history = s.teamHistory, s.history
	f.observerSection, f.err = s.section, s.err
}

// BeginPresentationFrame returns the matching release boundary. Calling the
// returned function after all views were built retires every unpresented reader.
// Accepted Host actions use their existing lifecycle contexts and keep running.
func (m *Manager) BeginPresentationFrame() func() {
	m.mu.Lock()
	m.presentationFrame++
	frame := m.presentationFrame
	m.mu.Unlock()
	return func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		if m.presentationFrame != frame {
			return
		}
		if m.primaryPresented != frame {
			m.primary.closeObservationLocked()
		} else {
			m.primary.retireUnpresentedVisualsLocked(frame)
		}
		for observer, c := range m.observers {
			if c.presented != frame {
				m.closeObserverLocked(observer, c)
			} else {
				c.retireUnpresentedVisualsLocked(frame)
			}
		}
	}
}

// CloseObserver releases the current presentation immediately, including a
// popover dismissed during the same frame in which it was drawn.
func (m *Manager) CloseObserver(o *projects.ObserverState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c := m.observers[o]; c != nil {
		m.closeObserverLocked(o, c)
	}
}

func (m *Manager) closeObserverLocked(o *projects.ObserverState, c *Controller) {
	f := &c.flow
	if f.observerOpen {
		f.observerOpen = false
		f.navigation++
	}
	c.closeObservationLocked()
	// Keep only an in-flight business action's owner until its normal callback
	// publishes the result. Closing presentation never cancels or loses that action.
	if f.busy {
		return
	}
	if o != nil {
		if m.observerReadings == nil {
			m.observerReadings = map[observerIdentity]*observerReadingState{}
		}
		m.observerReadings[readingIdentity(o, f.client)] = &observerReadingState{materials: f.materials, materialKey: f.materialKey,
			teamQuery: f.teamQuery, searchQuery: f.searchQuery, section: f.observerSection, err: f.err,
			teamHistory: f.teamHistory, history: f.history, reading: o.Reading()}
		o.DetachReads()
	}
	delete(m.observers, o)
}

func (c *Controller) closeObservationLocked() {
	c.stopProcessLocked()
	c.stopRefreshLocked()
	c.closeHistoricalLocked()
	c.closeActivityLocked()
	c.closeDeliveriesLocked()
	c.closeDocumentsLocked()
	c.closeMaterialCollectionLocked()
	c.closeVisualPagesLocked()
	if c.flow.catalogRead != nil {
		c.finishReadLocked(c.flow.catalogRead)
	}
}
