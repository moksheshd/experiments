// Command tui is the interactive version of the Chapter 5 lab.
//
// It runs the same three-endpoints-behind-an-auth-gate system as ../main.go, but
// instead of running two fixed scenarios and printing tables, it lets you turn
// two knobs live and watch the blast radius grow and shrink in real time:
//
//   - /profile's offered load (up/down): how hard the one noisy endpoint floods.
//     /notes and /tasks stay pinned at a light, innocent rate.
//   - Arrangement (s): toggle shared gate <-> bulkhead. This is the whole lesson.
//     On the shared gate, flood /profile and watch all three success bars fall
//     together, /notes and /tasks dragged down though their load never moved.
//     Switch to bulkhead and watch /notes and /tasks snap back to full health
//     while only /profile stays down: the damage is contained to its own lane.
//
// The load is open-loop and never backs off, so a flood on the shared road turns
// straight into rejections everywhere behind it. Everything here is a real
// running Go system: real goroutines, real semaphore slots, real auth waits.
// There are no modeled numbers.
//
// Controls: up/down (/profile +/-250 req/s), s (toggle shared/bulkhead), q to quit.
package main

import (
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/moksheshd/experiments/3b-bit-by-byte/season-01-github-august-17/styles"
)

const (
	authSlots   = 30                    // total auth slots
	authWork    = 20 * time.Millisecond // each auth check holds a slot this long
	authTimeout = 50 * time.Millisecond // give up waiting for a slot after this
	handlerWork = 5 * time.Millisecond  // each endpoint's own fast, healthy work
	lightRPS    = 200                   // fixed offered load for /notes and /tasks
	barWidth    = 40
	refresh     = 200 * time.Millisecond
)

// arrangement modes
const (
	modeShared int64 = iota
	modeBulkhead
)

func modeName(m int64) string {
	if m == modeBulkhead {
		return "bulkhead  (one gate per endpoint)"
	}
	return "shared gate  (one gate for all three)"
}

// gate is the auth road: a counting semaphore with a bounded wait.
type gate struct {
	sem chan struct{}
}

func newGate(slots int) *gate { return &gate{sem: make(chan struct{}, slots)} }

// pass runs one auth check: acquire a slot within authTimeout, hold it for the
// auth work, release. Returns false if it timed out (rejected at the door).
func (g *gate) pass() bool {
	t := time.NewTimer(authTimeout)
	select {
	case g.sem <- struct{}{}:
		t.Stop()
		time.Sleep(authWork)
		<-g.sem
		return true
	case <-t.C:
		return false
	}
}

// meter holds one endpoint's windowed success/reject counts, reset each frame.
type meter struct {
	winSuccess atomic.Int64
	winReject  atomic.Int64
}

// endpoint is one surface: a name, its offered rate, and its own metrics.
type endpoint struct {
	name string
	rps  *atomic.Int64
	m    *meter
}

// engine is the live system. Both the shared gate and the three bulkhead lanes
// are always running; the selected mode decides which gate each request uses, so
// toggling is instant and clean to watch.
type engine struct {
	mode   atomic.Int64
	shared *gate
	lanes  []*gate // one per endpoint, for bulkhead mode
	eps    []endpoint
}

func newEngine() *engine {
	e := &engine{
		shared: newGate(authSlots),
		lanes:  []*gate{newGate(authSlots / 3), newGate(authSlots / 3), newGate(authSlots / 3)},
	}
	names := []string{"/notes", "/tasks", "/profile"}
	rates := []int64{lightRPS, lightRPS, 3000}
	for i, n := range names {
		r := &atomic.Int64{}
		r.Store(rates[i])
		e.eps = append(e.eps, endpoint{name: n, rps: r, m: &meter{}})
	}
	e.mode.Store(modeShared)
	return e
}

// run starts one open-loop generator per endpoint. Each fires a flat batch every
// 10ms; /profile's batch tracks its dialed rate live.
func (e *engine) run() {
	for i := range e.eps {
		go e.generate(i)
	}
}

func (e *engine) generate(i int) {
	t := time.NewTicker(10 * time.Millisecond)
	defer t.Stop()
	for range t.C {
		perTick := int(e.eps[i].rps.Load()) * 10 / 1000
		if perTick < 1 {
			perTick = 1
		}
		for j := 0; j < perTick; j++ {
			go e.serve(i)
		}
	}
}

// serve is one request: authenticate at the gate the current mode selects, then
// run the endpoint's own fast handler. The handler is never the bottleneck.
func (e *engine) serve(i int) {
	g := e.shared
	if e.mode.Load() == modeBulkhead {
		g = e.lanes[i]
	}
	if !g.pass() {
		e.eps[i].m.winReject.Add(1)
		return
	}
	time.Sleep(handlerWork)
	e.eps[i].m.winSuccess.Add(1)
}

