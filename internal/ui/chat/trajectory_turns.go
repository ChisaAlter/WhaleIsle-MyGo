package chat

import (
	"fmt"

	"whaleisle/internal/store/session"
)

func trajectoryCoordinates(message session.Message) (turn, step float64, known bool) {
	if message.ConversationTurn != nil {
		if message.ConversationStep != nil {
			step = *message.ConversationStep
		}
		return *message.ConversationTurn, step, true
	}
	return transcriptEventCoordinates(message)
}

// Durable turn/start identity wins. Legacy messages without that boundary use
// the following assistant's turn, as the original ledger does for user input.
func trajectoryTurnGroups(messages []session.Message) ([]string, map[string]float64) {
	keys := make([]string, len(messages))
	numbers := map[string]float64{}
	byNumber := map[float64]string{}
	for _, message := range messages {
		if n, _, ok := trajectoryCoordinates(message); ok && message.TurnSeq > 0 {
			key := fmt.Sprintf("turn-seq:%g", message.TurnSeq)
			byNumber[n], numbers[key] = key, n
		}
	}
	for index, message := range messages {
		if message.TurnSeq > 0 {
			keys[index] = fmt.Sprintf("turn-seq:%g", message.TurnSeq)
		} else if number, _, ok := trajectoryCoordinates(message); ok {
			key := byNumber[number]
			if key == "" {
				key = fmt.Sprintf("turn-number:%g", number)
			}
			keys[index], numbers[key] = key, number
		}
	}
	next := ""
	for index := len(messages) - 1; index >= 0; index-- {
		if keys[index] != "" {
			next = keys[index]
			continue
		}
		if messages[index].Role == "user" || messages[index].Kind == session.MessageKindContext {
			keys[index] = next
		}
	}
	current, pendingLegacy, legacy := "", "", 0
	lastNumber := float64(0)
	for index, message := range messages {
		if keys[index] == "" {
			if message.Role == "user" || message.Kind == session.MessageKindContext {
				if pendingLegacy == "" {
					legacy++
					pendingLegacy = fmt.Sprintf("legacy-turn:%d", legacy)
					numbers[pendingLegacy] = lastNumber + 1
				}
				current = pendingLegacy
			}
			keys[index] = current
		} else {
			current = keys[index]
		}
		if number, ok := numbers[current]; ok {
			lastNumber = number
		}
		if message.Role == "assistant" || message.Kind == session.MessageKindThinking {
			pendingLegacy = ""
		}
	}
	return keys, numbers
}
