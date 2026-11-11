package proactive

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func completedDeliveryTask(t *testing.T, s *Service, dir string) Task {
	t.Helper()
	v, err := s.Create(CreateInput{Title: "Report result", Autonomy: 2})
	if err != nil {
		t.Fatal(err)
	}
	v, err = s.PrepareExecution(v.ID, RunSpec{HostHome: "host", Cwd: dir}, false)
	if err != nil {
		t.Fatal(err)
	}
	v, err = s.updateExecution(v.ID, v.Execution.RequestID, "completed", "verifying", "Exact report", "")
	if err != nil {
		t.Fatal(err)
	}
	return v
}
func deliveryTestRoute() DeliveryRoute {
	return DeliveryRoute{ID: "source-task", Title: "Review group", Channel: "feishu", BotID: "bot", Actor: "owner", ChatID: "chat-A", Incarnation: "instance-A"}
}

func TestDeliveryDecisionLeaseAndRestart(t *testing.T) {
	dir := t.TempDir()
	s := openService(t, dir)
	task := completedDeliveryTask(t, s, dir)
	a, err := s.RequestDelivery(task.ID, "host", deliveryTestRoute(), "Exact report\nOnly this text")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.ClaimDelivery(a.ID, "host"); !errors.Is(err, ErrConflict) {
		t.Fatal("unapproved send", err)
	}
	if _, err = s.Decide(a.ID, "rejected", ""); err != nil {
		t.Fatal(err)
	}
	current, _ := s.Get(task.ID)
	if current.State != "verifying" {
		t.Fatal("rejecting delivery failed the completed task", current)
	}
	a, err = s.RequestDelivery(task.ID, "host", deliveryTestRoute(), "Approved exact body")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Decide(a.ID, "approved", ""); err != nil {
		t.Fatal(err)
	}
	if _, err = s.PrepareExecution(task.ID, task.Execution.Spec, true); !errors.Is(err, ErrConflict) {
		t.Fatal("continuation invalidated queued delivery", err)
	}
	s = openService(t, dir)
	if _, err = s.ClaimDelivery(a.ID, "other-host"); !errors.Is(err, ErrConflict) {
		t.Fatal("foreign host claim", err)
	}
	claimed, err := s.ClaimDelivery(a.ID, "host")
	if err != nil {
		t.Fatal(err)
	}
	if claimed.Delivery.Text != "Approved exact body" || claimed.Delivery.Route.ChatID != "chat-A" {
		t.Fatal(claimed)
	}
	if _, err = s.ClaimDelivery(a.ID, "host"); !errors.Is(err, ErrConflict) {
		t.Fatal("duplicate dispatch", err)
	}
	if _, err = s.Transition(TransitionInput{TaskID: task.ID, State: "cancelled"}); !errors.Is(err, ErrConflict) {
		t.Fatal("in-flight cancelled as unsent", err)
	}
	s = openService(t, dir)
	if err = s.recoverDeliveries("host"); err != nil {
		t.Fatal(err)
	}
	if s.Snapshot().Outbox[1].Delivery.Status != "unknown" {
		t.Fatal(s.Snapshot().Outbox)
	}
	if _, err = s.ClaimDelivery(a.ID, "host"); !errors.Is(err, ErrConflict) {
		t.Fatal("restart dispatched twice", err)
	}
	if err = s.FinishDelivery(a.ID, claimed.Delivery.Lease, "sent", "", []string{"message-A"}); err != nil {
		t.Fatal(err)
	}
	s = openService(t, dir)
	if s.Snapshot().Outbox[1].Delivery.MessageIDs[0] != "message-A" {
		t.Fatal("receipt lost")
	}
	if _, err = s.Transition(TransitionInput{TaskID: task.ID, State: "delivered"}); err != nil {
		t.Fatal(err)
	}
	a, err = s.RequestDelivery(task.ID, "host", deliveryTestRoute(), "Another explicitly reviewed message")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Decide(a.ID, "rejected", ""); err != nil {
		t.Fatal(err)
	}
	current, _ = s.Get(task.ID)
	if current.State != "delivered" {
		t.Fatal("delivery changed accepted result")
	}
}

func TestDeliveryHostCannotForgeAndTransportFailureDoesNotRetry(t *testing.T) {
	dir := t.TempDir()
	s := openService(t, dir)
	task := completedDeliveryTask(t, s, dir)
	raw, _ := json.Marshal(map[string]any{"taskId": task.ID, "action": "external_message", "delivery": map[string]any{"status": "ready"}})
	value, err := s.Dispatch(context.Background(), "approval/request", raw)
	if err != nil {
		t.Fatal(err)
	}
	forged := value.(Approval)
	if forged.Delivery != nil {
		t.Fatal("Host forged a native delivery")
	}
	if _, err = s.ClaimDelivery(forged.ID, "host"); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	if _, err = s.Dispatch(context.Background(), "delivery/request", raw); err == nil {
		t.Fatal("Host admitted a native delivery")
	}
	// Use a separate result because rejection of a non-delivery proposal retains
	// its original task-failure semantics.
	task = completedDeliveryTask(t, s, dir)
	a, err := s.RequestDelivery(task.ID, "host", deliveryTestRoute(), "report")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Decide(a.ID, "approved", ""); err != nil {
		t.Fatal(err)
	}
	if err = s.settleDeliveryTransport(a.ID, errors.New("offline")); err == nil {
		t.Fatal("missing offline failure")
	}
	if _, err = s.ClaimDelivery(a.ID, "host"); !errors.Is(err, ErrConflict) {
		t.Fatal("failed transport was retried", err)
	}
	a, err = s.RequestDelivery(task.ID, "host", deliveryTestRoute(), "revised report")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Transition(TransitionInput{TaskID: task.ID, State: "cancelled"}); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Decide(a.ID, "approved", ""); !errors.Is(err, ErrConflict) {
		t.Fatal("cancelled task revived delivery", err)
	}
}
