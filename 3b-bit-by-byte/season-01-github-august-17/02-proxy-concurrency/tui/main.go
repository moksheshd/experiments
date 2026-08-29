// Command tui is the interactive version of the Chapter 2 lab.
//
// It runs the same Client -> Proxy -> Service system as ../main.go, but instead
// of sweeping fixed concurrency levels and printing a table, it lets you turn the
// client-concurrency knob live and watch two bars react in real time:
//
//   - Client concurrency: the load you are applying (you control this).
//   - Service in-flight: how many requests are actually inside the service. On
//     the same scale as the client bar, so you can watch it climb with the load
//     and then FREEZE at the proxy limit while the client bar keeps growing.
//   - Success rate: the fraction of requests getting through, which collapses
//     once you push past the limit.
//
// Everything here is a real running Go system: real goroutines, real proxy slots,
// real 20ms of work per request. Press up/down to change the load and feel the
// ceiling.
//
// Controls: up/down (+/-5 clients), left/right (+/-10 proxy limit), q to quit.
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
	serviceDur = 20 * time.Millisecond // work per request inside the service
	scaleMax   = 150.0                 // fixed bar scale so the ceiling is visible
	barWidth   = 40
	refresh    = 120 * time.Millisecond
)

// engine is the live Client -> Proxy -> Service system. The TUI reads its atomics
// each frame; a manager goroutine keeps the worker count equal to `desired`.
type engine struct {
	limit    atomic.Int64 // proxy concurrency limit (adjustable)
	desired  atomic.Int64 // how many client workers should be running (adjustable)
	inFlight atomic.Int64 // requests currently inside the service (live)

	winPeak    atomic.Int64 // peak service in-flight this frame (reset each snapshot)
	winSuccess atomic.Int64 // successes this frame (reset each snapshot)
	winTotal   atomic.Int64 // attempts this frame (reset each snapshot)
}

// run is the manager loop: it spawns and stops client workers so the live worker
// count tracks `desired`. Each worker hammers the proxy in a closed loop.
func (e *engine) run() {
	var stops []chan struct{}
	t := time.NewTicker(30 * time.Millisecond)
	defer t.Stop()
	for range t.C {
		want := int(e.desired.Load())
		for len(stops) < want {
			s := make(chan struct{})
			stops = append(stops, s)
			go e.worker(s)
		}
		for len(stops) > want {
			close(stops[len(stops)-1])
			stops = stops[:len(stops)-1]
		}
	}
}

// worker is one client: fire a request, tally the result, repeat until stopped.
func (e *engine) worker(stop <-chan struct{}) {
	// Each worker holds its own proxy slot for the duration of a served request,
	// so the shared limit is enforced by comparing in-flight against the limit.
	for {
		select {
		case <-stop:
			return
		default:
		}
		e.winTotal.Add(1)
		if e.serve() {
			e.winSuccess.Add(1)
		}
	}
}

// serve is one request's whole life. The proxy admits it only if fewer than
// `limit` requests are already in flight; otherwise it is rejected at the door.
// This models the same capacity constraint as ../main.go's semaphore, using a
// live-adjustable limit.
func (e *engine) serve() bool {
	limit := e.limit.Load()
	// Try to claim a slot: bump in-flight, but only if we stay within the limit.
	for {
		cur := e.inFlight.Load()
		if cur >= limit {
			// Proxy full: rejected. Back off one service time (the slot will not
			// free sooner) so the closed loop does not spin at CPU speed.
			time.Sleep(serviceDur)
			return false
		}
		if e.inFlight.CompareAndSwap(cur, cur+1) {
			updatePeak(&e.winPeak, cur+1)
			break
		}
	}
	time.Sleep(serviceDur) // the service's real work
	e.inFlight.Add(-1)
	return true
}

