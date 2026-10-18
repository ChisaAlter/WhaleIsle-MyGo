package main

import (
	"context"
	"fmt"
	"github.com/egoist/mygo"
	"time"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/harness"
	dshbotstore "whaleisle/internal/store/dshbot"
	assistantui "whaleisle/internal/ui/assistant"
)

func (s *mainState) bindRoutineTriggerModel(m *assistantui.RoutineModel, ep harness.Endpoint, scope dshbotstore.RoutineScope, current func() bool) {
	if _, err := pluginrpc.DshbotRoutineTriggerURL(ep.Origin, m.State.SelectedID); err != nil {
		m.TriggerUnavailable = "此 Host/任务无法表示为安全触发地址；未请求凭据 / Unsupported trigger address; no credential requested"
		return
	}
	sequence := m.State.Trigger.Sequence
	validView := func() bool {
		if current() {
			return true
		}
		s.assistantRoutines.CloseTrigger(scope, sequence)
		return false
	}
	m.OnPrepareTrigger = func() {
		if validView() {
			s.assistantRoutines.PrepareTrigger(scope)
			s.updateAssistantTasks()
		}
	}
	m.OnAcknowledgeTrigger = func(ack bool) {
		if validView() {
			s.assistantRoutines.AcknowledgeTrigger(scope, sequence, ack)
			s.updateAssistantTasks()
		}
	}
	m.OnCloseTrigger = func() {
		if validView() {
			s.assistantRoutines.CloseTrigger(scope, sequence)
			s.updateAssistantTasks()
		}
	}
	m.OnConfirmTrigger = func() {
		if !validView() {
			return
		}
		ticket, ok := s.assistantRoutines.BeginTrigger(scope, sequence)
		s.updateAssistantTasks()
		if !ok {
			return
		}
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			var token []byte
			var err error
			if validView() && s.assistantRoutines.CurrentTrigger(ticket) {
				token, err = s.pluginRPCClient(ep).DshbotObtainRoutineTriggerToken(ctx)
			} else {
				err = fmt.Errorf("trigger view changed before dispatch")
			}
			defer clear(token)
			if !validView() {
				return
			}
			if s.assistantRoutines.FinishTrigger(ticket, token, err) && validView() {
				s.updateAssistantTasks()
			}
		}()
	}
	m.TriggerMaterial = func() (string, string) {
		if !validView() {
			return "", ""
		}
		address, header, err := s.assistantRoutines.TriggerMaterial(scope, sequence, ep.Origin)
		if err != nil {
			return "", ""
		}
		return address, header
	}
	m.OnCopyTrigger = func(part string) {
		if !validView() {
			return
		}
		address, header, err := s.assistantRoutines.TriggerMaterial(scope, sequence, ep.Origin)
		if err != nil {
			return
		}
		// Only an explicit user click calls clipboard. No automatic execution,
		// clipboard polling, logging, file write or background credential refresh.
		switch part {
		case "address":
			mygo.Clipboard.WriteText(address)
		case "header":
			mygo.Clipboard.WriteText(header)
		}
	}
}
