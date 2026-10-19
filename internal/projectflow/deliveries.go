package projectflow

import (
	"context"
	"encoding/json"
	"fmt"
)

type DeliveryReportRef struct {
	WorkstreamID  string `json:"workstreamId"`
	DelegationRef string `json:"delegationRef"`
	ReportID      string `json:"reportId"`
}
type DeliveryTerminalRef struct {
	WorkstreamID  string `json:"workstreamId"`
	DelegationRef string `json:"delegationRef"`
	RunID         string `json:"runId"`
}
type DeliveryPageFile struct {
	AttachmentID string `json:"attachmentId"`
	Name         string `json:"name"`
	Bytes        int64  `json:"bytes"`
}

type DeliveryPageReference struct {
	SessionID string           `json:"sessionId"`
	File      DeliveryPageFile `json:"file"`
	Title     string           `json:"title"`
	CallID    string           `json:"callId"`
	Height    int              `json:"height"`
}

// HistoricalDelivery is the full recorded handover, not the current request
// card or a summary preview. Different IDs at one revision remain distinct.
type HistoricalDelivery struct {
	ID              string                  `json:"id"`
	ProjectID       string                  `json:"projectId"`
	RequestID       string                  `json:"requestId"`
	RequestRevision int                     `json:"requestRevision"`
	Title           string                  `json:"title"`
	Status          string                  `json:"status"`
	Outcome         string                  `json:"outcome"`
	Summary         string                  `json:"summary"`
	Evidence        []string                `json:"evidence"`
	RemainingIssues []string                `json:"remainingIssues"`
	At              float64                 `json:"at"`
	Applicable      bool                    `json:"applicable"`
	ReportRefs      []DeliveryReportRef     `json:"reportRefs"`
	TerminalRefs    []DeliveryTerminalRef   `json:"terminalRefs"`
	Artifacts       []Artifact              `json:"artifacts"`
	Pages           []DeliveryPageReference `json:"pages"`
	Error           json.RawMessage         `json:"error"`
}
type DeliveryHistoryPage struct {
	Items      []HistoricalDelivery `json:"items"`
	Total      int                  `json:"total"`
	NextCursor string               `json:"nextCursor"`
}

func validateDeliveryIdentity(projectID string, d HistoricalDelivery) error {
	if projectID == "" || d.ProjectID != projectID || d.ID == "" || d.RequestID == "" || d.RequestRevision < 1 {
		return fmt.Errorf("Historical delivery Project or version identity does not match")
	}
	if d.Status != "delivered" && d.Status != "historical" && d.Status != "obsolete" {
		return fmt.Errorf("Host returned an unrecorded delivery attempt")
	}
	for _, ref := range d.ReportRefs {
		if ref.WorkstreamID == "" || ref.DelegationRef == "" {
			return fmt.Errorf("Historical delivery has an incomplete report reference")
		}
	}
	return nil
}
func validateDeliveryPage(projectID, cursor string, page DeliveryHistoryPage) error {
	if page.Items == nil {
		return fmt.Errorf("Host returned no historical delivery page")
	}
	seen := map[string]bool{}
	for _, d := range page.Items {
		if err := validateDeliveryIdentity(projectID, d); err != nil {
			return err
		}
		if seen[d.ID] {
			return fmt.Errorf("Host returned duplicate historical delivery identities")
		}
		seen[d.ID] = true
	}
	if page.NextCursor != "" && (len(page.Items) == 0 || page.NextCursor != page.Items[len(page.Items)-1].ID || page.NextCursor == cursor) {
		return fmt.Errorf("Host returned an invalid historical delivery cursor")
	}
	return nil
}
func (c Client) DeliveryHistory(ctx context.Context, projectID, cursor string) (DeliveryHistoryPage, error) {
	var out struct {
		Deliveries DeliveryHistoryPage `json:"deliveries"`
	}
	if projectID == "" {
		return out.Deliveries, fmt.Errorf("Select a Project")
	}
	payload := map[string]any{"projectId": projectID, "kind": "deliveries"}
	if cursor != "" {
		payload["cursor"] = cursor
	}
	if err := c.call(ctx, "store/list", payload, &out); err != nil {
		return DeliveryHistoryPage{}, err
	}
	if err := validateDeliveryPage(projectID, cursor, out.Deliveries); err != nil {
		return DeliveryHistoryPage{}, err
	}
	return out.Deliveries, nil
}
func deliveryHasReport(d HistoricalDelivery, ref DeliveryReportRef) bool {
	for _, candidate := range d.ReportRefs {
		if candidate == ref {
			return true
		}
	}
	return false
}

