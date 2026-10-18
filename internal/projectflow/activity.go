package projectflow

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
)

// The original Activity tab is a projection of requests/get, not an event-log
// endpoint. Keep its actual snapshot boundary and do not invent Host cursors.
const ActivityPageSize = 20

type ActivityVersion struct {
	Revision int      `json:"revision"`
	Title    string   `json:"title"`
	Goal     string   `json:"goal"`
	At       *float64 `json:"at"`
}
type ActivityWork struct {
	WorkstreamID       string          `json:"workstreamId"`
	DelegationRef      string          `json:"delegationRef"`
	RunID              string          `json:"runId"`
	Title              string          `json:"title"`
	WorkerSessionID    string          `json:"workerSessionId"`
	MemberName         string          `json:"memberName"`
	Status             string          `json:"status"`
	Current            bool            `json:"current"`
	Phase              string          `json:"phase"`
	Summary            string          `json:"summary"`
	ReportID           string          `json:"reportId"`
	TerminalStopReason string          `json:"terminalStopReason"`
	BlockedReason      string          `json:"blockedReason"`
	WaitingReason      json.RawMessage `json:"waitingReason"`
	AssignmentAt       *float64        `json:"assignmentAt"`
	ReportAt           *float64        `json:"reportAt"`
	TerminalAt         *float64        `json:"terminalAt"`
}
type ActivityDelivery struct {
	ID              string                  `json:"id"`
	Summary         string                  `json:"summary"`
	Outcome         string                  `json:"outcome"`
	Applicable      bool                    `json:"applicable"`
	At              *float64                `json:"at"`
	Evidence        []string                `json:"evidence"`
	RemainingIssues []string                `json:"remainingIssues"`
	ReportRefs      []DeliveryReportRef     `json:"reportRefs"`
	TerminalRefs    []DeliveryTerminalRef   `json:"terminalRefs"`
	Artifacts       []Artifact              `json:"artifacts"`
	Pages           []DeliveryPageReference `json:"pages"`
}
type RequestActivityCard struct {
	ID             string            `json:"id"`
	ProjectID      string            `json:"projectId"`
	Revision       int               `json:"revision"`
	Title          string            `json:"title"`
	AuthorityError string            `json:"authorityError"`
	HandoverError  string            `json:"handoverError"`
	Versions       []ActivityVersion `json:"versions"`
	Work           []ActivityWork    `json:"work"`
	Delivery       *ActivityDelivery `json:"delivery"`
}
type ActivityEvent struct {
	Key, Kind, ProjectID, RequestID, Title string
	SnapshotRevision, Revision             int
	At                                     float64
	Version                                *ActivityVersion
	Work                                   *ActivityWork
	Delivery                               *ActivityDelivery
}

