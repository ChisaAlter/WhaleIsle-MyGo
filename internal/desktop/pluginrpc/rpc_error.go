package pluginrpc

import "fmt"

// RPCError is a definite rejection in an identity-checked Host response. A
// transport/decode failure is deliberately not this type: its write outcome
// may be unknown and must be checked rather than replayed.
type RPCError struct{ Code, Message string }

func (e *RPCError) Error() string { return fmt.Sprintf("pluginrpc: %s: %s", e.Code, e.Message) }
