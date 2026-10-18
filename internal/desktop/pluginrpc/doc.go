// Package pluginrpc calls harness plugin HTTP control planes that speak the
// typert-style client-request / server-response envelope on the web server
// (e.g. POST /dsh-whale/<endpoint>, POST /dshbot/<endpoint>).
//
// These are not typert remote.mux methods and are not covered by gen-api;
// they are registered by vendored plugins on the harness origin.
package pluginrpc
