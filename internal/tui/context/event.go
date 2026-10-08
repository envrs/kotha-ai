package tuicontext

import "time"

// Event is a status-bar notification.
type EventLevel string

const (
	EventInfo  EventLevel = "info"
	EventWarn  EventLevel = "warn"
	EventError EventLevel = "error"
)

type Event struct {
	Level   EventLevel
	Message string
	TTL     time.Duration
	At      time.Time
}

func InfoEvent(msg string) Event {
	return Event{Level: EventInfo, Message: msg, At: time.Now()}
}

func (e Event) Expired(now time.Time) bool {
	if e.TTL <= 0 {
		return false
	}
	return now.Sub(e.At) > e.TTL
}
