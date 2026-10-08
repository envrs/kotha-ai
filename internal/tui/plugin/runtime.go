package plugin

import (
	"fmt"
	"sort"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kothagpt/kotha/internal/tui/util"
)

// Runtime owns plugin lifecycle: registration, init, command/route lookup.
type Runtime struct {
	mu       sync.RWMutex
	plugins  map[string]Plugin
	commands map[string]Command
	routes   map[string]RouteHandler
	Slots    *Slots
}

func NewRuntime() *Runtime {
	return &Runtime{
		plugins:  map[string]Plugin{},
		commands: map[string]Command{},
		routes:   map[string]RouteHandler{},
		Slots:    NewSlots(),
	}
}

// Register adds a plugin without initializing it. Duplicate names fail.
func (r *Runtime) Register(p Plugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	name := p.Name()
	if name == "" {
		return fmt.Errorf("plugin: empty name")
	}
	if _, exists := r.plugins[name]; exists {
		return fmt.Errorf("plugin %q already registered", name)
	}
	r.plugins[name] = p
	return nil
}

// InitAll initializes every registered plugin in sorted name order,
// wiring them to this runtime as their HostAPI.
func (r *Runtime) InitAll() error {
	r.mu.RLock()
	names := make([]string, 0, len(r.plugins))
	for name := range r.plugins {
		names = append(names, name)
	}
	r.mu.RUnlock()
	sort.Strings(names)
	for _, name := range names {
		r.mu.RLock()
		p := r.plugins[name]
		r.mu.RUnlock()
		if err := p.Init(r); err != nil {
			return fmt.Errorf("plugin %q init: %w", name, err)
		}
	}
	return nil
}

// PluginNames returns registered plugin names in sorted order.
func (r *Runtime) PluginNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.plugins))
	for name := range r.plugins {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// Commands returns all registered commands sorted by ID.
func (r *Runtime) Commands() []Command {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Command, 0, len(r.commands))
	for _, cmd := range r.commands {
		out = append(out, cmd)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// RunCommand executes a command by ID.
func (r *Runtime) RunCommand(id string) tea.Cmd {
	r.mu.RLock()
	cmd, ok := r.commands[id]
	r.mu.RUnlock()
	if !ok || cmd.Run == nil {
		return util.ReportWarn("unknown command: " + id)
	}
	return cmd.Run
}

// ServeRoute dispatches a plugin route, reporting a warning for misses
// (rendered by the PluginRouteMissing core component).
func (r *Runtime) ServeRoute(route string) tea.Cmd {
	r.mu.RLock()
	handler, ok := r.routes[route]
	r.mu.RUnlock()
	if !ok {
		return util.ReportWarn("no plugin handles route: " + route)
	}
	return handler(route)
}

// HostAPI implementation.

func (r *Runtime) RegisterCommand(cmd Command) {
	if cmd.ID == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.commands[cmd.ID] = cmd
}

func (r *Runtime) RegisterSlot(slot string, item SlotItem) {
	r.Slots.Add(slot, item)
}

func (r *Runtime) RegisterRoute(route string, handler RouteHandler) {
	if route == "" || handler == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.routes[route] = handler
}

func (r *Runtime) NotifyInfo(msg string) tea.Cmd { return util.ReportInfo(msg) }
func (r *Runtime) NotifyWarn(msg string) tea.Cmd { return util.ReportWarn(msg) }
func (r *Runtime) NotifyError(err error) tea.Cmd { return util.ReportError(err) }
