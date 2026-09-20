// Command sharedbottleneck is the companion lab for 3B: Bit By Byte, Chapter 5,
// "The Road Everyone Shares."
//
// It reproduces the blast-radius mechanism from the August 17, 2026 GitHub
// incident. GitHub reported that the original failure cascaded until four HAProxy
// nodes exhausted their flow limits, which degraded the gateway authentication
// path and spread authentication latency and failures across nearly every
// surface: Issues, Pull Requests, the APIs, Actions, Copilot, SAML and OIDC
// sign-in, SCIM, Team Sync. Those features share almost no code. What they share
// is one road: before any of them does its own job, it has to authenticate. Here
// we build the smallest system where saturating that one shared road takes down
// everything behind it, then contain the damage with a bulkhead.
//
// The system is three independent endpoints behind one shared auth gate:
//
//   - /notes, /tasks, /profile are three separate handlers with nothing in
//     common. Each does a trivially fast slice of its own work (--handler-ms,
//     default 5ms) and shares no logic with the others.
//   - In front of all three sits an AUTH gate: a counting semaphore of
//     --auth-slots slots (default 30). Every request must pass auth before it
//     reaches its handler. Each auth check holds a slot for --auth-ms (default
//     20ms), so the gate's throughput ceiling is slots/auth-ms, about 1500
//     auth/s in total across all three endpoints. A request that cannot get a
//     slot within --auth-timeout-ms is rejected: an auth failure at the door.
//
// The load is open-loop, one flat generator per endpoint, so offered rates never
// back off. /notes and /tasks are light and innocent (--notes-rps, --tasks-rps,
// default 200 each). /profile floods (--profile-rps, default 3000). We run the
// same load under two arrangements:
//
//   - SHARED: all three endpoints authenticate through the ONE gate. /profile's
//     flood consumes the shared slots, so /notes and /tasks are rejected and
//     slowed too, even though their own handlers are perfect and their own load
//     is tiny. All three success rates fall together, in lockstep.
//   - BULKHEAD: each endpoint gets its OWN gate of auth-slots/3 slots. /profile
//     now saturates only its own lane and tanks, while /notes and /tasks stay
//     healthy. The blast radius is contained back to local.
//
// Everything in this lab is measured from the real run: offered rate, success
// rate, admitted throughput, and p50/p95 latency per endpoint. There is no
// modeled number.
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/moksheshd/experiments/3b-bit-by-byte/season-01-github-august-17/styles"
)

type config struct {
	authSlots   int           // total auth slots (shared: one pool; bulkhead: split three ways)
	authWork    time.Duration // how long each auth check holds a slot
	authTimeout time.Duration // how long a request waits for a slot before it is rejected
	handlerWork time.Duration // per-request work inside each endpoint's own handler
	notesRPS    int           // offered load for /notes (light, innocent)
	tasksRPS    int           // offered load for /tasks (light, innocent)
	profileRPS  int           // offered load for /profile (the flood)
	settle      time.Duration // let the pipeline fill before measuring
	window      time.Duration // measurement window after settling
	mode        string        // "shared" | "bulkhead" | "both"
	csvPath     string
}

// endpoint is one of the three independent surfaces and its offered load.
type endpoint struct {
	name string
	rps  int
}

// result holds the measured outcome for one endpoint in one scenario.
type result struct {
	name       string
	offered    int           // offered req/s (flat, open loop)
	successPct float64       // measured: share of requests that passed auth and completed
	admitted   float64       // measured: successful req/s
	p50        time.Duration // measured latency of successful requests
	p95        time.Duration // measured
}

const genInterval = 10 * time.Millisecond // load generator tick granularity

func main() {
	cfg := parseFlags()
	printHeader(cfg)

	eps := []endpoint{
		{"/notes", cfg.notesRPS},
		{"/tasks", cfg.tasksRPS},
		{"/profile", cfg.profileRPS},
	}

	switch cfg.mode {
	case "both":
		shared := runScenario(cfg, "shared", eps)
		bulk := runScenario(cfg, "bulkhead", eps)
		printScenario("SHARED pool: all three authenticate through one gate", shared)
		fmt.Println()
		printScenario("BULKHEAD: each endpoint gets its own gate", bulk)
		fmt.Println()
		printChart("Success rate per endpoint  SHARED pool", shared)
		fmt.Println()
		printChart("Success rate per endpoint  BULKHEAD", bulk)
		fmt.Println()
		printFooter(cfg, shared, bulk)
		if cfg.csvPath != "" {
			writeAndReport(cfg.csvPath, shared, bulk)
		}
	default:
		res := runScenario(cfg, cfg.mode, eps)
		printScenario("Scenario: "+cfg.mode, res)
		fmt.Println()
		printChart("Success rate per endpoint  "+cfg.mode, res)
		fmt.Println()
		if cfg.csvPath != "" {
			writeAndReport(cfg.csvPath, res, nil)
		}
	}
}

