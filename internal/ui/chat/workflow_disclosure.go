package chat

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/workflowrun"
)

type workflowDisclosureFacts struct {
	mode  string
	count int
}

type workflowDisclosure struct {
	workflowDisclosureFacts
	open, pendingCleanCollapse bool
	content                    ui.Handle
}

type workflowDisclosures struct {
	run    workflowDisclosure
	phases map[string]*workflowDisclosure
}

func workflowPhaseFacts(phase workflowrun.Phase) workflowDisclosureFacts {
	facts := workflowDisclosureFacts{mode: "clean", count: len(phase.Members)}
	for _, member := range phase.Members {
		if workflowAbnormal(member.Status) {
			facts.mode = "abnormal"
			break
		}
		if member.Status == "running" {
			facts.mode = "running"
		}
	}
	return facts
}

func workflowRunFacts(run workflowrun.Run) workflowDisclosureFacts {
	facts := workflowDisclosureFacts{mode: "clean"}
	if workflowAbnormal(run.Status) {
		facts.mode = "abnormal"
	} else if run.Status == "running" {
		facts.mode = "running"
	}
	for _, phase := range run.Phases {
		p := workflowPhaseFacts(phase)
		facts.count += p.count
		if p.mode == "abnormal" || p.mode == "running" && facts.mode == "clean" {
			facts.mode = p.mode
		}
	}
	return facts
}

func newWorkflowDisclosure(facts workflowDisclosureFacts) workflowDisclosure {
	return workflowDisclosure{workflowDisclosureFacts: facts, open: facts.mode != "clean"}
}

// A clean transition must not unmount the member action holding focus; it
// settles after focus leaves, as in WorkflowRunPanel's advanceDisclosureState.
func (s *workflowDisclosure) advance(facts workflowDisclosureFacts, focusWithin bool) {
	if s.workflowDisclosureFacts == facts {
		if s.pendingCleanCollapse && !focusWithin {
			s.open, s.pendingCleanCollapse = false, false
		}
		return
	}
	previous := s.mode
	s.workflowDisclosureFacts = facts
	if facts.mode == "clean" {
		s.pendingCleanCollapse = s.open && focusWithin
		s.open = s.pendingCleanCollapse
		return
	}
	if previous == "clean" || facts.mode == "abnormal" && previous != "abnormal" {
		s.open = true
	}
	s.pendingCleanCollapse = false
}

func (s *workflowDisclosure) toggle() {
	s.open, s.pendingCleanCollapse = !s.open, false
}

func workflowDisclosureArrow(open bool) string {
	if open {
		return "▾ "
	}
	return "▸ "
}

func newWorkflowDisclosures(run workflowrun.Run) workflowDisclosures {
	s := workflowDisclosures{run: newWorkflowDisclosure(workflowRunFacts(run)), phases: map[string]*workflowDisclosure{}}
	for _, phase := range run.Phases {
		p := newWorkflowDisclosure(workflowPhaseFacts(phase))
		s.phases[phase.Key] = &p
	}
	return s
}

// A phase's new cycle can reveal a manually collapsed run even when another
// phase keeps its aggregate mode abnormal. Removed phases lose their state.
func (s *workflowDisclosures) syncPhases(run workflowrun.Run) bool {
	startedCycle := false
	current := make(map[string]bool, len(run.Phases))
	for _, phase := range run.Phases {
		current[phase.Key] = true
		facts := workflowPhaseFacts(phase)
		if previous := s.phases[phase.Key]; previous != nil {
			if previous.mode == "clean" && (facts.mode != "clean" || facts.count != previous.count) {
				startedCycle = true
			}
		} else {
			p := newWorkflowDisclosure(facts)
			s.phases[phase.Key] = &p
		}
	}
	for key := range s.phases {
		if !current[key] {
			delete(s.phases, key)
		}
	}
	return startedCycle
}