func activityFinite(at *float64) bool { return at != nil && !math.IsNaN(*at) && !math.IsInf(*at, 0) }
func activityKey(kind string, parts ...any) string {
	raw, _ := json.Marshal(parts)
	return kind + ":" + string(raw)
}
func ProjectActivityEvents(card RequestActivityCard) ([]ActivityEvent, error) {
	if card.ProjectID == "" || card.ID == "" || card.Revision < 1 || card.Versions == nil || card.Work == nil {
		return nil, fmt.Errorf("Host returned no complete request activity snapshot")
	}
	var events []ActivityEvent
	versions := map[int]bool{}
	works := map[string]bool{}
	base := func(kind, key, title string, at float64) ActivityEvent {
		return ActivityEvent{Key: key, Kind: kind, ProjectID: card.ProjectID, RequestID: card.ID, SnapshotRevision: card.Revision, Revision: card.Revision, Title: title, At: at}
	}
	for _, version := range card.Versions {
		if version.Revision < 1 || versions[version.Revision] {
			return nil, fmt.Errorf("Host returned an ambiguous requirement version")
		}
		versions[version.Revision] = true
		if !activityFinite(version.At) {
			continue
		}
		v := version
		e := base("version", activityKey("version", version.Revision), version.Title, *version.At)
		e.Revision = version.Revision
		e.Version = &v
		events = append(events, e)
	}
	for _, work := range card.Work {
		identity := activityKey("work", work.WorkstreamID, work.DelegationRef)
		if work.WorkstreamID == "" || work.DelegationRef == "" || works[identity] {
			return nil, fmt.Errorf("Host returned an ambiguous work assignment")
		}
		works[identity] = true
		for _, part := range []struct {
			kind string
			at   *float64
		}{{"assignment", work.AssignmentAt}, {"report", work.ReportAt}, {"terminal", work.TerminalAt}} {
			if !activityFinite(part.at) {
				continue
			}
			w := work
			e := base(part.kind, activityKey(part.kind, work.WorkstreamID, work.DelegationRef, work.RunID), work.Title, *part.at)
			e.Work = &w
			events = append(events, e)
		}
	}
	if d := card.Delivery; d != nil {
		if d.ID == "" {
			return nil, fmt.Errorf("Host returned an unidentified delivery")
		}
		if activityFinite(d.At) {
			copy := *d
			e := base("delivery", activityKey("delivery", d.ID), card.Title, *d.At)
			e.Delivery = &copy
			events = append(events, e)
		}
	}
	sort.SliceStable(events, func(i, j int) bool { return events[i].At > events[j].At })
	return events, nil
}
func (c Client) RequestActivity(ctx context.Context, projectID, requestID string) (RequestActivityCard, error) {
	var out struct {
		Card RequestActivityCard `json:"card"`
	}
	if projectID == "" || requestID == "" {
		return out.Card, fmt.Errorf("Select a Project request")
	}
	if err := c.call(ctx, "requests/get", map[string]any{"projectId": projectID, "requestId": requestID}, &out); err != nil {
		return RequestActivityCard{}, err
	}
	if out.Card.ProjectID != projectID || out.Card.ID != requestID {
		return RequestActivityCard{}, fmt.Errorf("Request activity source identity does not match")
	}
	if _, err := ProjectActivityEvents(out.Card); err != nil {
		return RequestActivityCard{}, err
	}
	return out.Card, nil
}
func ActivityEventReports(event ActivityEvent) []DeliveryReportRef {
	if event.Work != nil && event.Work.ReportID != "" {
		return []DeliveryReportRef{{WorkstreamID: event.Work.WorkstreamID, DelegationRef: event.Work.DelegationRef, ReportID: event.Work.ReportID}}
	}
	if event.Delivery != nil {
		return append([]DeliveryReportRef(nil), event.Delivery.ReportRefs...)
	}
	return nil
}
func activityHasReport(event ActivityEvent, ref DeliveryReportRef) bool {
	for _, r := range ActivityEventReports(event) {
		if r == ref && r.WorkstreamID != "" && r.DelegationRef != "" {
			return true
		}
	}
	return false
}
func (c Client) ActivityReport(ctx context.Context, event ActivityEvent, ref DeliveryReportRef) (Report, error) {
	if event.ProjectID == "" || event.RequestID == "" || event.SnapshotRevision < 1 || !activityHasReport(event, ref) {
		return Report{}, fmt.Errorf("Report does not belong to this request activity entry")
	}
	page, err := c.ReportHistory(ctx, event.ProjectID, ref.WorkstreamID, "", ref.DelegationRef)
	if err != nil {
		return Report{}, err
	}
	report := page.Items[0]
	if err := validateDeliveryReport(ref, report); err != nil {
		return Report{}, err
	}
	if len(report.Request) == 0 || string(report.Request) == "null" {
		report.Request = append(json.RawMessage(nil), page.Request...)
	}
	return report, nil
}
func cloneActivityEvent(e ActivityEvent) ActivityEvent {
	if e.Version != nil {
		v := *e.Version
		if v.At != nil {
			at := *v.At
			v.At = &at
		}
		e.Version = &v
	}
	if e.Work != nil {
		w := *e.Work
		w.WaitingReason = append(json.RawMessage(nil), w.WaitingReason...)
		if w.AssignmentAt != nil {
			at := *w.AssignmentAt
			w.AssignmentAt = &at
		}
		if w.ReportAt != nil {
			at := *w.ReportAt
			w.ReportAt = &at
		}
		if w.TerminalAt != nil {
			at := *w.TerminalAt
			w.TerminalAt = &at
		}
		e.Work = &w
	}
	if e.Delivery != nil {
		d := *e.Delivery
		d.Evidence = append([]string(nil), d.Evidence...)
		d.RemainingIssues = append([]string(nil), d.RemainingIssues...)
		d.ReportRefs = append([]DeliveryReportRef(nil), d.ReportRefs...)
		d.TerminalRefs = append([]DeliveryTerminalRef(nil), d.TerminalRefs...)
		d.Artifacts = append([]Artifact(nil), d.Artifacts...)
		d.Pages = append([]DeliveryPageReference(nil), d.Pages...)
		if d.At != nil {
			at := *d.At
			d.At = &at
		}
		e.Delivery = &d
	}
	return e
}
func cloneActivityEvents(events []ActivityEvent) []ActivityEvent {
	out := make([]ActivityEvent, len(events))
	for i, e := range events {
		out[i] = cloneActivityEvent(e)
	}
	return out
}