func parseFlags() config {
	authSlots := flag.Int("auth-slots", 30, "total auth slots: the shared gate's concurrency limit (bulkhead splits this three ways)")
	authMs := flag.Int("auth-ms", 20, "how long each auth check holds a slot, in ms (sets the gate's throughput: slots/auth-ms)")
	authTimeoutMs := flag.Int("auth-timeout-ms", 50, "how long a request waits for an auth slot before it is rejected, in ms")
	handlerMs := flag.Int("handler-ms", 5, "per-request work inside each endpoint's own handler, in ms (kept trivially fast on purpose)")
	notesRPS := flag.Int("notes-rps", 200, "offered load for /notes, in req/s (light and innocent)")
	tasksRPS := flag.Int("tasks-rps", 200, "offered load for /tasks, in req/s (light and innocent)")
	profileRPS := flag.Int("profile-rps", 3000, "offered load for /profile, in req/s (the flood)")
	settleMs := flag.Int("settle-ms", 1000, "time to let the pipeline fill before measuring, in ms")
	windowMs := flag.Int("window-ms", 3000, "measurement window after settling, in ms")
	mode := flag.String("mode", "both", "scenario: shared | bulkhead | both")
	csvPath := flag.String("csv", "", "optional path to write results as CSV")
	flag.Parse()

	if *authSlots < 3 {
		fmt.Fprintln(os.Stderr, "--auth-slots must be at least 3 so the bulkhead can give each endpoint a slot")
		os.Exit(1)
	}
	switch *mode {
	case "shared", "bulkhead", "both":
	default:
		fmt.Fprintf(os.Stderr, "invalid --mode %q: want shared | bulkhead | both\n", *mode)
		os.Exit(1)
	}

	return config{
		authSlots:   *authSlots,
		authWork:    time.Duration(*authMs) * time.Millisecond,
		authTimeout: time.Duration(*authTimeoutMs) * time.Millisecond,
		handlerWork: time.Duration(*handlerMs) * time.Millisecond,
		notesRPS:    *notesRPS,
		tasksRPS:    *tasksRPS,
		profileRPS:  *profileRPS,
		settle:      time.Duration(*settleMs) * time.Millisecond,
		window:      time.Duration(*windowMs) * time.Millisecond,
		mode:        *mode,
		csvPath:     *csvPath,
	}
}

// authGate is the shared road: a counting semaphore with a bounded wait. A
// request tries to acquire a slot within timeout. On success it holds the slot
// for the auth work, then releases it. On timeout it is rejected, exactly like a
// request that never gets past a saturated gateway auth path. This is the whole
// bottleneck: the handlers behind it are never the constraint.
type authGate struct {
	sem     chan struct{}
	timeout time.Duration
}

func newGate(slots int, timeout time.Duration) *authGate {
	return &authGate{sem: make(chan struct{}, slots), timeout: timeout}
}

// pass runs one auth check. It returns true if the request got a slot within the
// timeout and held it for the auth work, false if it gave up waiting (rejected).
func (g *authGate) pass(authWork time.Duration) bool {
	t := time.NewTimer(g.timeout)
	select {
	case g.sem <- struct{}{}:
		t.Stop()
		time.Sleep(authWork) // the auth check's work, holding the slot
		<-g.sem
		return true
	case <-t.C:
		return false // could not get a slot in time: auth failure at the door
	}
}

// meter collects one endpoint's measured outcomes during the window. Successes
// and rejects are atomic; latency samples go under a mutex, which is cheap at
// these rates and keeps the hot path off a shared counter.
type meter struct {
	success atomic.Int64
	reject  atomic.Int64
	mu      sync.Mutex
	lat     []time.Duration
}

