package plugin

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

type testPlugin struct {
	name string
	init func(HostAPI) error
}

func (p testPlugin) Name() string { return p.name }

func (p testPlugin) Init(api HostAPI) error {
	if p.init != nil {
		return p.init(api)
	}
	return nil
}

func TestRegisterAndInit(t *testing.T) {
	r := NewRuntime()
	var inited []string
	if err := r.Register(testPlugin{name: "b", init: func(api HostAPI) error {
		inited = append(inited, "b")
		api.RegisterCommand(Command{ID: "b-cmd", Title: "B", Run: func() tea.Msg { return nil }})
		api.RegisterSlot("statusbar", SlotItem{Plugin: "b", Priority: 1, View: func(int) string { return "b" }})
		api.RegisterRoute("/b", func(route string) tea.Cmd { return nil })
		return nil
	}}); err != nil {
		t.Fatal(err)
	}
	if err := r.Register(testPlugin{name: "a"}); err != nil {
		t.Fatal(err)
	}
	if err := r.Register(testPlugin{name: "a"}); err == nil {
		t.Fatal("expected duplicate error")
	}
	if err := r.Register(testPlugin{}); err == nil {
		t.Fatal("expected empty name error")
	}
	if err := r.InitAll(); err != nil {
		t.Fatal(err)
	}
	if len(inited) != 1 || r.PluginNames()[0] != "a" {
		t.Fatalf("init order/names wrong: %v %v", inited, r.PluginNames())
	}
	if len(r.Commands()) != 1 || r.Commands()[0].ID != "b-cmd" {
		t.Fatal("command missing")
	}
	if got := r.Slots.Render("statusbar", 80); got != "b" {
		t.Fatalf("slot render=%q", got)
	}
}

func TestRunAndServe(t *testing.T) {
	r := NewRuntime()
	_ = r.Register(testPlugin{name: "p", init: func(api HostAPI) error {
		api.RegisterCommand(Command{ID: "run", Title: "Run", Run: func() tea.Msg { return "ran" }})
		return nil
	}})
	if err := r.InitAll(); err != nil {
		t.Fatal(err)
	}
	if msg := r.RunCommand("run")(); msg != "ran" {
		t.Fatalf("run=%v", msg)
	}
	if cmd := r.RunCommand("missing"); cmd == nil {
		t.Fatal("expected warn cmd for missing command")
	}
	if cmd := r.ServeRoute("/missing"); cmd == nil {
		t.Fatal("expected warn cmd for missing route")
	}
	shim := CommandShim{Runtime: r}
	if msg := shim.Run("run")(); msg != "ran" {
		t.Fatal("shim run broken")
	}
	if msg := (CommandShim{}).Run("x")(); msg == nil {
		t.Fatal("expected nil-runtime warning")
	}
	d := shim.ShimCommand(Command{ID: "run", Title: "Run", Run: func() tea.Msg { return "ran" }})
	if msg := d.Handler(d)(); msg != "ran" {
		t.Fatal("shim dialog handler broken")
	}
}

func TestInitError(t *testing.T) {
	r := NewRuntime()
	_ = r.Register(testPlugin{name: "bad", init: func(HostAPI) error { return errors.New("boom") }})
	if err := r.InitAll(); err == nil {
		t.Fatal("expected init error")
	}
}

func TestAdapters(t *testing.T) {
	cmds := []Command{{ID: "a", Title: "A", Run: func() tea.Msg { return "a" }}}
	d := ToDialogCommands(cmds)
	if len(d) != 1 || d[0].ID != "a" {
		t.Fatal("dialog adapter broken")
	}
	if msg := d[0].Handler(d[0])(); msg != "a" {
		t.Fatal("dialog handler broken")
	}
	p := ToPaletteItems(cmds)
	if len(p) != 1 || p[0].Action() != "a" {
		t.Fatal("palette adapter broken")
	}
	fp := FuncPlugin{PluginName: "f", InitFunc: func(HostAPI) error { return nil }}
	if fp.Name() != "f" || fp.Init(nil) != nil {
		t.Fatal("func plugin broken")
	}
	if (FuncPlugin{PluginName: "nil"}).Init(nil) != nil {
		t.Fatal("nil init should succeed")
	}
}

func TestSlotsEdgeCases(t *testing.T) {
	s := NewSlots()
	if s.Add("", SlotItem{}) {
		t.Fatal("empty slot should be rejected")
	}
	s.Add("b", SlotItem{Plugin: "x", Priority: 1, View: func(int) string { return "b1" }})
	s.Add("a", SlotItem{Plugin: "x", Priority: 5, View: func(int) string { return "" }})
	s.Add("a", SlotItem{Plugin: "y", Priority: 1, View: func(int) string { return "a1" }})
	s.Add("a", SlotItem{Plugin: "z", Priority: 9, View: func(int) string { return "a9" }})
	if got := s.Render("a", 80); got != "a9\na1" {
		t.Fatalf("priority order wrong: %q", got)
	}
	if names := s.Names(); len(names) != 2 || names[0] != "a" {
		t.Fatalf("names=%v", names)
	}
	if len(s.List("missing")) != 0 {
		t.Fatal("expected empty list")
	}
}
