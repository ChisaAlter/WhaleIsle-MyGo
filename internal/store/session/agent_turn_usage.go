package session

import (
	"encoding/json"
	"whaleisle/internal/projectflow"
)

// AgentTurnUsage reuses the exact durable-attempt accounting for the addressed
// page. Incomplete pages keep missing accounting absent rather than invent zero.
func AgentTurnUsage(records []projectflow.ProcessRecord, messages []Message) map[float64]TurnUsage {
	store := Store{messages: messages, trajectoryBoundaries: map[float64]json.RawMessage{}}
	for _, record := range records {
		if record.Event.Seq > 0 {
			raw, err := json.Marshal(record.Event)
			if err == nil {
				store.trajectoryBoundaries[record.Event.Seq] = raw
			}
		}
	}
	return store.turnUsageLocked()
}
