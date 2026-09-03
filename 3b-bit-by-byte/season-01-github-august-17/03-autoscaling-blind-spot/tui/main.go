// Command tui is the interactive version of the Chapter 3 lab.
//
// It runs the same open-loop Client -> Proxy -> Service as ../main.go, but instead
// of sweeping fixed latency steps and printing tables, it lets you turn two knobs
// live and watch the autoscaling blind spot happen in real time:
//
//   - Service latency (up/down): how long each request takes. Raise it and the
//     backend "gets slower," exactly like a degrading dependency.
//   - Autoscaler signal (s): cycle none -> CPU -> concurrency. This is the whole
//     lesson. On CPU, raise the latency and watch concurrency pin at the limit,
//     CPU fall, replicas stay flat, and the reject bar climb. Switch to
//     concurrency and watch replicas jump and the rejects vanish, same load.
//
// The arrival rate is fixed and never backs off, so rising latency turns straight
// into rising concurrency. Everything here is a real running Go system: real
// goroutines, real proxy slots, real sleeps. Service CPU is the one modeled
// number, derived from measured throughput.
//
// Controls: up/down (latency +/-25ms), s (cycle autoscaler signal), q to quit.
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
	ratePerSec      = 1000 // fixed arrival rate (open loop)
	limitPerReplica = 100  // proxy concurrency limit per replica
	cpuBaseline     = 40.0 // modeled CPU% when throughput == arrival rate
	targetCPU       = 70.0 // CPU autoscaler target
	targetUtil      = 80.0 // concurrency autoscaler target
	maxReplicas     = 20
	barWidth        = 40
	scaleMaxConc    = 800.0 // fixed bar scale for the concurrency bars
	refresh         = 150 * time.Millisecond
	controlInterval = 300 * time.Millisecond
)

// scale signal modes
const (
	scaleNone int64 = iota
	scaleCPU
	scaleConc
)

func scaleName(m int64) string {
	switch m {
	case scaleCPU:
		return "CPU  (the blind spot)"
	case scaleConc:
		return "proxy concurrency  (the fix)"
	default:
		return "none"
	}
}

// engine is the live open-loop Client -> Proxy -> Service with an autoscaler.
// The TUI reads its atomics each frame; a generator keeps the arrival rate flat
// and an autoscaler resizes the replica count from whichever signal is selected.
type engine struct {
	latency   atomic.Int64 // per-request service work, in ms (you dial this)
	scaleMode atomic.Int64 // none / cpu / concurrency (you toggle this)
	replicas  atomic.Int64 // current replica count (autoscaler-driven)

	inFlight  atomic.Int64 // requests in flight at the proxy right now
	completed atomic.Int64 // lifetime completions, sampled for throughput

	winPeak    atomic.Int64 // peak in flight this frame
	winSuccess atomic.Int64 // successes this frame
	winReject  atomic.Int64 // rejects this frame
}

func newEngine() *engine {
	e := &engine{}
	e.latency.Store(50)
	e.replicas.Store(1)
	e.scaleMode.Store(scaleCPU)
	return e
}

func (e *engine) capacity() int64 { return e.replicas.Load() * limitPerReplica }

// generate is the open-loop load: a fixed batch of arrivals every 10ms, forever.
func (e *engine) generate() {
	perTick := ratePerSec * 10 / 1000
	t := time.NewTicker(10 * time.Millisecond)
	defer t.Stop()
	for range t.C {
		for i := 0; i < perTick; i++ {
			go e.serve()
		}
	}
}

// serve is one request's whole life. Admit only if in flight is below the live
// capacity; otherwise it is rejected at the door. Admitted requests sleep for the
// current latency (the service's work) and then free their slot.
func (e *engine) serve() {
	for {
		n := e.inFlight.Load()
		if n >= e.capacity() {
			e.winReject.Add(1)
			return // rejected
		}
		if e.inFlight.CompareAndSwap(n, n+1) {
			updatePeak(&e.winPeak, n+1)
			break
		}
	}
	time.Sleep(time.Duration(e.latency.Load()) * time.Millisecond)
	e.inFlight.Add(-1)
	e.completed.Add(1)
	e.winSuccess.Add(1)
}

// autoscale is the control loop: every controlInterval it reads the selected
// signal and resizes replicas with the standard HPA rule. On CPU it never fires,
// because CPU only falls as latency rises; on concurrency it tracks the load.
func (e *engine) autoscale() {
	t := time.NewTicker(controlInterval)
	defer t.Stop()
	lastCompleted := e.completed.Load()
	lastTime := time.Now()
	for now := range t.C {
		c := e.completed.Load()
		dt := now.Sub(lastTime).Seconds()
		lastTime = now
		thru := 0.0
		if dt > 0 {
			thru = float64(c-lastCompleted) / dt
		}
		lastCompleted = c

		r := e.replicas.Load()
		switch e.scaleMode.Load() {
		case scaleCPU:
			cpu := clampCPU(thru / float64(ratePerSec) * cpuBaseline)
			setReplicas(&e.replicas, int64(ceilf(float64(r)*cpu/targetCPU)))
		case scaleConc:
			util := float64(e.inFlight.Load()) / float64(r*limitPerReplica) * 100
			setReplicas(&e.replicas, int64(ceilf(float64(r)*util/targetUtil)))
		case scaleNone:
			// ceiling never moves
		}
	}
}

