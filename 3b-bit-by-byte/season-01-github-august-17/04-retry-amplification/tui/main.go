// Command tui is the interactive version of the Chapter 4 lab.
//
// It runs the same open-loop Client -> Service as ../main.go, but instead of
// sweeping fixed offered-load steps and printing tables, it lets you turn two
// knobs live and watch retry amplification happen in real time:
//
//   - Offered load (up/down): the flat rate of logical requests the client
//     generates. Push it past the service's ~1250 req/s capacity and the service
//     starts rejecting.
//   - Retry discipline (s): toggle naive <-> disciplined. This is the whole
//     lesson. On naive, raise the load and watch the "attempts@service" bar lift
//     off from the "offered" bar as retries pile on. Press s for disciplined
//     (backoff + jitter + a 10% retry budget) and watch that bar collapse back
//     down to the offered bar, same load, while goodput barely moves.
//
// The offered rate is fixed and never backs off, so an overloaded service turns
// straight into rejections and retries. Everything here is a real running Go
// system: real goroutines, a real concurrency-limited service, real sleeps. There
// are no modeled numbers.
//
// Controls: up/down (offered +/-250 req/s), s (toggle naive/disciplined), q to quit.
package main

import (
	"fmt"
	"math/rand/v2"
	"os"
	"strings"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/moksheshd/experiments/3b-bit-by-byte/season-01-github-august-17/styles"
)

const (
	limit       = 50                    // service concurrency limit
	serviceMs   = 40                    // real work per admitted request, ms
	maxAttempts = 5                     // attempts per logical request
	budgetFrac  = 0.10                  // disciplined: retry budget as fraction of offered
	backoffBase = 50 * time.Millisecond // disciplined: base backoff, doubled each retry
	barWidth    = 40
	scaleMaxRPS = 20000.0 // fixed bar scale for the rate bars (offered + attempts)
	refresh     = 200 * time.Millisecond
)

// retry discipline modes
const (
	modeNaive int64 = iota
	modeDisciplined
)

func modeName(m int64) string {
	if m == modeDisciplined {
		return "disciplined  (backoff + jitter + 10% budget)"
	}
	return "naive  (retry immediately, no budget)"
}

func capacityRPS() float64 { return float64(limit) / (float64(serviceMs) / 1000) }

// engine is the live open-loop Client -> Service. The TUI reads its atomics each
// frame; a generator keeps the offered rate flat and each logical request retries
// according to the selected discipline.
type engine struct {
	offered atomic.Int64 // offered logical req/s (you dial this)
	mode    atomic.Int64 // naive / disciplined (you toggle this)

	inFlight atomic.Int64 // requests in flight at the service right now

	winOffered  atomic.Int64 // logical requests started this frame
	winAttempts atomic.Int64 // attempts reaching the service this frame
	winSuccess  atomic.Int64 // logical successes this frame

	budget bucket
}

func newEngine() *engine {
	e := &engine{}
	e.offered.Store(1000)
	e.mode.Store(modeNaive)
	e.budget.cap = int64(budgetFrac*4000*0.25) + 1 // sized for the top of the dial
	return e
}

// call is one attempt at the service: admit if below the limit, else reject fast.
func (e *engine) call() bool {
	for {
		n := e.inFlight.Load()
		if n >= limit {
			return false // service full: rejected
		}
		if e.inFlight.CompareAndSwap(n, n+1) {
			break
		}
	}
	time.Sleep(time.Duration(serviceMs) * time.Millisecond)
	e.inFlight.Add(-1)
	return true
}

// generate is the open-loop load: a fixed batch of logical requests every 10ms.
func (e *engine) generate() {
	t := time.NewTicker(10 * time.Millisecond)
	defer t.Stop()
	for range t.C {
		perTick := int(e.offered.Load()) * 10 / 1000
		for i := 0; i < perTick; i++ {
			go e.logical()
		}
	}
}

// logical is one logical request's whole life: try, and on rejection retry per the
// current discipline. Naive retries immediately; disciplined spends a budget token
// and waits a backed-off, jittered interval, or gives up when the budget is spent.
func (e *engine) logical() {
	e.winOffered.Add(1)
	disciplined := e.mode.Load() == modeDisciplined
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 && disciplined {
			if !e.budget.take() {
				return // budget spent: fail fast
			}
			time.Sleep(backoffDur(backoffBase, attempt-1))
		}
		e.winAttempts.Add(1)
		if e.call() {
			e.winSuccess.Add(1)
			return
		}
	}
}

// refillBudget tops up the disciplined retry budget at budgetFrac x offered
// tokens/sec, carrying the fractional remainder in this single goroutine.
func (e *engine) refillBudget() {
	const refill = 20 * time.Millisecond
	carry := 0.0
	t := time.NewTicker(refill)
	defer t.Stop()
	for range t.C {
		carry += budgetFrac * float64(e.offered.Load()) * refill.Seconds()
		whole := int64(carry)
		if whole > 0 {
			carry -= float64(whole)
			e.budget.add(whole)
		}
	}
}

