package pluginrpc

import "fmt"

// RPCError is a failure in an identity-checked Host response. Whether a write
// was committed depends on the endpoint's error contract, not this type alone.
// Transport and decode failures are not RPCError values.
type RPCError struct{ Code, Message string }

func (e *RPCError) Error() string { return fmt.Sprintf("pluginrpc: %s: %s", e.Code, e.Message) }