// runScenario drives all three endpoints at their offered rates for the whole
// run and measures the steady state over the final window. In shared mode every
// endpoint uses one gate; in bulkhead mode each gets its own gate of
// auth-slots/3 slots.
func runScenario(cfg config, mode string, eps []endpoint) []result {
	gates := make(map[string]*authGate, len(eps))
	if mode == "shared" {
		shared := newGate(cfg.authSlots, cfg.authTimeout)
		for _, ep := range eps {
			gates[ep.name] = shared
		}
	} else {
		per := cfg.authSlots / len(eps)
		if per < 1 {
			per = 1
		}
		for _, ep := range eps {
			gates[ep.name] = newGate(per, cfg.authTimeout)
		}
	}

	meters := make(map[string]*meter, len(eps))
	for _, ep := range eps {
		meters[ep.name] = &meter{}
	}

	var measuring atomic.Bool
	stop := make(chan struct{})
	var wg sync.WaitGroup

	// One open-loop generator per endpoint: a fixed batch of arrivals every
	// genInterval, so each offered rate stays flat no matter how the gate behaves.
	for _, ep := range eps {
		ep, gate, m := ep, gates[ep.name], meters[ep.name]
		perTick := ep.rps * int(genInterval/time.Millisecond) / 1000
		if perTick < 1 {
			perTick = 1
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			t := time.NewTicker(genInterval)
			defer t.Stop()
			for {
				select {
				case <-stop:
					return
				case <-t.C:
					for i := 0; i < perTick; i++ {
						wg.Add(1)
						go func() {
							defer wg.Done()
							handle(cfg, gate, m, &measuring)
						}()
					}
				}
			}
		}()
	}

	// Let the pipeline fill, then measure the steady state.
	time.Sleep(cfg.settle)
	measuring.Store(true)
	time.Sleep(cfg.window)
	measuring.Store(false)
	close(stop)
	wg.Wait()

	out := make([]result, len(eps))
	for i, ep := range eps {
		out[i] = summarize(ep, meters[ep.name], cfg.window)
	}
	return out
}

// handle is one request's whole life: authenticate at the shared gate, and only
// if that succeeds run the endpoint's own (trivially fast) handler. The handler
// is never the bottleneck; the gate is.
func handle(cfg config, gate *authGate, m *meter, measuring *atomic.Bool) {
	start := time.Now()
	ok := gate.pass(cfg.authWork)
	meas := measuring.Load()
	if !ok {
		if meas {
			m.reject.Add(1)
		}
		return
	}
	time.Sleep(cfg.handlerWork) // the endpoint's own work, healthy and quick
	if meas {
		m.success.Add(1)
		d := time.Since(start)
		m.mu.Lock()
		m.lat = append(m.lat, d)
		m.mu.Unlock()
	}
}

func summarize(ep endpoint, m *meter, window time.Duration) result {
	succ := m.success.Load()
	rej := m.reject.Load()
	successPct := 0.0
	if succ+rej > 0 {
		successPct = float64(succ) / float64(succ+rej) * 100
	}
	sort.Slice(m.lat, func(i, j int) bool { return m.lat[i] < m.lat[j] })
	return result{
		name:       ep.name,
		offered:    ep.rps,
		successPct: successPct,
		admitted:   float64(succ) / window.Seconds(),
		p50:        percentile(m.lat, 0.50),
		p95:        percentile(m.lat, 0.95),
	}
}

func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	i := int(p * float64(len(sorted)))
	if i >= len(sorted) {
		i = len(sorted) - 1
	}
	return sorted[i]
}

func printHeader(cfg config) {
	meta := fmt.Sprintf("auth gate %d slots  ·  auth %s/check  ·  handlers %s  ·  /profile floods at %d/s",
		cfg.authSlots, shortDur(cfg.authWork), shortDur(cfg.handlerWork), cfg.profileRPS)
	fmt.Println()
	fmt.Println(styles.Header("3B: Bit By Byte   Chapter 5: The Road Everyone Shares", meta))
	fmt.Println()
	fmt.Println(styles.Sub.Render("Three independent endpoints (/notes, /tasks, /profile) sit behind one"))
	fmt.Println(styles.Sub.Render("shared auth gate. /notes and /tasks are light; /profile floods. Watch the"))
	fmt.Println(styles.Sub.Render("flood on the shared road drag the two innocent endpoints down with it,"))
	fmt.Println(styles.Sub.Render("then give each its own lane and watch the blast radius shrink to local."))
	fmt.Println()
}

