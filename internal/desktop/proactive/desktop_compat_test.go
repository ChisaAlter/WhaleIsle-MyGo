package proactive

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Opt-in against the owning desktop implementation, with an isolated journal.
// This exercises its actual replay/writer rather than a second Go fixture.
func TestDesktopProactiveJournalCompatibility(t *testing.T) {
	source, node := os.Getenv("WHALEISLE_PROACTIVE_DESKTOP_SOURCE"), os.Getenv("WHALEISLE_PROACTIVE_NODE")
	if source == "" || node == "" {
		t.Skip("set explicit desktop source and Node paths for cross-runtime journal verification")
	}
	dir := t.TempDir()
	s := openService(t, dir)
	task := createTask(t, s, "原生创建的任务")
	a, err := s.RequestApproval(ApprovalInput{TaskID: task.ID, Action: "核对指定资料"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Decide(a.ID, "approved", "批准指定范围"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Schedule(ScheduleInput{TaskID: task.ID, At: "2026-11-20T10:00:00Z"}); err != nil {
		t.Fatal(err)
	}
	script := `
const {createProactiveService} = require(process.argv[1]);
const service = createProactiveService({userDataPath: process.argv[2]});
const before = { task: service.runtime.getTask(process.argv[3]), outbox: service.runtime.listOutbox(), jobs: service.scheduler.list() };
const task = service.runtime.createTask({title:'本体创建的任务', autonomy:2, source:'github'});
const approval = service.runtime.requestApproval(task.id, {action:'准备发布报告', target:'named-recipient', risk:'medium'});
process.stdout.write(JSON.stringify({before, task, approval}));
`
	out, err := exec.Command(node, "-e", script, filepath.Join(source, "src", "main", "proactive", "service.js"), dir, task.ID).CombinedOutput()
	if err != nil {
		t.Fatalf("desktop runtime: %v\n%s", err, out)
	}
	var result struct {
		Before struct {
			Task   Task       `json:"task"`
			Outbox []Approval `json:"outbox"`
			Jobs   []Job      `json:"jobs"`
		} `json:"before"`
		Task     Task     `json:"task"`
		Approval Approval `json:"approval"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatal(err)
	}
	if result.Before.Task.State != "resumed" || len(result.Before.Outbox) != 1 || result.Before.Outbox[0].Status != "approved" || len(result.Before.Jobs) != 1 {
		t.Fatalf("desktop rejected native journal: %s", out)
	}
	s = openService(t, dir)
	restored, err := s.Get(result.Task.ID)
	if err != nil || restored.State != "waiting_approval" || restored.ApprovalID != result.Approval.ID {
		t.Fatal(restored, err)
	}
	if _, err := s.Decide(result.Approval.ID, "rejected", "暂不发布"); err != nil {
		t.Fatal(err)
	}
	if got, _ := openService(t, dir).Get(result.Task.ID); got.State != "failed" {
		t.Fatal(got)
	}
}
