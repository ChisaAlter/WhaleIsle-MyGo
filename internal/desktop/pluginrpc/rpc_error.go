package pluginrpc

import "fmt"

// RPCError is a failure in an identity-checked Host response. Whether a write
// was committed depends on the endpoint's error contract, not this type alone.
// Transport and decode failures are not RPCError values.
type RPCError struct{ Code, Message string }

func (e *RPCError) Error() string { return fmt.Sprintf("pluginrpc: %s: %s", e.Code, e.Message) }

// HTTPError is a non-2xx HTTP response returned by a local plugin control-plane
// endpoint on the Host (for example HTTP 404 or 405 when a plugin is not mounted).
type HTTPError struct {
	StatusCode int
	Plugin     string
	Method     string
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("pluginrpc: HTTP %d: %s", e.StatusCode, e.Body)
}
