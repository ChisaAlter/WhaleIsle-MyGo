package chat

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/store/session"
)

func mentionLRUCountedBuilder(calls *int) func([]session.Message, session.Message) func(string) string {
	return func(messages []session.Message, closing session.Message) func(string) string {
		*calls++
		return buildMentionResolver(messages, closing)
	}
}

func mentionLRUAssertBounds(t *testing.T, memo *mentionMemoStore) {
	t.Helper()
	memo.mu.Lock()
	defer memo.mu.Unlock()
	bytes := 0
	for _, raw := range memo.source {
		bytes += len(raw)
	}
	if bytes != memo.sourceBytes {
		t.Fatalf("journal bytes=%d, counter=%d", bytes, memo.sourceBytes)
	}
	for _, entry := range memo.entries {
		bytes += len(entry.key.raw)
	}
	if bytes != memo.rawBytes || bytes > mentionMemoByteLimit || len(memo.entries) > mentionMemoEntryLimit {
		t.Fatalf("retained raw=%d counter=%d entries=%d", bytes, memo.rawBytes, len(memo.entries))
	}
}

func TestMentionMemoLRUKeepsSixteenWarmClosings(t *testing.T) {
	messages, allClosings := mentionLRUHistory(20)
	closings := allClosings[len(allClosings)-16:]
	memo, builds := mentionMemoStore{}, 0
	build := mentionLRUCountedBuilder(&builds)
	for _, closing := range closings {
		if memo.resolver(messages, closing, build) == nil {
			t.Fatal("fixture has no delivered path")
		}
	}
	generation, source := memo.generation, &memo.source[0]
	for range 12 {
		for _, closing := range closings {
			memo.resolver(messages, closing, build)
		}
	}
	if builds != 16 || len(memo.entries) != 16 || memo.generation != generation || &memo.source[0] != source {
		t.Fatalf("warm closings rebuilt or recopied journal: builds=%d entries=%d generation=%d", builds, len(memo.entries), memo.generation)
	}
	mentionLRUAssertBounds(t, &memo)
}

func TestMentionMemoLRUEvictsOnlyLeastRecentlyUsed(t *testing.T) {
	messages, closings := mentionLRUHistory(17)
	memo, builds := mentionMemoStore{}, 0
	build := mentionLRUCountedBuilder(&builds)
	for _, closing := range closings[:16] {
		memo.resolver(messages, closing, build)
	}
	memo.resolver(messages, closings[0], build) // Refresh the oldest entry.
	memo.resolver(messages, closings[16], build)
	if memo.findClosing(closings[1]) >= 0 || memo.findClosing(closings[0]) < 0 || memo.findClosing(closings[16]) < 0 || len(memo.entries) != 16 {
		t.Fatal("17th closing did not evict only the least recently used entry")
	}
	for _, closing := range append([]session.Message{closings[0]}, closings[2:]...) {
		memo.resolver(messages, closing, build)
	}
	if builds != 17 {
		t.Fatalf("a surviving LRU entry rebuilt: builds=%d", builds)
	}
	mentionLRUAssertBounds(t, &memo)
}

func TestMentionMemoLRUPreservesPreciseClosingIdentityAndNil(t *testing.T) {
	messages, closings := mentionLRUHistory(2)
	memo, builds := mentionMemoStore{}, 0
	build := mentionLRUCountedBuilder(&builds)
	closing := closings[0]
	first := memo.resolver(messages, closing, build)
	// Same IDs and same-length bytes do not own the same turn's resolver.
	other := closing
	other.Raw = append(json.RawMessage(nil), closing.Raw...)
	copy(other.Raw, strings.Replace(string(other.Raw), `"turn":0`, `"turn":1`, 1))
	other.Raw = json.RawMessage(strings.Replace(string(other.Raw), `"seq":3`, `"seq":7`, 1))
	second := memo.resolver(messages, other, build)
	if first == nil || second == nil || first("report-000.txt") != "/synthetic/report-000.txt" || second("report-001.txt") != "/synthetic/report-001.txt" {
		t.Fatal("different closing raw reused another turn or sequence boundary")
	}
	for _, state := range []struct {
		role      string
		streaming bool
	}{{"user", false}, {"assistant", true}} {
		nilClosing := closing
		nilClosing.Role, nilClosing.Streaming = state.role, state.streaming
		if memo.resolver(messages, nilClosing, build) != nil || memo.resolver(messages, nilClosing, build) != nil {
			t.Fatal("ineligible closing reused completed resolver")
		}
	}
	if builds != 4 {
		t.Fatalf("nil results were not cached: builds=%d", builds)
	}
	memo.resolver(messages, closing, build)
	if builds != 4 || first("report-000.txt") != "/synthetic/report-000.txt" {
		t.Fatal("different closing entries mutated an already returned resolver")
	}
}

func TestMentionMemoLRUSourceMutationInvalidatesAllEntries(t *testing.T) {
	for _, inPlace := range []bool{false, true} {
		t.Run(fmt.Sprint("inPlace=", inPlace), func(t *testing.T) {
			messages, closings := mentionLRUHistory(16)
			memo, builds := mentionMemoStore{}, 0
			build := mentionLRUCountedBuilder(&builds)
			var first func(string) string
			for _, closing := range closings {
				resolve := memo.resolver(messages, closing, build)
				if first == nil {
					first = resolve
				}
			}
			generation := memo.generation
			replacement := strings.Replace(string(messages[1].Raw), "/synthetic/report-000.txt", "/synthetic/report-999.txt", 1)
			if inPlace {
				copy(messages[1].Raw, replacement)
			} else {
				messages[1].Raw = json.RawMessage(replacement)
			}
			resolve := memo.resolver(messages, closings[0], build)
			if memo.generation != generation+1 || len(memo.entries) != 1 || resolve("report-999.txt") != "/synthetic/report-999.txt" || resolve("report-000.txt") != "" {
				t.Fatal("exact journal replacement did not invalidate all sixteen old entries")
			}
			for _, closing := range closings[1:] {
				memo.resolver(messages, closing, build)
			}
			if builds != 32 || first("report-000.txt") != "/synthetic/report-000.txt" {
				t.Fatalf("rebuild count or caller-retained resolver changed: builds=%d", builds)
			}
			mentionLRUAssertBounds(t, &memo)
		})
	}
}