// snapshot reads the counters for one frame and resets the windowed ones.
func (e *engine) snapshot() (offeredRate, attemptsRate, amp, goodput, successPct float64) {
	off := e.winOffered.Swap(0)
	att := e.winAttempts.Swap(0)
	good := e.winSuccess.Swap(0)
	sec := refresh.Seconds()
	offeredRate = float64(off) / sec
	attemptsRate = float64(att) / sec
	goodput = float64(good) / sec
	if off > 0 {
		amp = float64(att) / float64(off)
		successPct = float64(good) / float64(off) * 100
	}
	return offeredRate, attemptsRate, amp, goodput, successPct
}

// bucket is a lock-free token bucket for the retry budget.
type bucket struct {
	tokens atomic.Int64
	cap    int64
}

func (b *bucket) take() bool {
	for {
		n := b.tokens.Load()
		if n <= 0 {
			return false
		}
		if b.tokens.CompareAndSwap(n, n-1) {
			return true
		}
	}
}

func (b *bucket) add(n int64) {
	for {
		old := b.tokens.Load()
		nv := old + n
		if nv > b.cap {
			nv = b.cap
		}
		if b.tokens.CompareAndSwap(old, nv) {
			return
		}
	}
}

func backoffDur(base time.Duration, retry int) time.Duration {
	shift := retry - 1
	if shift > 10 {
		shift = 10
	}
	span := int64(base) << shift
	if span <= 0 {
		return 0
	}
	return time.Duration(rand.Int64N(span) + 1)
}

// --- TUI ---

var (
	label = styles.Sub
	dim   = styles.Sub
	load  = lipgloss.NewStyle().Foreground(styles.Accent) // blue: offered load
	ok    = styles.OK                                     // green: healthy
	warn  = styles.Attn                                   // amber: amplifying
	bad   = styles.Err                                    // red: failing
)

type tickMsg time.Time

type model struct {
	e            *engine
	offeredRate  float64
	attemptsRate float64
	amp          float64
	goodput      float64
	successPct   float64
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
			m.e.offered.Store(clamp(m.e.offered.Load()+250, 250, 8000))
		case "down", "j":
			m.e.offered.Store(clamp(m.e.offered.Load()-250, 250, 8000))
		case "s", " ":
			m.e.mode.Store((m.e.mode.Load() + 1) % 2)
		}
	case tickMsg:
		m.offeredRate, m.attemptsRate, m.amp, m.goodput, m.successPct = m.e.snapshot()
		return m, tick()
	}
	return m, nil
}

func (m model) View() string {
	offered := m.e.offered.Load()
	mode := m.e.mode.Load()

	var b strings.Builder
	b.WriteString(styles.Header(
		"3B: Bit By Byte   Chapter 4: When Retries Become the Outage",
		fmt.Sprintf("service limit %d  ·  capacity about %.0f req/s  ·  retries: %s", limit, capacityRPS(), modeName(mode))) + "\n\n")

	b.WriteString(dim.Render(fmt.Sprintf("Offered load: %d req/s   (service can clear about %.0f)", offered, capacityRPS())) + "\n\n")

	// Offered and attempts share one scale, so you can watch attempts lift off
	// from offered under naive retries and collapse back under disciplined.
	b.WriteString(row("Offered       ", m.offeredRate, scaleMaxRPS, load,
		fmt.Sprintf("%.0f/s", m.offeredRate)))
	b.WriteString(row("Attempts@svc  ", m.attemptsRate, scaleMaxRPS, ampStyle(m.amp),
		fmt.Sprintf("%.0f/s  (%.1fx offered)", m.attemptsRate, m.amp)))
	b.WriteString("\n")
	b.WriteString(row("Goodput       ", m.goodput, scaleMaxRPS, ok,
		fmt.Sprintf("%.0f/s  (capacity %.0f)", m.goodput, capacityRPS())))
	b.WriteString(row("Success rate  ", m.successPct, 100, rateStyle(m.successPct),
		fmt.Sprintf("%.0f%%", m.successPct)))

	b.WriteString("\n")
	if mode == modeNaive {
		if m.amp >= 1.5 {
			b.WriteString(warn.Render("Attempts have lifted off from offered: every rejection is retried") + "\n")
			b.WriteString(warn.Render("immediately, so the load balloons while goodput stays at the ceiling.") + "\n")
			b.WriteString(warn.Render("Retries added load, not capacity. Press s to add discipline.") + "\n")
		} else {
			b.WriteString(dim.Render("Naive retries. Raise the offered load (up) past capacity and watch") + "\n")
			b.WriteString(dim.Render("the attempts bar lift off from the offered bar.") + "\n")
		}
	} else {
		b.WriteString(ok.Render("Disciplined. Backoff, jitter, and a 10% budget pin attempts near the") + "\n")
		b.WriteString(ok.Render("offered rate, so the service keeps its goodput without the storm.") + "\n")
	}

	b.WriteString("\n")
	b.WriteString(dim.Render("up/down: offered +/-250   s: toggle naive/disciplined   q: quit"))
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

func ampStyle(amp float64) lipgloss.Style {
	switch {
	case amp < 1.5:
		return ok
	case amp < 2.5:
		return warn
	default:
		return bad
	}
}

func rateStyle(pct float64) lipgloss.Style {
	switch {
	case pct >= 95:
		return ok
	case pct >= 50:
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

func main() {
	e := newEngine()
	go e.generate()
	go e.refillBudget()

	p := tea.NewProgram(model{e: e})
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