type ActivityScope struct {
	Host, ProjectID, RequestID string
	Generation                 uint64
}
type ActivitySnapshot struct {
	Scope                                ActivityScope
	Title                                string
	Revision                             int
	Events, Visible                      []ActivityEvent
	Filter                               string
	Page, FilteredTotal                  int
	View                                 uint64
	Loaded, Busy                         bool
	Error, AuthorityError, HandoverError string
	Selected                             *ActivityEvent
	ReportRef                            *DeliveryReportRef
	Report                               *Report
}
type ActivityBrowser struct {
	state  ActivitySnapshot
	serial uint64
	active bool
}
type ActivityRequest struct {
	owner  *ActivityBrowser
	scope  ActivityScope
	serial uint64
	kind   string
	event  ActivityEvent
	ref    DeliveryReportRef
}

func (r ActivityRequest) Event() ActivityEvent         { return cloneActivityEvent(r.event) }
func (r ActivityRequest) Reference() DeliveryReportRef { return r.ref }
func NewActivityBrowser(scope ActivityScope) *ActivityBrowser {
	return &ActivityBrowser{state: ActivitySnapshot{Scope: scope, Filter: "all"}, active: true}
}
func (b *ActivityBrowser) View() uint64 { return b.serial }
func (b *ActivityBrowser) filtered() []ActivityEvent {
	var out []ActivityEvent
	for _, e := range b.state.Events {
		if b.state.Filter == "all" || e.Kind == b.state.Filter {
			out = append(out, e)
		}
	}
	return out
}
func (b *ActivityBrowser) Snapshot() ActivitySnapshot {
	s := b.state
	s.View = b.serial
	s.Events = cloneActivityEvents(s.Events)
	all := b.filtered()
	s.FilteredTotal = len(all)
	start := min(s.Page*ActivityPageSize, len(all))
	s.Visible = cloneActivityEvents(all[start:min(start+ActivityPageSize, len(all))])
	if s.Selected != nil {
		e := cloneActivityEvent(*s.Selected)
		s.Selected = &e
	}
	if s.ReportRef != nil {
		r := *s.ReportRef
		s.ReportRef = &r
	}
	if s.Report != nil {
		r := cloneDeliveryReport(*s.Report)
		s.Report = &r
	}
	return s
}
func (b *ActivityBrowser) begin(kind string) (ActivityRequest, error) {
	if !b.active || b.state.Scope.Host == "" || b.state.Scope.ProjectID == "" || b.state.Scope.RequestID == "" {
		return ActivityRequest{}, fmt.Errorf("Request activity is closed")
	}
	if b.state.Busy {
		return ActivityRequest{}, fmt.Errorf("An activity read is pending")
	}
	b.serial++
	b.state.Busy = true
	b.state.Error = ""
	return ActivityRequest{owner: b, scope: b.state.Scope, serial: b.serial, kind: kind}, nil
}
func (b *ActivityBrowser) Pending(r ActivityRequest) bool {
	return b.active && b.state.Busy && r.owner == b && r.scope == b.state.Scope && r.serial == b.serial
}
func (b *ActivityBrowser) BeginRefresh() (ActivityRequest, error) {
	if b.state.Selected != nil {
		return ActivityRequest{}, fmt.Errorf("Return to the timeline before refreshing")
	}
	return b.begin("refresh")
}
func (b *ActivityBrowser) CompleteRefresh(r ActivityRequest, card RequestActivityCard, err error) bool {
	if !b.Pending(r) || r.kind != "refresh" {
		return false
	}
	b.state.Busy = false
	if err == nil && (card.ProjectID != r.scope.ProjectID || card.ID != r.scope.RequestID) {
		err = fmt.Errorf("Request activity source identity does not match")
	}
	var events []ActivityEvent
	if err == nil {
		events, err = ProjectActivityEvents(card)
	}
	if err != nil {
		b.state.Error = err.Error()
		return true
	}
	b.state.Title = card.Title
	b.state.Revision = card.Revision
	b.state.Events = cloneActivityEvents(events)
	b.state.Loaded = true
	b.state.Page = 0
	b.state.AuthorityError = card.AuthorityError
	b.state.HandoverError = card.HandoverError
	b.state.Error = ""
	return true
}
func (b *ActivityBrowser) Filter(kind string) error {
	if !b.active || b.state.Busy || b.state.Selected != nil {
		return fmt.Errorf("Return to the idle timeline before filtering")
	}
	switch kind {
	case "all", "version", "assignment", "report", "terminal", "delivery":
	default:
		return fmt.Errorf("Unknown activity type")
	}
	b.serial++
	b.state.Filter = kind
	b.state.Page = 0
	b.state.Error = ""
	return nil
}
func (b *ActivityBrowser) MovePage(direction int) error {
	if !b.active || b.state.Busy || b.state.Selected != nil || direction != 1 && direction != -1 {
		return fmt.Errorf("Activity page is not available")
	}
	next := b.state.Page + direction
	if next < 0 || next*ActivityPageSize >= len(b.filtered()) {
		return fmt.Errorf("No further local activity page")
	}
	b.serial++
	b.state.Page = next
	b.state.Error = ""
	return nil
}
func (b *ActivityBrowser) Select(key string) error {
	if !b.active || b.state.Busy || !b.state.Loaded || b.state.Selected != nil {
		return fmt.Errorf("Activity timeline is not ready")
	}
	for _, event := range b.Snapshot().Visible {
		if event.Key == key {
			copy := cloneActivityEvent(event)
			b.serial++
			b.state.Selected = &copy
			b.state.ReportRef = nil
			b.state.Report = nil
			b.state.Error = ""
			return nil
		}
	}
	return fmt.Errorf("Activity entry is not in this visible request snapshot page")
}
func (b *ActivityBrowser) BeginReport(key string, ref DeliveryReportRef) (ActivityRequest, error) {
	e := b.state.Selected
	if e == nil || e.Key != key || !activityHasReport(*e, ref) {
		return ActivityRequest{}, fmt.Errorf("Select a recorded report from this activity entry")
	}
	r, err := b.begin("report")
	if err != nil {
		return r, err
	}
	r.event = cloneActivityEvent(*e)
	r.ref = ref
	if b.state.ReportRef == nil || *b.state.ReportRef != ref {
		b.state.Report = nil
	}
	copy := ref
	b.state.ReportRef = &copy
	return r, nil
}
func (b *ActivityBrowser) CompleteReport(r ActivityRequest, report Report, err error) bool {
	if !b.Pending(r) || r.kind != "report" || b.state.Selected == nil || b.state.Selected.Key != r.event.Key || b.state.Selected.SnapshotRevision != r.event.SnapshotRevision || b.state.ReportRef == nil || *b.state.ReportRef != r.ref {
		return false
	}
	b.state.Busy = false
	if err == nil {
		err = validateDeliveryReport(r.ref, report)
	}
	if err != nil {
		b.state.Error = err.Error()
		return true
	}
	copy := cloneDeliveryReport(report)
	b.state.Report = &copy
	b.state.Error = ""
	return true
}
func (b *ActivityBrowser) BackReport() {
	b.serial++
	b.state.Busy = false
	b.state.ReportRef = nil
	b.state.Report = nil
	b.state.Error = ""
}
func (b *ActivityBrowser) BackList() { b.BackReport(); b.state.Selected = nil }
func (b *ActivityBrowser) Close()    { b.BackList(); b.active = false }
