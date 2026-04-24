package poller

import (
	"time"

	"github.com/meopedevts/revu/internal/store"
)

// Event kinds emitted by the poller. Strings are stable — they are used as
// the event name on the Wails bus (frontend listens for these literals).
const (
	EventPRNew           = "pr:new"
	EventPRStatusChanged = "pr:status-changed"
	EventPollCompleted   = "poll:completed"
)

// Event is the payload passed to the registered EventHandler. PR is set for
// per-PR events (New, StatusChanged); At is always populated; Err is set
// only on poll:completed when the tick failed.
type Event struct {
	Kind string          `json:"kind"`
	PR   *store.PRRecord `json:"pr,omitempty"`
	At   time.Time       `json:"at"`
	Err  string          `json:"err,omitempty"`
}

// EventHandler receives events from the poller. Handlers should be fast —
// they run inline on the polling goroutine. Fan-out or buffering is the
// handler's responsibility.
type EventHandler func(Event)

// WithEventHandler wires a handler that receives per-poll events. Passing
// nil disables emission.
func WithEventHandler(h EventHandler) Option {
	return func(p *Poller) {
		p.eventHandler = h
	}
}

func (p *Poller) emit(e Event) {
	if p.eventHandler == nil {
		return
	}
	if e.At.IsZero() {
		e.At = time.Now().UTC()
	}
	p.eventHandler(e)
}