// snapshot reads the counters for one frame and resets the windowed ones.
func (e *engine) snapshot() (inFlight, peak int64, cpu, rejectRate float64) {
	inFlight = e.inFlight.Load()
	peak = e.winPeak.Swap(inFlight)
	succ := e.winSuccess.Swap(0)
	rej := e.winReject.Swap(0)
	// Throughput over the frame -> modeled CPU.
	thru := float64(succ) / refresh.Seconds()
	cpu = clampCPU(thru / float64(ratePerSec) * cpuBaseline)
	if succ+rej > 0 {
		rejectRate = float64(rej) / float64(succ+rej) * 100
	}
	return inFlight, peak, cpu, rejectRate
}

func updatePeak(peak *atomic.Int64, v int64) {
	for {
		old := peak.Load()
		if v <= old || peak.CompareAndSwap(old, v) {
			return
		}
	}
}

func setReplicas(r *atomic.Int64, desired int64) {
	if desired < 1 {
		desired = 1
	}
	if desired > maxReplicas {
		desired = maxReplicas
	}
	r.Store(desired)
}

func clampCPU(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

func ceilf(v float64) float64 {
	i := float64(int64(v))
	if v > i {
		return i + 1
	}
	return i
}

// --- TUI ---

var (
	label = styles.Sub
	dim   = styles.Sub
	load  = lipgloss.NewStyle().Foreground(styles.Accent) // blue: offered load
	ok    = styles.OK                                     // green: healthy
	warn  = styles.Attn                                   // amber: at the ceiling
	bad   = styles.Err                                    // red: failing
)

type tickMsg time.Time

type model struct {
	e          *engine
	inFlight   int64
	peak       int64
	cpu        float64
	rejectRate float64
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
			m.e.latency.Store(clamp(m.e.latency.Load()+25, 5, 2000))
		case "down", "j":
			m.e.latency.Store(clamp(m.e.latency.Load()-25, 5, 2000))
		case "s", " ":
			m.e.scaleMode.Store((m.e.scaleMode.Load() + 1) % 3)
			m.e.replicas.Store(1) // reset so the next signal's effect is clean to watch
		}
	case tickMsg:
		m.inFlight, m.peak, m.cpu, m.rejectRate = m.e.snapshot()
		return m, tick()
	}
	return m, nil
}

func (m model) View() string {
	latency := m.e.latency.Load()
	replicas := m.e.replicas.Load()
	capacity := replicas * limitPerReplica
	mode := m.e.scaleMode.Load()
	offered := float64(ratePerSec) * float64(latency) / 1000

	var b strings.Builder
	b.WriteString(styles.Header(
		"3B: Bit By Byte   Chapter 3: When CPU Says Everything Is Fine",
		fmt.Sprintf("arrival %d/s (flat)  ·  limit %d/replica  ·  autoscaler: %s", ratePerSec, limitPerReplica, scaleName(mode))) + "\n\n")

	b.WriteString(dim.Render(fmt.Sprintf("Service latency: %dms   (concurrency the load implies: about %.0f)", latency, offered)) + "\n\n")

	// Offered concurrency and actual in-flight share one scale, so you can watch
	// in-flight pin at the ceiling while offered keeps climbing.
	b.WriteString(row("Offered conc  ", offered, scaleMaxConc, load,
		fmt.Sprintf("%.0f", offered)))
	b.WriteString(row("In-flight     ", float64(m.peak), scaleMaxConc, inflightStyle(m.peak, capacity),
		fmt.Sprintf("%d  (capacity %d)", m.peak, capacity)))
	b.WriteString("\n")
	b.WriteString(row("Service CPU   ", m.cpu, 100, ok, fmt.Sprintf("%.0f%%  (scale at %.0f%%)", m.cpu, targetCPU)))
	b.WriteString(row("Reject rate   ", m.rejectRate, 100, rateStyle(m.rejectRate), fmt.Sprintf("%.0f%%", m.rejectRate)))
	b.WriteString(row("Replicas      ", float64(replicas), maxReplicas, load, fmt.Sprintf("%d", replicas)))

	b.WriteString("\n")
	switch mode {
	case scaleCPU:
		if m.rejectRate >= 5 {
			b.WriteString(warn.Render("CPU is falling while requests are rejected. The autoscaler watches CPU,") + "\n")
			b.WriteString(warn.Render("so it sees a healthy number and never scales. Press s to fix the signal.") + "\n")
		} else {
			b.WriteString(dim.Render("Watching CPU. Raise the latency (up) until concurrency hits the limit.") + "\n")
		}
	case scaleConc:
		b.WriteString(ok.Render("Watching proxy concurrency. Replicas track the load; the rejects vanish.") + "\n")
	default:
		b.WriteString(dim.Render("No autoscaler: the ceiling never moves. Press s to give it a signal.") + "\n")
	}

	b.WriteString("\n")
	b.WriteString(dim.Render("up/down: latency +/-25ms   s: cycle autoscaler signal   q: quit"))
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

func inflightStyle(peak, capacity int64) lipgloss.Style {
	if peak >= capacity {
		return warn // pinned at the ceiling
	}
	return ok
}

func rateStyle(rate float64) lipgloss.Style {
	switch {
	case rate < 1:
		return ok
	case rate < 25:
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
	go e.autoscale()

	p := tea.NewProgram(model{e: e})
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
