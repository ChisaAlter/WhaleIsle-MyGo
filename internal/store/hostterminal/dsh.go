package hostterminal

import (
	"context"
	"whaleisle/internal/dsh"
)

type dshRemote struct {
	client *dsh.Client
	mux    *dsh.Mux
}

func (r dshRemote) Call(ctx context.Context, method string, args, out any) error {
	return r.client.Call(ctx, method, args, out)
}
func (r dshRemote) Open(ctx context.Context, method string, args any) (Stream, error) {
	return r.mux.Open(ctx, method, map[string]any{"args": args})
}

// Bind binds one authenticated Host generation. Nil detaches, releasing window
// holds while leaving Host cleanup policy in charge of process reclamation.
func (s *Store) Bind(client *dsh.Client, mux *dsh.Mux, ready bool, notify func()) {
	if client == nil || mux == nil {
		s.bind(nil, nil, false, notify)
		return
	}
	s.bind(client, dshRemote{client, mux}, ready, notify)
}

// Transport is called by the application's existing mux supervisor. It does not
// create another connection and ignores callbacks from replaced Hosts.
func (s *Store) Transport(client *dsh.Client, ready bool) { s.transport(client, ready) }