func printScenario(title string, rs []result) {
	fmt.Println(styles.Bold.Render(title))
	fmt.Println(styles.Bold.Render(
		pad("Endpoint", 11) + pad("Offered", 9) + pad("Success", 10) +
			pad("Admit", 9) + pad("p50", 9) + "p95"))
	fmt.Println(styles.Bold.Render(
		pad("", 11) + pad("req/s", 9) + pad("rate", 10) +
			pad("req/s", 9) + pad("lat", 9) + "lat"))
	fmt.Println(styles.Rule(58))
	for _, r := range rs {
		row := pad(r.name, 11) +
			pad(fmt.Sprintf("%d", r.offered), 9) +
			pad(fmt.Sprintf("%.0f%%", r.successPct), 10) +
			pad(fmt.Sprintf("%.0f", r.admitted), 9) +
			pad(shortDur(r.p50), 9) +
			shortDur(r.p95)
		// Green while the endpoint keeps its promises, red once it sheds load.
		if r.successPct >= 90 {
			fmt.Println(styles.OK.Render(row))
		} else {
			fmt.Println(styles.Err.Render(row))
		}
	}
}

// printChart draws one horizontal success-rate bar per endpoint, scaled 0-100,
// so the lockstep collapse (shared) and the contained damage (bulkhead) are
// visible at a glance.
func printChart(title string, rs []result) {
	fmt.Println(styles.Bold.Render(title))
	const width = 40
	for _, r := range rs {
		v := r.successPct
		if v < 0 {
			v = 0
		}
		if v > 100 {
			v = 100
		}
		filled := int(v / 100 * width)
		style := styles.OK
		if v < 90 {
			style = styles.Err
		}
		bar := style.Render(strings.Repeat("█", filled)) +
			styles.Sub.Render(strings.Repeat("·", width-filled))
		fmt.Printf("  %s  %s  %s\n",
			styles.Sub.Render(pad(r.name, 9)),
			bar,
			styles.Bold.Render(fmt.Sprintf("%.0f%%", v)))
	}
}

func printFooter(cfg config, shared, bulk []result) {
	fmt.Println(styles.Rule(58))
	sn := byName(shared)
	bn := byName(bulk)
	fmt.Println(styles.Sub.Render(fmt.Sprintf(
		"/profile floods at %d/s against a gate that clears about %d auth/s.",
		cfg.profileRPS, int(float64(cfg.authSlots)/cfg.authWork.Seconds()))))
	fmt.Println()
	fmt.Println(styles.Sub.Render(fmt.Sprintf(
		"Shared: /notes %.0f%% and /tasks %.0f%% succeed, though each offers only",
		sn["/notes"].successPct, sn["/tasks"].successPct)))
	fmt.Println(styles.Sub.Render(fmt.Sprintf(
		"%d/s. They fall in lockstep with /profile (%.0f%%): three tenants of one road.",
		cfg.notesRPS, sn["/profile"].successPct)))
	fmt.Println()
	fmt.Println(styles.Sub.Render(fmt.Sprintf(
		"Bulkhead: /notes %.0f%% and /tasks %.0f%% stay healthy while /profile still",
		bn["/notes"].successPct, bn["/tasks"].successPct)))
	fmt.Println(styles.Sub.Render(fmt.Sprintf(
		"tanks (%.0f%%). Same flood, same handlers. Only the blast radius changed.",
		bn["/profile"].successPct)))
	fmt.Println()
	fmt.Println(styles.Sub.Render("Try it: --profile-rps reshapes the flood; --auth-slots resizes the road;"))
	fmt.Println(styles.Sub.Render("--mode shared or --mode bulkhead runs just one arrangement."))
	fmt.Println()
}

func byName(rs []result) map[string]result {
	m := make(map[string]result, len(rs))
	for _, r := range rs {
		m[r.name] = r
	}
	return m
}

func writeAndReport(path string, shared, bulk []result) {
	if err := writeCSV(path, shared, bulk); err != nil {
		fmt.Fprintln(os.Stderr, "could not write CSV:", err)
		os.Exit(1)
	}
	fmt.Println(styles.Sub.Render("Wrote results to " + path))
}

func writeCSV(path string, shared, bulk []result) error {
	var b strings.Builder
	b.WriteString("scenario,endpoint,offered_rps,success_pct,admitted_rps,p50_ms,p95_ms\n")
	writeRows := func(scenario string, rs []result) {
		for _, r := range rs {
			b.WriteString(fmt.Sprintf("%s,%s,%d,%.1f,%.0f,%.1f,%.1f\n",
				scenario, r.name, r.offered, r.successPct, r.admitted, ms(r.p50), ms(r.p95)))
		}
	}
	writeRows("shared", shared)
	if bulk != nil {
		writeRows("bulkhead", bulk)
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func ms(d time.Duration) float64 { return float64(d.Microseconds()) / 1000 }

func shortDur(d time.Duration) string {
	if d == 0 {
		return "-"
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	return fmt.Sprintf("%dms", d.Milliseconds())
}

func pad(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
