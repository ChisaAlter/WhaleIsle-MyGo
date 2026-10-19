package jobflow

import (
	"encoding/json"
	"errors"
	"testing"
	"testing/synctest"
	"time"
)

func TestPopupReopenKeepsSelectionAndStartsFreshObservation(t *testing.T) {
	c, f, generation := fixture()
	defer c.Bind(nil, "")
	rows := nextOpen(t, f)
	rows.stream.frames <- json.RawMessage(`{"type":"rows","jobs":[{"id":"owned","owner":"session-a","status":"running","output":{"total":4}}]}`)
	waitState(t, c, func(s Snapshot) bool { return len(s.Rows) == 1 })
	c.Toggle(generation, "owned")
	first := nextOpen(t, f)
	c.Close(generation)
	closed := c.Snapshot()
	if closed.Open || closed.SelectedID != "owned" || closed.Observed.Text != "" {
		t.Fatalf("closing did not release output while retaining disclosure: %+v", closed)
	}
	c.Open(closed.Generation)
	again := nextOpen(t, f)
	if again.method != "job/follow" || again.request["jobId"] != "owned" || again.request["sessionId"] != "session-a" {
		t.Fatalf("wrong resumed observation: %+v", again)
	}
	if _, resumedCursor := again.request["from"]; resumedCursor {
		t.Fatal("reopened output reused an unmounted observation cursor")
	}
	first.stream.frames <- json.RawMessage(`{"type":"output","chunks":[{"at":0,"text":"retired"}],"next":7}`)
	again.stream.frames <- json.RawMessage(`{"type":"opened","job":{"id":"owned","status":"running"},"from":0}`)
	again.stream.frames <- json.RawMessage(`{"type":"output","chunks":[{"at":0,"text":"current"}],"next":7}`)
	waitState(t, c, func(s Snapshot) bool { return s.Observed.Text == "current" })
}

func TestKillFailureExpiresWithoutRetiringNewOwnerNotice(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c, f, generation := fixture()
		defer c.Bind(nil, "")
		f.killErr = errors.New("owned failure")
		rows := nextOpen(t, f)
		rows.stream.frames <- json.RawMessage(`{"type":"rows","jobs":[{"id":"owned","status":"running"}]}`)
		synctest.Wait()
		c.ArmKill(generation, "owned")
		c.ConfirmKill(generation, "owned")
		synctest.Wait()
		if c.Snapshot().Kill.Phase != "failed" {
			t.Fatal("missing failed notice")
		}
		time.Sleep(4 * time.Second)
		synctest.Wait()
		if c.Snapshot().Kill.Phase != "" {
			t.Fatal("failure notice outlived four seconds")
		}
		c.ArmKill(generation, "owned")
		c.ConfirmKill(generation, "owned")
		synctest.Wait()
		time.Sleep(2 * time.Second)
		c.Bind(f, "session-b")
		generation = c.Snapshot().Generation
		c.Open(generation)
		rows = nextOpen(t, f)
		rows.stream.frames <- json.RawMessage(`{"type":"rows","jobs":[{"id":"owned","status":"running"}]}`)
		synctest.Wait()
		c.ArmKill(generation, "owned")
		c.ConfirmKill(generation, "owned")
		synctest.Wait()
		time.Sleep(2 * time.Second)
		synctest.Wait()
		if c.Snapshot().Kill.Phase != "failed" {
			t.Fatal("old owner timer cleared the replacement failure")
		}
	})
}
