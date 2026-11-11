package main

import (
	"testing"
	"time"

	"whaleisle/internal/desktop/proactive"
)

func TestProactivePanelPersistAndRetainDraftOnConflict(t *testing.T) {
	dir := t.TempDir()
	s := &mainState{}
	s.openProactive(dir)
	s.proactivePanel.title, s.proactivePanel.description = "检查当前候选", "保存明确的完成条件"
	s.proactiveModel().OnAutonomy(3)
	wait := func() {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			s.mu.Lock()
			busy := s.proactivePanel.busy
			s.mu.Unlock()
			if !busy {
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
		t.Fatal("native task write did not finish")
	}
	s.proactiveModel().OnCreate()
	wait()
	m := s.proactiveModel()
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
	s.mu.Lock()
	s.proactivePanel.title = "失败时保留的草稿"
	s.mu.Unlock()
	s.proactiveModel().OnCreate()
	wait()
	m = s.proactiveModel()
	if m.Error == "" || m.Title != "失败时保留的草稿" || len(m.State.Tasks) != 1 {
		t.Fatal("conflict erased draft or published unsaved state", m)
	}
	reopened, err := proactive.Open(dir)
	if err != nil || len(reopened.Snapshot().Tasks) != 2 {
		t.Fatal("conflict overwrote external work", err)
	}
}
