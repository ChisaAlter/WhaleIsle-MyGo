package main

import "testing"

func TestAssistantGroupRuntimeShellLifecycle(t *testing.T) {
	var state mainState
	svc := state.assistantService()
	svc.BeginGroupRuntimeFrame()
	svc.FinishGroupRuntimeFrame()
	svc.StopGroupRuntime()
	svc.OnWindowHide()
	svc.OnWindowClosed()
}