// snapshot reads the live counters for one frame and resets the windowed ones so
// the next frame measures a fresh interval.
func (e *engine) snapshot() (inFlight, peak int64, rate float64) {
	inFlight = e.inFlight.Load()
	peak = e.winPeak.Swap(inFlight)
	total := e.winTotal.Swap(0)
	success := e.winSuccess.Swap(0)
	if total > 0 {
		rate = float64(success) / float64(total) * 100
	} else {
		rate = 100
	}
	return inFlight, peak, rate
}

func updatePeak(peak *atomic.Int64, v int64) {
	for {
		old := peak.Load()
		if v <= old || peak.CompareAndSwap(old, v) {
			return
		}
	}
}

// --- TUI ---

// Colors come from the season's shared theme so the TUI reads as part of
// 3B: Bit By Byte, not a one-off.
var (
	label = styles.Sub
	dim   = styles.Sub
	load  = lipgloss.NewStyle().Foreground(styles.Accent) // blue: client load
	ok    = styles.OK                                     // green: healthy
	warn  = styles.Attn                                   // amber: at the ceiling
	bad   = styles.Err                                    // red: failing
)

type tickMsg time.Time

type model struct {
	e        *engine
	inFlight int64
	peak     int64
	rate     float64
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
			m.e.desired.Store(clamp(m.e.desired.Load()+5, 0, 200))
		case "down", "j":
			m.e.desired.Store(clamp(m.e.desired.Load()-5, 0, 200))
		case "right", "l":
			m.e.limit.Store(clamp(m.e.limit.Load()+10, 1, 200))
		case "left", "h":
			m.e.limit.Store(clamp(m.e.limit.Load()-10, 1, 200))
		}
	case tickMsg:
		m.inFlight, m.peak, m.rate = m.e.snapshot()
		return m, tick()
	}
	return m, nil
}

func (m model) View() string {
	desired := m.e.desired.Load()
	limit := m.e.limit.Load()

	var b strings.Builder
	b.WriteString(styles.Header(
		"3B: Bit By Byte   Chapter 2: The Proxy Beside Your Application",
		"Client -> Proxy (limit "+itoa(limit)+") -> Service ("+serviceDur.String()+"/req)  ·  interactive") + "\n\n")

	// Client load and service in-flight share one scale, so the split is visible:
	// the load bar keeps growing while the service bar freezes at the limit.
	b.WriteString(row("Client load    ", float64(desired), scaleMax, load,
		fmt.Sprintf("%d clients", desired)))
	b.WriteString(row("Service in-flt ", float64(m.peak), scaleMax, serviceStyle(m.peak, limit),
		fmt.Sprintf("%d  (cap %d)", m.peak, limit)))
	b.WriteString("\n")
	b.WriteString(row("Success rate   ", m.rate, 100, rateStyle(m.rate),
		fmt.Sprintf("%.0f%%", m.rate)))

	b.WriteString("\n")
	if m.peak >= limit && desired > limit {
		b.WriteString(warn.Render("The service is pinned at the limit. Pushing more load only grows") + "\n")
		b.WriteString(warn.Render("rejections; the service cannot get busier. Green service, failing client.") + "\n")
	} else {
		b.WriteString(dim.Render("Below the limit: every request gets through, the service tracks the load.") + "\n")
	}

	b.WriteString("\n")
	b.WriteString(dim.Render("up/down: client load +/-5   left/right: proxy limit +/-10   q: quit"))
	return b.String()
}

func row(name string, v, max float64, style lipgloss.Style, value string) string {
	return fmt.Sprintf("  %s %s  %s\n", label.Render(name), bar(v, max, style), style.Render(value))
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

func serviceStyle(peak, limit int64) lipgloss.Style {
	if peak >= limit {
		return warn // pinned at the ceiling
	}
	return ok
}

func rateStyle(rate float64) lipgloss.Style {
	switch {
	case rate >= 99:
		return ok
	case rate >= 75:
		return warn
	default:
		return bad
	}
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

func itoa(v int64) string { return fmt.Sprintf("%d", v) }

func main() {
	e := &engine{}
	e.limit.Store(50)
	e.desired.Store(30)
	go e.run()

	p := tea.NewProgram(model{e: e})
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
