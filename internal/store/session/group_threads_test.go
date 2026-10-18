package session

import (
	"encoding/json"
	"strings"
	"testing"
)

func groupFixture(raw string) Message {
	var h struct{ Seq float64 }
	json.Unmarshal([]byte(raw), &h)
	return Message{Raw: json.RawMessage(raw), Seq: h.Seq}
}
func TestGroupThreadReplyProtocol(t *testing.T) {
	id, text, ok, err := ParseGroupThreadReply("/dshbot-thread-reply thread:1  hello\n@everyone")
	if !ok || err != nil || id != "thread:1" || text != "hello\n@everyone" {
		t.Fatal(id, text, ok, err)
	}
	encoded, err := EncodeGroupThreadReply(id, text)
	if err != nil || encoded != "[DSHBOT_THREAD_REPLY:thread:1]\nhello\n@everyone" {
		t.Fatal(encoded, err)
	}
	if id, body := ParseGroupThreadEnvelope(encoded); id != "thread:1" || body != text {
		t.Fatal(id, body)
	}
	if _, _, ok, _ := ParseGroupThreadReply("/dshbot-thread-replying x"); ok {
		t.Fatal("claimed unrelated command")
	}
	for _, id := range []string{"", "x]\n", strings.Repeat("a", 129)} {
		if _, err := EncodeGroupThreadReply(id, ""); err == nil {
			t.Fatalf("accepted %q", id)
		}
	}
	target := (&ComposerTarget{mode: PromptModeSteer}).GroupReplyTarget()
	if target.mode != PromptModeQueue {
		t.Fatal(target.mode)
	}
}
func TestGroupThreadProjection(t *testing.T) {
	members := []GroupMember{{ID: "bot1", Name: "stable", Title: "Display"}}
	events := []Message{
		groupFixture(`{"seq":1,"time":100,"type":"user/message","data":{"source":{"kind":"user"},"content":[{"type":"text","text":"[DSHBOT_THREAD_REPLY:t1]\nquestion"}]}}`),
		groupFixture(`{"seq":2,"type":"tool/call","data":{"name":"ask_participant","callId":"c1","arguments":"{\"botId\":\"stable\",\"threadId\":\"t1\"}"}}`),
		groupFixture(`{"seq":3,"type":"assistant/message","data":{"message":{"content":[{"type":"text","text":"coordinator internal"}]}}}`),
		groupFixture(`{"seq":4,"time":200,"type":"tool/result","data":{"message":{"source":{"callId":"c1"},"content":[{"type":"tool-result","content":[{"type":"text","text":"answer\nNEXT: PASS"}]}]}}}`),
		groupFixture(`{"seq":5,"time":300,"type":"tool/result","data":{"threadId":"other","message":{"source":{"callId":"c1"},"content":[{"type":"text","text":"wrong thread"}]}}}`),
	}
	got := ProjectGroupThreads(events, members)
	if len(got) != 1 || got[0].ID != "t1" || len(got[0].Messages) != 2 {
		t.Fatalf("%+v", got)
	}
	reply := got[0].Messages[1]
	if reply.Text != "answer" || reply.BotID != "bot1" || reply.Speaker != "Display" {
		t.Fatal(reply)
	}
}
func TestGroupLateCarrierAndMentions(t *testing.T) {
	raw := `{"seq":1,"type":"agent/inbox/spliced","data":{"inserted":[{"source":{"kind":"plugin","plugin":"dshbot","form":"notice","dshbotGroupLateResult":{"carrierId":"same","threadId":"t1","botId":"b","callId":"c","texts":["late","PASS","ignored third"]}}}]}}`
	events := []Message{groupFixture(raw), groupFixture(raw)}
	got := ProjectGroupThreads(events, nil)
	if len(got) != 1 || len(got[0].Messages) != 2 || !got[0].Messages[0].Late {
		t.Fatal(got)
	}
	mentions := GroupMentions([]GroupMember{{ID: "b", Name: "stable", Title: "Display"}}, "display")
	if len(mentions) != 1 || mentions[0].Value != "stable" {
		t.Fatal(mentions)
	}
	if m := GroupMentions(nil, "all"); len(m) != 1 || m[0].Value != "everyone" {
		t.Fatal(m)
	}
}
func TestGroupLegacyThreadGapAndRelay(t *testing.T) {
	var msgs []Message
	for i, at := range []int{1, 2, 1000000} {
		raw := `{"seq":1,"time":TIME,"type":"user/message","data":{"source":{"kind":"plugin","plugin":"dshbot","form":"relay"},"content":[{"type":"text","text":"hello"}]}}`
		raw = strings.ReplaceAll(raw, "TIME", []string{"1", "2", "1000000"}[i])
		_ = at
		msgs = append(msgs, groupFixture(raw))
	}
	got := ProjectGroupThreads(msgs, nil)
	if len(got) != 2 || got[0].ID != "legacy-0" || len(got[0].Messages) != 2 || got[1].ID != "legacy-1" {
		t.Fatal(got)
	}
}

func TestGroupLateCarrierSurvivesClassification(t *testing.T) {
	raw := json.RawMessage(`{"seq":9,"type":"agent/inbox/spliced","data":{"inserted":[{"source":{"kind":"plugin","plugin":"dshbot","form":"notice","dshbotGroupLateResult":{"threadId":"old","botId":"b","callId":"c","texts":["late"]}}}]}}`)
	m, ok := classifyPersistenceEvent(raw)
	if !ok {
		t.Fatal("late carrier lost in follow projection")
	}
	threads := ProjectGroupThreads([]Message{m}, nil)
	if len(threads) != 1 || threads[0].ID != "old" {
		t.Fatal(threads)
	}
}
