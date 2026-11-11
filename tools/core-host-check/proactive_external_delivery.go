package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/desktop/proactive"
	"whaleisle/internal/dsh"
)

//go:embed proactive_delivery_fixture.mjs
var proactiveDeliveryFixture string

func prepareProactiveDeliveryFixture(root, vendor, workspace string) (string, error) {
	if err := os.Setenv("WHALEISLE_QA_VENDOR", vendor); err != nil {
		return "", err
	}
	if err := os.Setenv("WHALEISLE_QA_WORKSPACE", workspace); err != nil {
		return "", err
	}
	module := filepath.Join(root, "qa-feishu-channel.mjs")
	if err := os.WriteFile(module, []byte(proactiveDeliveryFixture), 0600); err != nil {
		return "", err
	}
	entry, _ := json.Marshal(fixtureFileURL(module))
	patch := filepath.Join(root, "qa-feishu-channel.patch.yml")
	return patch, os.WriteFile(patch, []byte("- insert:\n    - id: qa-proactive-channel\n      name: "+string(entry)+"\n"), 0600)
}

func exerciseProactiveExternalDelivery(ctx context.Context, client *dsh.Client, workspace, home string, pass func(string, string), open func() (*proactive.Service, error), restart func() (*dsh.Client, error)) error {
	s, err := open()
	if err != nil {
		return err
	}
	runner := proactive.NewRunner(ctx, s, proactive.Host{Client: client}, home, nil)
	defer func() { runner.Close() }()
	rpc := &pluginrpc.Client{BaseURL: client.Endpoint.Origin, Cookie: client.Endpoint.Cookie, HTTPClient: client.HTTP}
	type effect struct {
		Data struct {
			ReceiveID string `json:"receive_id"`
			Content   string `json:"content"`
			UUID      string `json:"uuid"`
		} `json:"data"`
	}
	type fixtureState struct {
		Ready   bool     `json:"ready"`
		Effects []effect `json:"effects"`
	}
	fixture := func(method string, payload any) (fixtureState, error) {
		var out fixtureState
		err := rpc.Call(ctx, "qa-proactive-channel", method, payload, &out)
		return out, err
	}
	if err := await(ctx, func() (bool, error) { v, e := fixture("status", map[string]any{}); return v.Ready, e }); err != nil {
		return err
	}
	task, err := s.Create(proactive.CreateInput{Title: "QA_HELLO: prepare a task result", Autonomy: 2})
	if err != nil {
		return err
	}
	task, err = runner.Start(task.ID, proactive.RunSpec{HostHome: home, Cwd: workspace}, false)
	if err != nil {
		return err
	}
	if err := await(ctx, func() (bool, error) {
		v, e := s.Get(task.ID)
		if e != nil {
			return false, e
		}
		if v.Execution.Phase == "unknown" {
			return false, fmt.Errorf("task: %s", v.Execution.Error)
		}
		return v.State == "verifying", nil
	}); err != nil {
		return err
	}
	routes, err := runner.DeliveryRoutes(ctx)
	if err != nil || len(routes) != 1 {
		return fmt.Errorf("delivery routes: %v %v", routes, err)
	}
	propose := func(text string) (proactive.Approval, error) {
		return runner.ProposeDelivery(ctx, task.ID, routes[0].ID, text)
	}
	readApproval := func(id string) proactive.Approval {
		for _, a := range s.Snapshot().Outbox {
			if a.ID == id {
				return a
			}
		}
		return proactive.Approval{}
	}
	waitStatus := func(id, want string) error {
		return await(ctx, func() (bool, error) {
			a := readApproval(id)
			if a.Delivery == nil {
				return false, fmt.Errorf("missing delivery")
			}
			if a.Delivery.Status == "failed" && want != "failed" {
				return false, fmt.Errorf("delivery: %s", a.Delivery.Error)
			}
			return a.Delivery.Status == want, nil
		})
	}
	a, err := propose("Reject this proposed report")
	if err != nil {
		return err
	}
	before, err := fixture("status", map[string]any{})
	if err != nil || len(before.Effects) != 0 {
		return fmt.Errorf("effect before approval: %v", err)
	}
	if _, err := s.Decide(a.ID, "rejected", ""); err != nil {
		return err
	}
	if _, err := s.ClaimDelivery(a.ID, home); err == nil {
		return fmt.Errorf("rejected delivery was claimable")
	}
	a, err = propose("批准的核对结果\nOnly this exact body")
	if err != nil {
		return err
	}
	if _, err := s.Decide(a.ID, "approved", ""); err != nil {
		return err
	}
	if err := waitStatus(a.ID, "sent"); err != nil {
		return err
	}
	sent, err := fixture("status", map[string]any{})
	if err != nil || len(sent.Effects) != 1 {
		return fmt.Errorf("SDK effect count: %v %v", sent, err)
	}
	var message struct {
		Text string `json:"text"`
	}
	_ = json.Unmarshal([]byte(sent.Effects[0].Data.Content), &message)
	if sent.Effects[0].Data.ReceiveID != "qa-chat" || message.Text != a.Delivery.Text || sent.Effects[0].Data.UUID == "" || len(readApproval(a.ID).Delivery.MessageIDs) != 1 {
		return fmt.Errorf("exact destination/body/receipt mismatch")
	}
	var duplicate any
	if err := rpc.Call(ctx, "native-proactive", "delivery/send", map[string]string{"approvalId": a.ID}, &duplicate); err == nil {
		return fmt.Errorf("repeated ID dispatched")
	}
	pass("proactive-approved-feishu-adapter", "real Host and existing Feishu channel adapter sent the exact approved destination/body once into an SDK recorder and persisted its message receipt; rejected proposal sent nothing")
	a, err = propose("Must not send after owner revocation")
	if err != nil {
		return err
	}
	if _, err := fixture("revoke", map[string]any{}); err != nil {
		return err
	}
	if _, err := s.Decide(a.ID, "approved", ""); err != nil {
		return err
	}
	if err := waitStatus(a.ID, "failed"); err != nil {
		return err
	}
	blocked, err := fixture("status", map[string]any{})
	if err != nil || len(blocked.Effects) != 1 {
		return fmt.Errorf("revoked delivery dispatched: %v", err)
	}
	pass("proactive-delivery-revoked-owner", "revoking the original Feishu task owner after preview prevented the approved send before SDK dispatch")
	if _, err := fixture("restore", map[string]any{}); err != nil {
		return err
	}
	if _, err := fixture("mode", map[string]string{"mode": "unknown"}); err != nil {
		return err
	}
	a, err = propose("Unknown receipt must not be retried")
	if err != nil {
		return err
	}
	if _, err := s.Decide(a.ID, "approved", ""); err != nil {
		return err
	}
	if err := waitStatus(a.ID, "unknown"); err != nil {
		return err
	}
	runner.Close()
	client, err = restart()
	if err != nil {
		return err
	}
	s, err = open()
	if err != nil {
		return err
	}
	runner = proactive.NewRunner(ctx, s, proactive.Host{Client: client}, home, nil)
	rpc = &pluginrpc.Client{BaseURL: client.Endpoint.Origin, Cookie: client.Endpoint.Cookie, HTTPClient: client.HTTP}
	if err := await(ctx, func() (bool, error) { v, e := fixture("status", map[string]any{}); return v.Ready, e }); err != nil {
		return err
	}
	if readApproval(a.ID).Delivery.Status != "unknown" {
		return fmt.Errorf("restart hid unknown receipt")
	}
	if err := rpc.Call(ctx, "native-proactive", "delivery/send", map[string]string{"approvalId": a.ID}, &duplicate); err == nil {
		return fmt.Errorf("unknown delivery replayed")
	}
	restored, err := fixture("status", map[string]any{})
	if err != nil || len(restored.Effects) != 2 {
		return fmt.Errorf("restart effect count: %v %v", restored, err)
	}
	pass("proactive-delivery-unknown-no-replay", "missing provider message ID persisted unknown across actual Host restart and duplicate send could not replay the effect")
	return nil
}
