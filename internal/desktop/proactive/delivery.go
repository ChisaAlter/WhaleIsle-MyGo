package proactive

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strings"

	"whaleisle/internal/desktop/pluginrpc"
)

// DeliveryRoute is a previously authorized IM task route, returned by the Host.
// It contains no login credentials and cannot select an arbitrary new recipient.
type DeliveryRoute struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Channel     string `json:"channel"`
	BotID       string `json:"botId"`
	Actor       string `json:"actor"`
	ChatID      string `json:"chatId"`
	Incarnation string `json:"incarnation"`
}

type Delivery struct {
	HostHome   string        `json:"hostHome"`
	RequestID  string        `json:"requestId"`
	Route      DeliveryRoute `json:"route"`
	Text       string        `json:"text"`
	Status     string        `json:"status"`
	Lease      string        `json:"lease,omitempty"`
	Error      string        `json:"error,omitempty"`
	MessageIDs []string      `json:"messageIds,omitempty"`
}

func (s *Service) deliveryPending(taskID string) bool {
	for _, a := range s.approvals {
		if a.TaskID == taskID && a.Delivery != nil && (a.Delivery.Status == "pending" || a.Delivery.Status == "ready" || a.Delivery.Status == "sending") {
			return true
		}
	}
	return false
}

// RequestDelivery is native-only, as is Decide. The HTTP control plane may
// consume this exact approval but cannot manufacture or approve a delivery.
func (s *Service) RequestDelivery(taskID, hostHome string, route DeliveryRoute, text string) (Approval, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tasks[taskID]
	if !ok {
		return Approval{}, ErrNotFound
	}
	if t.Execution == nil || t.Execution.Phase != "completed" || t.Execution.Spec.HostHome != hostHome || (t.State != "verifying" && t.State != "delivered") || t.ApprovalID != "" || s.deliveryPending(taskID) {
		return Approval{}, fmt.Errorf("%w: finish the task and pending delivery first", ErrConflict)
	}
	if route.Channel != "feishu" || route.ID == "" || route.BotID == "" || route.Actor == "" || route.ChatID == "" || route.Incarnation == "" || strings.TrimSpace(text) == "" {
		return Approval{}, fmt.Errorf("an authorized Feishu conversation and message are required")
	}
	if len([]byte(text)) > 20000 {
		return Approval{}, fmt.Errorf("delivery text exceeds 20,000 UTF-8 bytes")
	}
	d := &Delivery{HostHome: hostHome, RequestID: t.Execution.RequestID, Route: route, Text: text, Status: "pending"}
	target, _ := json.Marshal(map[string]any{"route": route, "text": text})
	a := Approval{ID: rand.Text(), TaskID: taskID, Action: "external_message", Target: target, DataClass: "task-result", Risk: "review-required", Status: "pending", CreatedAt: timestamp(s.now()), Delivery: d, Effect: "pending"}
	if err := s.commit(Event{Type: "approval.requested", Approval: &a}); err != nil {
		return Approval{}, err
	}
	return clone(a), nil
}

func (s *Service) ClaimDelivery(id, hostHome string) (Approval, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.approvals[id]
	if !ok || a.Delivery == nil {
		return Approval{}, ErrNotFound
	}
	d := clone(*a.Delivery)
	t := s.tasks[a.TaskID]
	if a.Status != "approved" || d.Status != "ready" || d.HostHome != hostHome || t.Execution == nil || t.Execution.RequestID != d.RequestID || (t.State != "verifying" && t.State != "delivered") {
		return Approval{}, fmt.Errorf("%w: delivery is not authorized for dispatch", ErrConflict)
	}
	d.Status, d.Lease = "sending", rand.Text()
	if err := s.commit(Event{Type: "delivery.updated", ApprovalID: id, Delivery: &d}); err != nil {
		return Approval{}, err
	}
	return clone(s.approvals[id]), nil
}