// DeliveryReport follows the frozen report reference, never the work's latest
// report. A report's original requirement revision may precede the handover;
// retain and show that recorded provenance rather than relabelling it.
func (c Client) DeliveryReport(ctx context.Context, projectID string, d HistoricalDelivery, ref DeliveryReportRef) (Report, error) {
	if err := validateDeliveryIdentity(projectID, d); err != nil {
		return Report{}, err
	}
	if !deliveryHasReport(d, ref) {
		return Report{}, fmt.Errorf("Report does not belong to this recorded delivery")
	}
	page, err := c.ReportHistory(ctx, projectID, ref.WorkstreamID, "", ref.DelegationRef)
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
func validateDeliveryReport(ref DeliveryReportRef, report Report) error {
	if report.DelegationRef != ref.DelegationRef || ref.ReportID != "" && report.CallID != ref.ReportID {
		return fmt.Errorf("Host did not return the exact report recorded by this delivery")
	}
	return nil
}

// Scope owns both the opaque cursor chain and selected historical version.
type DeliveryScope struct {
	Host, ProjectID string
	Generation      uint64
}
type DeliverySnapshot struct {
	Scope        DeliveryScope
	Title        string
	Page         DeliveryHistoryPage
	Cursors      []string
	Loaded, Busy bool
	Selected     *HistoricalDelivery
	ReportRef    *DeliveryReportRef
	Report       *Report
	Error        string
}
type DeliveryBrowser struct {
	state  DeliverySnapshot
	serial uint64
	active bool
}

// NavigationToken identifies this exact delivery selection/read generation.
func (b *DeliveryBrowser) NavigationToken() uint64 { return b.serial }

type DeliveryRequest struct {
	owner     *DeliveryBrowser
	scope     DeliveryScope
	serial    uint64
	kind      string
	cursors   []string
	delivery  HistoricalDelivery
	reference DeliveryReportRef
}

func (r DeliveryRequest) ProjectID() string { return r.scope.ProjectID }
func (r DeliveryRequest) Cursor() string {
	if len(r.cursors) == 0 {
		return ""
	}
	return r.cursors[len(r.cursors)-1]
}
func (r DeliveryRequest) Delivery() HistoricalDelivery { return cloneHistoricalDelivery(r.delivery) }
func (r DeliveryRequest) Reference() DeliveryReportRef { return r.reference }
func NewDeliveryBrowser(scope DeliveryScope, title string) *DeliveryBrowser {
	return &DeliveryBrowser{state: DeliverySnapshot{Scope: scope, Title: title, Cursors: []string{""}}, active: true}
}
func cloneHistoricalDelivery(d HistoricalDelivery) HistoricalDelivery {
	d.Evidence = append([]string(nil), d.Evidence...)
	d.RemainingIssues = append([]string(nil), d.RemainingIssues...)
	d.ReportRefs = append([]DeliveryReportRef(nil), d.ReportRefs...)
	d.TerminalRefs = append([]DeliveryTerminalRef(nil), d.TerminalRefs...)
	d.Artifacts = append([]Artifact(nil), d.Artifacts...)
	d.Pages = append([]DeliveryPageReference(nil), d.Pages...)
	d.Error = append(json.RawMessage(nil), d.Error...)
	return d
}
func cloneDeliveryPage(p DeliveryHistoryPage) DeliveryHistoryPage {
	items := make([]HistoricalDelivery, len(p.Items))
	for i, d := range p.Items {
		items[i] = cloneHistoricalDelivery(d)
	}
	p.Items = items
	return p
}
func cloneDeliveryReport(r Report) Report {
	r.Evidence = append([]string(nil), r.Evidence...)
	r.RemainingIssues = append([]string(nil), r.RemainingIssues...)
	r.Artifacts = append([]Artifact(nil), r.Artifacts...)
	r.Request = append(json.RawMessage(nil), r.Request...)
	return r
}
func (b *DeliveryBrowser) Snapshot() DeliverySnapshot {
	s := b.state
	s.Page = cloneDeliveryPage(s.Page)
	s.Cursors = append([]string(nil), s.Cursors...)
	if s.Selected != nil {
		d := cloneHistoricalDelivery(*s.Selected)
		s.Selected = &d
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
func (b *DeliveryBrowser) begin(kind string) (DeliveryRequest, error) {
	if !b.active || b.state.Scope.Host == "" || b.state.Scope.ProjectID == "" {
		return DeliveryRequest{}, fmt.Errorf("Project delivery history is closed")
	}
	if b.state.Busy {
		return DeliveryRequest{}, fmt.Errorf("A historical delivery read is pending")
	}
	b.serial++
	b.state.Busy = true
	b.state.Error = ""
	return DeliveryRequest{owner: b, scope: b.state.Scope, serial: b.serial, kind: kind}, nil
}
func (b *DeliveryBrowser) Pending(r DeliveryRequest) bool {
	return b.active && b.state.Busy && r.owner == b && r.scope == b.state.Scope && r.serial == b.serial
}
func (b *DeliveryBrowser) BeginPage(action string) (DeliveryRequest, error) {
	if b.state.Selected != nil {
		return DeliveryRequest{}, fmt.Errorf("Return to the delivery collection before changing pages")
	}
	cursors := append([]string(nil), b.state.Cursors...)
	switch action {
	case "refresh":
	case "next":
		if b.state.Page.NextCursor == "" {
			return DeliveryRequest{}, fmt.Errorf("No earlier delivery page")
		}
		cursors = append(cursors, b.state.Page.NextCursor)
	case "previous":
		if len(cursors) < 2 {
			return DeliveryRequest{}, fmt.Errorf("No previous delivery page")
		}
		cursors = cursors[:len(cursors)-1]
	default:
		return DeliveryRequest{}, fmt.Errorf("Unsupported delivery page action")
	}
	r, err := b.begin("page")
	r.cursors = cursors
	return r, err
}
func (b *DeliveryBrowser) CompletePage(r DeliveryRequest, page DeliveryHistoryPage, err error) bool {
	if !b.Pending(r) || r.kind != "page" {
		return false
	}
	b.state.Busy = false
	if err == nil {
		err = validateDeliveryPage(r.scope.ProjectID, r.Cursor(), page)
	}
	if err == nil && page.NextCursor != "" {
		for _, cursor := range r.cursors {
			if cursor == page.NextCursor {
				err = fmt.Errorf("Host repeated an earlier delivery cursor")
				break
			}
		}
	}
	if err != nil {
		b.state.Error = err.Error()
		return true
	}
	b.state.Page = cloneDeliveryPage(page)
	b.state.Cursors = append([]string(nil), r.cursors...)
	b.state.Loaded = true
	b.state.Error = ""
	return true
}
func (b *DeliveryBrowser) Select(id string) error {
	if !b.active || b.state.Busy || !b.state.Loaded {
		return fmt.Errorf("Historical delivery collection is not ready")
	}
	for _, d := range b.state.Page.Items {
		if d.ID == id {
			copy := cloneHistoricalDelivery(d)
			b.serial++
			b.state.Selected = &copy
			b.state.ReportRef = nil
			b.state.Report = nil
			b.state.Error = ""
			return nil
		}
	}
	return fmt.Errorf("Selected delivery is not in this Project history page")
}
func (b *DeliveryBrowser) BeginReport(deliveryID string, ref DeliveryReportRef) (DeliveryRequest, error) {
	d := b.state.Selected
	if d == nil || d.ID != deliveryID || !deliveryHasReport(*d, ref) {
		return DeliveryRequest{}, fmt.Errorf("Select a report from this exact historical delivery")
	}
	r, err := b.begin("report")
	if err != nil {
		return r, err
	}
	r.delivery = cloneHistoricalDelivery(*d)
	r.reference = ref
	if b.state.ReportRef == nil || *b.state.ReportRef != ref {
		b.state.Report = nil
	}
	copy := ref
	b.state.ReportRef = &copy
	return r, nil
}
func (b *DeliveryBrowser) CompleteReport(r DeliveryRequest, report Report, err error) bool {
	if !b.Pending(r) || r.kind != "report" || b.state.Selected == nil || b.state.Selected.ID != r.delivery.ID || b.state.Selected.RequestID != r.delivery.RequestID || b.state.Selected.RequestRevision != r.delivery.RequestRevision || b.state.ReportRef == nil || *b.state.ReportRef != r.reference {
		return false
	}
	b.state.Busy = false
	if err == nil {
		err = validateDeliveryReport(r.reference, report)
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
func (b *DeliveryBrowser) BackReport() {
	b.serial++
	b.state.Busy = false
	b.state.ReportRef = nil
	b.state.Report = nil
	b.state.Error = ""
}
func (b *DeliveryBrowser) BackList() { b.BackReport(); b.state.Selected = nil }
func (b *DeliveryBrowser) Close()    { b.BackList(); b.active = false }
