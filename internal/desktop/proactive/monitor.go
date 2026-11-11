package proactive

import (
	"context"
	"errors"
	"sync"
	"time"
)

type SourceReader interface {
	Read(context.Context, Source) ([]SourceItem, error)
}
type NoticeDelivery func(context.Context, Notice) (string, error)

// Monitor owns polling and local notification delivery independently of a Host.
// Source mutations cancel only its current read; execution is owned by Runner.
type Monitor struct {
	ctx        context.Context
	cancel     context.CancelFunc
	service    *Service
	reader     SourceReader
	deliver    NoticeDelivery
	notify     func(error)
	wake       chan struct{}
	done       chan struct{}
	mu         sync.Mutex
	readCancel context.CancelFunc
}

func NewMonitor(parent context.Context, service *Service, reader SourceReader, deliver NoticeDelivery, notify func(error)) *Monitor {
	ctx, cancel := context.WithCancel(parent)
	m := &Monitor{ctx: ctx, cancel: cancel, service: service, reader: reader, deliver: deliver, notify: notify, wake: make(chan struct{}, 1), done: make(chan struct{})}
	go m.loop()
	return m
}
func (m *Monitor) Wake() {
	select {
	case m.wake <- struct{}{}:
	default:
	}
}
func (m *Monitor) SourcesChanged() {
	m.mu.Lock()
	if m.readCancel != nil {
		m.readCancel()
	}
	m.mu.Unlock()
	m.Wake()
}
func (m *Monitor) Close()  { m.cancel(); <-m.done }
func (m *Monitor) Cancel() { m.cancel() }
func (m *Monitor) changed(err error) {
	if m.notify != nil {
		m.notify(err)
	}
}

func (m *Monitor) loop() {
	defer close(m.done)
	if err := m.service.recoverNotices(); err != nil {
		m.changed(err)
		return
	}
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		if m.ctx.Err() != nil {
			return
		}
		m.deliverPending()
		for _, source := range m.service.Sources() {
			if m.ctx.Err() != nil {
				return
			}
			if !source.Enabled || m.reader == nil {
				continue
			}
			if next, err := time.Parse(time.RFC3339Nano, source.NextCheck); err == nil && next.After(time.Now()) {
				continue
			}
			ctx, cancel := context.WithTimeout(m.ctx, 90*time.Second)
			m.mu.Lock()
			m.readCancel = cancel
			m.mu.Unlock()
			started := time.Now()
			items, err := m.reader.Read(ctx, source)
			cancelled := errors.Is(ctx.Err(), context.Canceled)
			cancel()
			m.mu.Lock()
			m.readCancel = nil
			m.mu.Unlock()
			if cancelled {
				continue
			}
			applyErr := m.service.ApplyDiscovery(source, started, items, err)
			if applyErr != nil && !errors.Is(applyErr, ErrConflict) {
				if saveErr := m.service.ApplyDiscovery(source, started, nil, applyErr); saveErr != nil {
					m.changed(saveErr)
					return
				}
				m.changed(applyErr)
			} else {
				m.changed(nil)
			}
		}
		select {
		case <-m.ctx.Done():
			return
		case <-tick.C:
		case <-m.wake:
		}
	}
}

func (m *Monitor) deliverPending() {
	for _, notice := range m.service.Notices() {
		if m.ctx.Err() != nil {
			return
		}
		if notice.DesktopStatus != "pending" || notice.ReadAt != "" {
			continue
		}
		owned, err := m.service.ClaimNotice(notice.ID)
		if errors.Is(err, ErrConflict) {
			continue
		}
		if err != nil {
			m.changed(err)
			return
		}
		status := "in-app"
		if m.deliver != nil {
			status, err = m.deliver(m.ctx, owned)
		}
		if m.ctx.Err() != nil {
			return
		}
		message := ""
		if err != nil {
			status, message = "failed", err.Error()
		}
		if saveErr := m.service.FinishNotice(owned.ID, status, message); saveErr != nil {
			m.changed(saveErr)
			return
		}
		m.changed(nil)
	}
}
