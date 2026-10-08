package core

import "time"

// ConnectedTracker models a "useConnected"-style liveness hook: components
// mark heartbeats and query staleness without owning timers.
type ConnectedTracker struct {
	LastSeen   time.Time
	StaleAfter time.Duration
	everSeen   bool
}

func NewConnectedTracker(staleAfter time.Duration) *ConnectedTracker {
	if staleAfter <= 0 {
		staleAfter = 30 * time.Second
	}
	return &ConnectedTracker{StaleAfter: staleAfter}
}

func (c *ConnectedTracker) MarkAlive(now time.Time) {
	c.LastSeen = now
	c.everSeen = true
}

func (c *ConnectedTracker) Connected(now time.Time) bool {
	if !c.everSeen {
		return false
	}
	return now.Sub(c.LastSeen) <= c.StaleAfter
}
