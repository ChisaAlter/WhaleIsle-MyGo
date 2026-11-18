package main

import (
	"testing"
	"time"

	"whaleisle/internal/desktop/proactive"
)

func TestProactivePanelPersistAndRetainDraftOnConflict(t *testing.T) {
	dir := t.TempDir()
	s := &mainState{}
	svc := s.assistantService()
	t.Cleanup(svc.SuspendAll)
	s.openProactive(dir)
	svc.SetProactiveDraft("检查当前候选", "保存明确的完成条件")
	svc.ProactiveModel().OnAutonomy(3)
	wait := func() {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			if !svc.ProactiveSnapshot().Busy {
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
		t.Fatal("native task write did not finish")
	}
	svc.ProactiveModel().OnCreate()
	wait()
	m := svc.ProactiveModel()
	if m.Title != "" || len(m.State.Tasks) != 1 || m.Selected != m.State.Tasks[0].ID {
		t.Fatal("native create did not consume saved draft", m)
	}
	if m.State.Tasks[0].Autonomy != 3 {
		t.Fatal("native autonomy selection not persisted")
	}
	other, err := proactive.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := other.Create(proactive.CreateInput{Title: "外部新增的任务"}); err != nil {
		t.Fatal(err)
	}
	svc.SetProactiveDraft("失败时保留的草稿", "")
	svc.ProactiveModel().OnCreate()
	wait()
	m = svc.ProactiveModel()
	if m.Error == "" || m.Title != "失败时保留的草稿" || len(m.State.Tasks) != 1 {
		t.Fatal("conflict erased draft or published unsaved state", m)
	}
	reopened, err := proactive.Open(dir)
	if err != nil || len(reopened.Snapshot().Tasks) != 2 {
		t.Fatal("conflict overwrote external work", err)
	}
}