func (s *Service) FinishDelivery(id, lease, status, message string, messageIDs []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.approvals[id]
	if !ok || a.Delivery == nil {
		return ErrNotFound
	}
	d := clone(*a.Delivery)
	if lease == "" || d.Lease != lease || (status != "sent" && status != "failed" && status != "unknown") {
		return ErrConflict
	}
	if d.Status != "sending" && d.Status != "unknown" {
		return ErrConflict
	}
	if status == "sent" && len(messageIDs) == 0 {
		return fmt.Errorf("sent delivery requires a provider message receipt")
	}
	d.Status, d.Error, d.MessageIDs = status, message, append([]string(nil), messageIDs...)
	return s.commit(Event{Type: "delivery.updated", ApprovalID: id, Delivery: &d})
}

// On reconnect, an in-flight send is inspected by the user rather than replayed.
func (s *Service) recoverDeliveries(hostHome string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var events []Event
	for _, a := range s.approvals {
		if a.Delivery == nil || a.Delivery.HostHome != hostHome || a.Delivery.Status != "sending" {
			continue
		}
		d := clone(*a.Delivery)
		d.Status = "unknown"
		d.Error = "发送时连接中断，请先核对飞书会话。"
		events = append(events, Event{Type: "delivery.updated", ApprovalID: a.ID, Delivery: &d})
	}
	return s.commit(events...)
}

func (s *Service) settleDeliveryTransport(id string, err error) error {
	s.mu.Lock()
	a := s.approvals[id]
	if a.Delivery != nil && a.Delivery.Status == "ready" {
		d := clone(*a.Delivery)
		d.Status = "failed"
		d.Error = err.Error()
		saveErr := s.commit(Event{Type: "delivery.updated", ApprovalID: id, Delivery: &d})
		s.mu.Unlock()
		if saveErr != nil {
			return saveErr
		}
		return err
	}
	s.mu.Unlock()
	if a.Delivery == nil || a.Delivery.Status != "sending" {
		return err
	}
	return s.FinishDelivery(id, a.Delivery.Lease, "unknown", "未收到发送回执，请先核对飞书会话。", nil)
}

func (h Host) deliveryRPC() *pluginrpc.Client {
	return &pluginrpc.Client{BaseURL: h.Client.Endpoint.Origin, Cookie: h.Client.Endpoint.Cookie, HTTPClient: h.Client.HTTP}
}
func (h Host) DeliveryRoutes(ctx context.Context) ([]DeliveryRoute, error) {
	var out []DeliveryRoute
	err := h.deliveryRPC().Call(ctx, "native-proactive", "delivery/routes", map[string]any{}, &out)
	return out, err
}
func (h Host) deliver(ctx context.Context, id string) error {
	var out struct {
		Status string `json:"status"`
	}
	return h.deliveryRPC().Call(ctx, "native-proactive", "delivery/send", map[string]string{"approvalId": id}, &out)
}

func (r *Runner) DeliveryRoutes(ctx context.Context) ([]DeliveryRoute, error) {
	return r.host.DeliveryRoutes(ctx)
}
func (r *Runner) ProposeDelivery(ctx context.Context, taskID, routeID, text string) (Approval, error) {
	routes, err := r.host.DeliveryRoutes(ctx)
	if err != nil {
		return Approval{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.ctx.Err() != nil {
		return Approval{}, r.ctx.Err()
	}
	if r.frozen > 0 {
		return Approval{}, fmt.Errorf("desktop lifecycle operation is in progress")
	}
	for _, route := range routes {
		if route.ID == routeID {
			return r.service.RequestDelivery(taskID, r.home, route, text)
		}
	}
	return Approval{}, fmt.Errorf("所选飞书会话已不可用，请刷新后再选择。")
}

func (r *Runner) dispatchDeliveries() {
	for _, a := range r.service.Snapshot().Outbox {
		if a.Status != "approved" || a.Delivery == nil || a.Delivery.Status != "ready" || a.Delivery.HostHome != r.home {
			continue
		}
		r.mu.Lock()
		if r.frozen > 0 || r.ctx.Err() != nil {
			r.mu.Unlock()
			return
		}
		r.deliveryRunning = true
		r.mu.Unlock()
		err := r.host.deliver(r.ctx, a.ID)
		if err != nil {
			err = r.service.settleDeliveryTransport(a.ID, err)
		}
		r.mu.Lock()
		r.deliveryRunning = false
		r.mu.Unlock()
		r.changed(err)
	}
}

func (r *Runner) DeliveryRunning() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.deliveryRunning
}
