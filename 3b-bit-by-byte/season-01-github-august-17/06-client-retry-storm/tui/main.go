// Command tui is the interactive version of the Chapter 6 lab.
//
// It runs the same fleet -> endpoint system as ../main.go, but instead of
// sweeping fixed scenarios and printing a table, it lets you flip the fault and
// the fixes live and watch the amplification react:
//
//   - f: toggle the endpoint fault (fast responses vs slow, delayed ones).
//   - o: toggle failover to a fresh region. Watch the amplification NOT move,
//     because the clients generate the load and they never changed.
//   - s: cycle the client behavior: buggy -> sane timeout -> backoff+jitter ->
//     retry budget. Watch the storm bar collapse as discipline is added.
//
// Everything here is a real running Go system: real client goroutines, real
// heartbeats, real reissued attempts. The one bar you watch is the aggregate
// attempts per second reaching the endpoint, as a multiple of the calm baseline.
//
// Controls: f (fault), o (failover), s (cycle fix), q to quit.
package main

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/moksheshd/experiments/3b-bit-by-byte/season-01-github-august-17/styles"
)

const (
	clients  = 800                    // fleet size
	interval = 500 * time.Millisecond // per-client heartbeat
	fastResp = 20 * time.Millisecond  // healthy endpoint response
	slowResp = 500 * time.Millisecond // faulted endpoint response
	timeout  = 50 * time.Millisecond  // buggy client's aggressive timeout
	saneTO   = 800 * time.Millisecond // disciplined timeout (longer than a slow response)
	backoff  = 50 * time.Millisecond  // disciplined base backoff
	maxTry   = 40                     // safety cap
	barWidth = 40
	scaleMax = 12.0 // fixed bar scale in multiples of baseline
	refresh  = 200 * time.Millisecond
)

// client fix modes
const (
	fixNone    int64 = iota // the bug: reissue immediately on timeout
	fixTimeout              // sane timeout, longer than a slow response
	fixBackoff              // backoff + jitter
	fixBudget               // backoff + jitter + a hard per-request cap (stand-in for a fleet budget)
)

func fixName(m int64) string {
	switch m {
	case fixTimeout:
		return "sane timeout"
	case fixBackoff:
		return "backoff + jitter"
	case fixBudget:
		return "retry budget"
	default:
		return "buggy (reissue immediately)"
	}
}

// engine is the live fleet -> endpoint system. The TUI reads its atomics each
// frame; the clients read the fault and fix flags each logical request.
type engine struct {
	fault    atomic.Bool  // endpoint slow?
	failover atomic.Bool  // fresh region? (cosmetic: the clients are the load)
	fix      atomic.Int64 // client behavior
	seeds    atomic.Int64

	winAttempts atomic.Int64 // attempts this frame (reset each snapshot)
}

func (e *engine) serve() {
	if e.fault.Load() {
		time.Sleep(slowResp)
		return
	}
	time.Sleep(fastResp)
}

func (e *engine) logical(seed int64) {
	rng := rand.New(rand.NewSource(seed))
	fix := e.fix.Load()
	to := timeout
	tryCap := maxTry
	if fix == fixTimeout {
		to = saneTO
	}
	if fix == fixBudget {
		tryCap = 2 // hard cap: one try plus one retry, then fail fast
	}

	done := make(chan struct{})
	got := make(chan struct{}, 1)
	go func() {
		wait := backoff
		for tries := 0; tries < tryCap; tries++ {
			select {
			case <-done:
				return
			default:
			}
			e.winAttempts.Add(1)
			go func() {
				e.serve()
				select {
				case got <- struct{}{}:
				default:
				}
			}()
			step := to
			if fix == fixBackoff || fix == fixBudget {
				step = time.Duration(float64(wait) * (0.5 + rng.Float64()))
				wait *= 2
			}
			select {
			case <-done:
				return
			case <-time.After(step):
			}
		}
	}()
	select {
	case <-got:
	case <-time.After(slowResp + 2*time.Second):
	}
	close(done)
}

func (e *engine) run() {
	for i := 0; i < clients; i++ {
		go func() {
			t := time.NewTicker(interval)
			defer t.Stop()
			for range t.C {
				go e.logical(e.seeds.Add(1))
			}
		}()
	}
}

func (e *engine) snapshot() float64 {
	att := e.winAttempts.Swap(0)
	return float64(att) / refresh.Seconds()
}

// --- TUI ---

var (
	dim  = styles.Sub
	okS  = styles.OK
	warn = styles.Attn
	bad  = styles.Err
)

type tickMsg time.Time

type model struct {
	e        *engine
	rps      float64
	baseline float64
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
		case "f":
			m.e.fault.Store(!m.e.fault.Load())
		case "o":
			m.e.failover.Store(!m.e.failover.Load())
		case "s", " ":
			m.e.fix.Store((m.e.fix.Load() + 1) % 4)
		}
	case tickMsg:
		m.rps = m.e.snapshot()
		return m, tick()
	}
	return m, nil
}

func (m model) View() string {
	ampl := m.rps / m.baseline
	fault := m.e.fault.Load()
	failover := m.e.failover.Load()
	fix := m.e.fix.Load()

	faultStr := "healthy"
	if fault {
		faultStr = "SLOW (delayed responses)"
	}
	failStr := "primary region"
	if failover {
		failStr = "failed over to fresh region"
	}

	var b strings.Builder
	b.WriteString(styles.Header(
		"3B: Bit By Byte   Chapter 6: The Copilot Retry Storm",
		fmt.Sprintf("fleet %d  ·  baseline %.0f req/s  ·  endpoint: %s", clients, m.baseline, faultStr)) + "\n\n")

	b.WriteString(dim.Render(fmt.Sprintf("Client behavior: %s", fixName(fix))) + "\n")
	b.WriteString(dim.Render(fmt.Sprintf("Target: %s", failStr)) + "\n\n")

	style := okS
	if ampl >= 2 {
		style = bad
	} else if ampl >= 1.5 {
		style = warn
	}
	b.WriteString(fmt.Sprintf("  %s %s  %s\n",
		dim.Render("Load reaching endpoint "),
		bar(ampl, scaleMax, style),
		style.Render(fmt.Sprintf("%.1fx  (%.0f req/s)", ampl, m.rps))))

	b.WriteString("\n")
	switch {
	case fault && ampl >= 2 && failover:
		b.WriteString(warn.Render("Failover did nothing. The clients generate the load, so moving the") + "\n")
		b.WriteString(warn.Render("server just points the same storm at a fresh one. Press s to fix the client.") + "\n")
	case fault && ampl >= 2:
		b.WriteString(warn.Render("The storm is running: one slow response spawns a stack of retries.") + "\n")
		b.WriteString(warn.Render("Press o to fail over (watch it not help), or s to fix the client.") + "\n")
	case fault:
		b.WriteString(okS.Render("Disciplined clients: the endpoint is slow, but the load stays near 1x.") + "\n")
	default:
		b.WriteString(dim.Render("Endpoint healthy. Press f to fault it and start the storm.") + "\n")
	}

	b.WriteString("\n")
	b.WriteString(dim.Render("f: fault endpoint   o: failover   s: cycle client fix   q: quit"))
	return b.String()
}

func bar(v, max float64, style lipgloss.Style) string {
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

func main() {
	e := &engine{}
	e.fault.Store(true) // start in the storm so there is something to watch
	go e.run()

	baseline := float64(clients) / interval.Seconds()
	p := tea.NewProgram(model{e: e, baseline: baseline})
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
