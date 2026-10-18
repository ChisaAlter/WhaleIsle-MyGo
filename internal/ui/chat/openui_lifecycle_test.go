package chat

import (
	"context"
	"errors"
	"testing"
)

func TestOpenUIContextOwnedLifetime(t *testing.T) {
	ctx, cancel := newOpenUIContext()
	defer cancel()
	if deadline, ok := ctx.Deadline(); ok {
		t.Fatalf("active form has arbitrary deadline %v", deadline)
	}
	select {
	case <-ctx.Done():
		t.Fatal("new form already closed")
	default:
	}
	cancel()
	select {
	case <-ctx.Done():
	default:
		t.Fatal("explicit close did not cancel")
	}
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatalf("close reason %v", ctx.Err())
	}
}

func TestOpenUIFiveIndependentLifetimes(t *testing.T) {
	contexts := make([]context.Context, 5)
	cancels := make([]context.CancelFunc, 5)
	for i := range contexts {
		contexts[i], cancels[i] = newOpenUIContext()
		defer cancels[i]()
	}
	for i, ctx := range contexts {
		if _, ok := ctx.Deadline(); ok {
			t.Fatalf("instance %d has deadline", i+1)
		}
		select {
		case <-ctx.Done():
			t.Fatalf("instance %d cannot stay active", i+1)
		default:
		}
	}
	cancels[0]()
	for i := 1; i < len(contexts); i++ {
		select {
		case <-contexts[i].Done():
			t.Fatalf("closing first revoked instance %d", i+1)
		default:
		}
	}
}
