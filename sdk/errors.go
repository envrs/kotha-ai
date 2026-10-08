package sdk

import (
	"errors"

	"github.com/kothagpt/kotha/internal/llm/agent"
)

// Sentinel errors for misconfigured clients and re-exported agent errors.
var (
	ErrNoSessions = errors.New("sdk: no session service")
	ErrNoMessages = errors.New("sdk: no message service")
	ErrNoAgent    = errors.New("sdk: no agent service")

	ErrRequestCancelled = agent.ErrRequestCancelled
	ErrSessionBusy      = agent.ErrSessionBusy
)
