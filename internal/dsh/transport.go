package dsh

import (
	"net"
	"net/http"
	"time"
)

// hostIdleConnTimeout keeps a pooled connection for less time than the Host's
// Node HTTP server keeps it open (keepAliveTimeout defaults to 5 s). A client
// that holds an idle connection longer reuses a socket the server already
// closed, and the next POST — never retried by net/http — fails with a reset.
const hostIdleConnTimeout = 4 * time.Second

// NewHostTransport returns the connection policy for one Host generation.
// Each generation owns its pool so a restarted Host never inherits sockets
// that belonged to the previous process; the owner closes it on stop.
func NewHostTransport() *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          32,
		MaxIdleConnsPerHost:   16,
		IdleConnTimeout:       hostIdleConnTimeout,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 120 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}

// NewGenerationClient builds the RPC client for one Host generation with its
// own transport. Close releases the pool when the generation stops.
func NewGenerationClient(endpoint Endpoint) *Client {
	return &Client{Endpoint: endpoint, HTTP: &http.Client{Transport: NewHostTransport()}}
}

// Close drops the idle connections this client's transport holds. It is safe
// on clients that share the package default (a no-op for the shared pool's
// active requests: only idle sockets are closed).
func (c *Client) Close() {
	if c == nil || c.HTTP == nil {
		return
	}
	c.HTTP.CloseIdleConnections()
}
