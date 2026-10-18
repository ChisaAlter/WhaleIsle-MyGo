package dsh

import "context"

// NextPromptResult is the conversation plugin's completed-turn suggestion.
// Null text means the Host has no suggestion, never a locally generated prompt.
type NextPromptResult struct {
	Turn *float64 `json:"turn"`
	Seq  *float64 `json:"seq"`
	Text *string  `json:"text"`
}

// NextPrompt uses the Connection API's direct payload, not Typert's args wrapper.
func (c *Client) NextPrompt(ctx context.Context, sessionID string) (NextPromptResult, error) {
	var result NextPromptResult
	err := c.callPayload(ctx, "conversation.next-prompt", map[string]string{"sessionId": sessionID}, &result)
	return result, err
}