// snapshot reads and resets one endpoint's window, returning success% and rate.
func (m *meter) snapshot() (successPct, admittedRPS float64) {
	succ := m.winSuccess.Swap(0)
	rej := m.winReject.Swap(0)
	if succ+rej > 0 {
		successPct = float64(succ) / float64(succ+rej) * 100
	}
	admittedRPS = float64(succ) / refresh.Seconds()
	return successPct, admittedRPS
}

// --- TUI ---

var (
	dim = styles.Sub
	ok  = styles.OK
	bad = styles.Err
)

type tickMsg time.Time

type endpointView struct {
	name       string
	offered    int64
	successPct float64
	admitted   float64
}

type model struct {
	e     *engine
	views []endpointView
}

func (m model) Init() tea.Cmd { return tick() }

func tick() tea.Cmd {
	return tea.Tick(refresh, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "up", "k":
			p := m.e.eps[2].rps
			p.Store(clamp(p.Load()+250, 0, 12000))
		case "down", "j":
			p := m.e.eps[2].rps
			p.Store(clamp(p.Load()-250, 0, 12000))
		case "s", " ":
			m.e.mode.Store((m.e.mode.Load() + 1) % 2)
		}
	case tickMsg:
		m.views = m.views[:0]
		for i := range m.e.eps {
			pct, adm := m.e.eps[i].m.snapshot()
			m.views = append(m.views, endpointView{
				name:       m.e.eps[i].name,
				offered:    m.e.eps[i].rps.Load(),
				successPct: pct,
				admitted:   adm,
			})
		}
		return m, tick()
	}
	return m, nil
}

func (m model) View() string {
	mode := m.e.mode.Load()
	capacity := int(float64(authSlots) / authWork.Seconds())

	var b strings.Builder
	b.WriteString(styles.Header(
		"3B: Bit By Byte   Chapter 5: The Road Everyone Shares",
		fmt.Sprintf("auth gate %d slots (~%d auth/s)  ·  arrangement: %s", authSlots, capacity, modeName(mode))) + "\n\n")

	b.WriteString(dim.Render("Success rate per endpoint. /notes and /tasks are pinned at 200 req/s; only") + "\n")
	b.WriteString(dim.Render("/profile floods. Success bar full and green is healthy; short and red is failing.") + "\n\n")

	for _, v := range m.views {
		style := ok
		if v.successPct < 90 {
			style = bad
		}
		b.WriteString(row(
			pad(v.name, 10),
			v.successPct, 100, style,
			fmt.Sprintf("%.0f%%  (offered %d/s, admitting %.0f/s)", v.successPct, v.offered, v.admitted)))
	}

	b.WriteString("\n")
	switch mode {
	case modeShared:
		flooding := len(m.views) == 3 && m.views[0].successPct < 90
		if flooding {
			b.WriteString(bad.Render("Shared gate. /profile's flood fills the one gate, so /notes and /tasks fail") + "\n")
			b.WriteString(bad.Render("too, though their load never moved. Press s to give each its own lane.") + "\n")
		} else {
			b.WriteString(dim.Render("Shared gate. Raise /profile (up) until the flood starves the other two.") + "\n")
		}
	case modeBulkhead:
		b.WriteString(ok.Render("Bulkhead. Each endpoint has its own gate: /profile can only starve itself,") + "\n")
		b.WriteString(ok.Render("so /notes and /tasks stay healthy no matter how hard /profile floods.") + "\n")
	}

	b.WriteString("\n")
	b.WriteString(dim.Render("up/down: /profile +/-250 req/s   s: toggle shared/bulkhead   q: quit"))
	return b.String()
}

func row(name string, v, max float64, style lipgloss.Style, value string) string {
	return fmt.Sprintf("  %s %s  %s\n", dim.Render(name), bar(v, max, style), style.Render(value))
}

func bar(v, max float64, style lipgloss.Style) string {
	if max <= 0 {
		max = 1
	}
	frac := v / max
	if frac > 1 {
		frac = 1
	}
	if frac < 0 {
		frac = 0
	}
	filled := int(frac * barWidth)
	return style.Render(strings.Repeat("█", filled)) + dim.Render(strings.Repeat("·", barWidth-filled))
}

func clamp(v, lo, hi int64) int64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func pad(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

func main() {
	e := newEngine()
	e.run()

	p := tea.NewProgram(model{e: e})
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