func TestMentionMemoLRURawBudgetAndOversizedBypass(t *testing.T) {
	memo, builds := mentionMemoStore{}, 0
	build := func([]session.Message, session.Message) func(string) string { builds++; return nil }
	closing := session.Message{Role: "assistant", Raw: json.RawMessage(`{}`)}
	messages := []session.Message{{Raw: json.RawMessage(strings.Repeat(" ", mentionMemoByteLimit-2))}}
	memo.resolver(messages, closing, build)
	memo.resolver(messages, closing, build)
	if builds != 1 || memo.rawBytes != mentionMemoByteLimit {
		t.Fatal("exact combined raw budget was not retained")
	}
	oversized := session.Message{Role: "assistant", Raw: json.RawMessage(strings.Repeat(" ", mentionMemoByteLimit+1))}
	generation, source := memo.generation, &memo.source[0]
	memo.resolver(messages, oversized, build)
	largerSource := []session.Message{{Raw: append(append(json.RawMessage(nil), messages[0].Raw...), ' ')}}
	memo.resolver(largerSource, closing, build)
	if memo.generation != generation || &memo.source[0] != source || memo.findClosing(closing) < 0 || builds != 3 {
		t.Fatal("oversized closing/source pair evicted the useful warm journal")
	}
	memo.resolver(messages, closing, build)
	if builds != 3 {
		t.Fatal("oversized bypass made the previous hot entry cold")
	}
	// Within one journal, exceeding bytes evicts only the oldest closings.
	memo = mentionMemoStore{}
	messages = []session.Message{{Raw: json.RawMessage(strings.Repeat(" ", mentionMemoByteLimit-8))}}
	for _, raw := range []string{"1111", "2222", "3333"} {
		memo.resolver(messages, session.Message{Raw: json.RawMessage(raw)}, build)
	}
	if len(memo.entries) != 2 || memo.entries[0].key.raw != "3333" || memo.entries[1].key.raw != "2222" || memo.rawBytes != mentionMemoByteLimit {
		t.Fatal("combined closing raw budget did not evict only the least recent entry")
	}
	mentionLRUAssertBounds(t, &memo)
}

func TestMentionMemoLRUBuildOutsideLockRejectsStaleWriteback(t *testing.T) {
	messagesA, closingsA := mentionLRUHistory(1)
	messagesB := append([]session.Message(nil), messagesA...)
	messagesB[1].Raw = json.RawMessage(strings.Replace(string(messagesA[1].Raw), "report-000.txt", "report-999.txt", 1))
	var memo mentionMemoStore
	entered, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
	oldDone := make(chan func(string) string, 1)
	go func() {
		oldDone <- memo.resolver(messagesA, closingsA[0], func(messages []session.Message, closing session.Message) func(string) string {
			close(entered)
			<-release
			return buildMentionResolver(messages, closing)
		})
	}()
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("old builder did not start")
	}
	currentDone := make(chan func(string) string, 1)
	go func() { currentDone <- memo.resolver(messagesB, closingsA[0], buildMentionResolver) }()
	var current func(string) string
	select {
	case current = <-currentDone:
	case <-time.After(3 * time.Second):
		t.Fatal("new journal was blocked behind old JSON build")
	}
	if current == nil || current("report-999.txt") != "/synthetic/report-999.txt" {
		t.Fatal("new view resolved old source")
	}
	generation := memo.generation
	releaseOnce.Do(func() { close(release) })
	var old func(string) string
	select {
	case old = <-oldDone:
	case <-time.After(3 * time.Second):
		t.Fatal("old builder did not finish")
	}
	if old == nil || old("report-000.txt") != "/synthetic/report-000.txt" {
		t.Fatal("old caller did not receive its own source result")
	}
	if memo.generation != generation || !sameRawSources(memo.source, messagesB) || len(memo.entries) != 1 {
		t.Fatal("slow old build overwrote the current journal cache")
	}
	memo.resolver(messagesB, closingsA[0], func([]session.Message, session.Message) func(string) string {
		t.Fatal("new current resolver was not cached")
		return nil
	})
}

func TestMentionMemoLRUConcurrentIndependentInputs(t *testing.T) {
	messages, closings := mentionLRUHistory(16)
	var memo mentionMemoStore
	var builds atomic.Int64
	build := func(messages []session.Message, closing session.Message) func(string) string {
		builds.Add(1)
		return buildMentionResolver(messages, closing)
	}
	for _, closing := range closings {
		memo.resolver(messages, closing, build)
	}
	before := builds.Load()
	var workers sync.WaitGroup
	for index := 0; index < 16; index++ {
		workers.Add(1)
		go func(index int) {
			defer workers.Done()
			for range 8 {
				resolve := memo.resolver(messages, closings[index], build)
				token := fmt.Sprintf("report-%03d.txt", index)
				if resolve == nil || resolve(token) != "/synthetic/"+token {
					t.Errorf("concurrent closing %d got a foreign resolver", index)
					return
				}
			}
		}(index)
	}
	workers.Wait()
	if builds.Load() != before {
		t.Fatal("warm concurrent immutable inputs rebuilt")
	}
	mentionLRUAssertBounds(t, &memo)
}
