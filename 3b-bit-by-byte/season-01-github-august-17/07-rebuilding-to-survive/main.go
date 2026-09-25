// Command rebuild is the season capstone lab for 3B: Bit By Byte, Chapter 7,
// "The Fix Is Not the Lesson."
//
// The whole season broke one Client -> Proxy -> Service chain on purpose, one
// link at a time. This lab walks the same chain twice under the same offered
// load: once FRAGILE (the incident's failure mechanisms, in a small model) and
// once RESILIENT (every mitigation applied), and prints the two outcomes side by
// side. The argument it makes physical: the load was not the whole problem, the
// couplings were what let it cascade.
//
// Each link is a small, self-contained, measured experiment that reuses the
// mechanism from its own chapter:
//
//   - Capacity, the autoscaler signal (Chapter 3). Fragile scales on CPU and
//     never fires, so requests wait out the timeout and fail; resilient scales
//     on proxy concurrency and adds replicas.
//   - The queue, backpressure (Chapters 2 and 4). Replicas are held at one in
//     both stacks, so backpressure is the only difference. Fragile accepts and
//     waits, so served latency climbs toward the one-second timeout; resilient
//     rejects fast what it cannot serve, so served latency stays flat.
//   - Retries, a budget (Chapter 4). Fragile retries immediately with no budget
//     (up to five tries) and amplifies the load; resilient backs off with jitter
//     and spends from a budget that holds retries to 10% of requests.
//   - The shared road, a bulkhead (Chapter 5). Fragile runs one shared auth pool,
//     so a flood on one tenant starves an innocent one; resilient gives the
//     innocent tenant its own lane.
//   - The client, discipline (Chapter 6). Fragile clients reissue on a timeout
//     and manufacture a tenfold storm; resilient clients use a sane timeout and a
//     budget, so the load stays near baseline.
//
// Everything is measured from the real run except one clearly-labelled number:
// service CPU in the capacity link is modeled from throughput, exactly as in
// Chapter 3. The point there is directional, not numeric.
package main

import (
	"flag"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/moksheshd/experiments/3b-bit-by-byte/season-01-github-august-17/styles"
)

// metrics is one full walk of the chain under one configuration.
type metrics struct {
	replicas    int64         // capacity link: replicas the autoscaler settled on
	failedPct   float64       // capacity link: requests that failed under the load
	p95         time.Duration // queue link: p95 latency of served requests
	retryAmpl   float64       // retries link: attempts reaching the service, x offered
	innocentPct float64       // shared-road link: success rate of the innocent tenant
	clientAmpl  float64       // client link: fleet load reaching the endpoint, x baseline
}

func main() {
	mode := flag.String("mode", "both", "which stack to run: both | fragile | resilient")
	csvPath := flag.String("csv", "", "optional path to write results as CSV")
	flag.Parse()

	printHeader()

	var fragile, resilient metrics
	switch *mode {
	case "both":
		fragile = runStack(false)
		resilient = runStack(true)
		printTable(&fragile, &resilient)
		fmt.Println()
		printFooter(fragile, resilient)
	case "fragile":
		fragile = runStack(false)
		printTable(&fragile, nil)
	case "resilient":
		resilient = runStack(true)
		printTable(nil, &resilient)
	default:
		fmt.Fprintf(os.Stderr, "invalid --mode %q: want both | fragile | resilient\n", *mode)
		os.Exit(1)
	}

	if *csvPath != "" {
		if err := writeCSV(*csvPath, &fragile, &resilient); err != nil {
			fmt.Fprintln(os.Stderr, "could not write CSV:", err)
			os.Exit(1)
		}
		fmt.Println(styles.Sub.Render("Wrote results to " + *csvPath))
	}
}

// runStack walks the whole chain once. resilient=false is the incident; true is
// the rebuild. Same offered load feeds both.
func runStack(resilient bool) metrics {
	replicas, failed := linkCapacity(resilient)
	return metrics{
		replicas:    replicas,
		failedPct:   failed,
		p95:         linkQueue(resilient),
		retryAmpl:   linkRetries(resilient),
		innocentPct: linkSharedRoad(resilient),
		clientAmpl:  linkClient(resilient),
	}
}

// --- Link 1 and 2: capacity (autoscaler) and the queue (backpressure) ---

// linkCapacity runs an open-loop load at a fixed rate into a proxy whose service
// is slow enough to push concurrency past the limit. Fragile scales on modeled
// CPU (which only falls, so it never fires) and does not shed, so requests wait
// out the timeout and fail; resilient scales on concurrency and sheds fast.
// Returns the replicas it settled on and the percent of requests that failed.
func linkCapacity(resilient bool) (int64, float64) {
	const (
		rate      = 2000
		limit     = 100
		latency   = 200 * time.Millisecond
		settle    = 1200 * time.Millisecond
		window    = 1200 * time.Millisecond
		cpuBase   = 40.0
		targetCPU = 70.0
		targetU   = 80.0
		maxRepl   = 20
	)
	var replicas atomic.Int64
	replicas.Store(1)
	served, failed, completed := runProxyLoad(resilient, rate, limit, latency, settle, window, &replicas,
		cpuBase, targetCPU, targetU, maxRepl, nil)
	total := served + failed
	pct := 0.0
	if total > 0 {
		pct = float64(failed) / float64(total) * 100
	}
	_ = completed
	return replicas.Load(), pct
}

// linkQueue reuses the same proxy load but records latency, to show what
// backpressure buys. Replicas are held at one in both stacks, so the autoscaler
// cannot take the credit. Fragile accepts and waits (served latency climbs toward
// the timeout plus one service time); resilient rejects fast what it cannot serve
// (served latency stays at one service time).
func linkQueue(resilient bool) time.Duration {
	const (
		rate    = 2000
		limit   = 100
		latency = 200 * time.Millisecond
		settle  = 1200 * time.Millisecond
		window  = 1200 * time.Millisecond
		maxRepl = 1 // held at one: backpressure is the only difference
	)
	var replicas atomic.Int64
	replicas.Store(1)
	var lats latencies
	runProxyLoad(resilient, rate, limit, latency, settle, window, &replicas,
		40.0, 70.0, 80.0, maxRepl, &lats)
	return lats.percentile(0.95)
}

// runProxyLoad is the shared engine for the capacity and queue links. It drives a
// flat open-loop rate through a proxy with a live capacity (replicas x limit) and
// an autoscaler, into a service that sleeps `latency`. Returns served and failed
// counts measured in the window (a failure is a fast reject for resilient, a
// timed-out wait for fragile), and optionally records served latencies.
func runProxyLoad(resilient bool, rate, limit int, latency, settle, window time.Duration,
	replicas *atomic.Int64, cpuBase, targetCPU, targetU float64, maxRepl int, lats *latencies) (served, failed, completed int64) {

	const clientTimeout = 1000 * time.Millisecond
	var (
		inFlight   atomic.Int64
		servedA    atomic.Int64
		failedA    atomic.Int64
		completedA atomic.Int64
	)
	var measuring atomic.Bool
	var mu sync.Mutex // guards lats

	capacity := func() int64 { return replicas.Load() * int64(limit) }

	serve := func() (ok bool, waited time.Duration) {
		start := time.Now()
		for {
			n := inFlight.Load()
			if n < capacity() {
				if inFlight.CompareAndSwap(n, n+1) {
					break // admitted
				}
				continue
			}
			// Full. Resilient sheds immediately; fragile waits for a slot.
			if resilient {
				return false, 0
			}
			if time.Since(start) > clientTimeout {
				return false, 0 // waited too long: give up
			}
			time.Sleep(2 * time.Millisecond)
		}
		waited = time.Since(start)
		time.Sleep(latency)
		inFlight.Add(-1)
		completedA.Add(1)
		return true, waited + latency
	}

	stop := make(chan struct{})
	var wg sync.WaitGroup
	perTick := rate * 10 / 1000
	wg.Add(1)
	go func() {
		defer wg.Done()
		t := time.NewTicker(10 * time.Millisecond)
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
						meas := measuring.Load()
						ok, total := serve()
						if !meas {
							return
						}
						if ok {
							servedA.Add(1)
							if lats != nil {
								mu.Lock()
								lats.v = append(lats.v, total)
								mu.Unlock()
							}
						} else {
							failedA.Add(1)
						}
					}()
				}
			}
		}
	}()

	// Autoscaler.
	wg.Add(1)
	go func() {
		defer wg.Done()
		t := time.NewTicker(200 * time.Millisecond)
		defer t.Stop()
		lastCompleted := completedA.Load()
		lastTime := time.Now()
		for {
			select {
			case <-stop:
				return
			case now := <-t.C:
				c := completedA.Load()
				dt := now.Sub(lastTime).Seconds()
				lastTime = now
				thru := 0.0
				if dt > 0 {
					thru = float64(c-lastCompleted) / dt
				}
				lastCompleted = c
				r := replicas.Load()
				if resilient {
					util := float64(inFlight.Load()) / float64(r*int64(limit)) * 100
					setReplicas(replicas, int64(math.Ceil(float64(r)*util/targetU)), maxRepl)
				} else {
					cpu := clampCPU(thru / float64(rate) * cpuBase)
					setReplicas(replicas, int64(math.Ceil(float64(r)*cpu/targetCPU)), maxRepl)
				}
			}
		}
	}()

	time.Sleep(settle)
	measuring.Store(true)
	time.Sleep(window)
	measuring.Store(false)
	close(stop)
	wg.Wait()
	return servedA.Load(), failedA.Load(), completedA.Load()
}

// --- Link 3: retries (budget) ---

// linkRetries drives an offered rate above the service capacity and lets clients
// retry. Fragile retries immediately, up to maxTry tries, with no budget;
// resilient backs off with jitter and spends from a retry budget that every
// request refills, so retries cannot exceed 10% of requests. Returns attempts
// reaching the service, as a multiple of offered (the amplification).
func linkRetries(resilient bool) float64 {
	const (
		offered   = 2000
		limit     = 50
		serviceMs = 40 * time.Millisecond
		maxTry    = 5
		settle    = 800 * time.Millisecond
		window    = 1500 * time.Millisecond
		budget    = 0.10 // retries may not exceed 10% of requests
		unit      = 1000 // budget tokens are counted in thousandths of a retry
	)
	var (
		inFlight atomic.Int64
		attempts atomic.Int64
		offeredA atomic.Int64
		tokens   atomic.Int64
	)
	var measuring atomic.Bool

	serveOnce := func() bool {
		for {
			n := inFlight.Load()
			if n >= int64(limit) {
				return false // fast reject (backpressure)
			}
			if inFlight.CompareAndSwap(n, n+1) {
				break
			}
		}
		time.Sleep(serviceMs)
		inFlight.Add(-1)
		return true
	}

	logical := func(rng *rand.Rand) {
		meas := measuring.Load()
		if meas {
			offeredA.Add(1)
		}
		if resilient {
			tokens.Add(int64(budget * unit)) // every request earns a tenth of a retry
		}
		wait := 50 * time.Millisecond
		for try := 0; try < maxTry; try++ {
			if try > 0 && resilient {
				if tokens.Add(-unit) < 0 {
					tokens.Add(unit) // not enough budget: put it back
					return           // budget spent: fail fast
				}
			}
			if meas {
				attempts.Add(1)
			}
			if serveOnce() {
				return
			}
			if resilient {
				time.Sleep(time.Duration(float64(wait) * (0.5 + rng.Float64())))
				wait *= 2
			}
			// fragile: retry immediately
		}
	}

	stop := make(chan struct{})
	var wg sync.WaitGroup
	perTick := offered * 10 / 1000
	var seeds atomic.Int64
	wg.Add(1)
	go func() {
		defer wg.Done()
		t := time.NewTicker(10 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				for i := 0; i < perTick; i++ {
					wg.Add(1)
					go func(seed int64) {
						defer wg.Done()
						logical(rand.New(rand.NewSource(seed)))
					}(seeds.Add(1))
				}
			}
		}
	}()

	time.Sleep(settle)
	measuring.Store(true)
	time.Sleep(window)
	measuring.Store(false)
	close(stop)
	wg.Wait()

	off := offeredA.Load()
	if off == 0 {
		return 0
	}
	return float64(attempts.Load()) / float64(off)
}

// --- Link 4: the shared road (bulkhead) ---

// linkSharedRoad puts an innocent, light tenant and a flooding tenant behind an
// auth gate. Fragile shares one pool, so the flood starves the innocent tenant;
// resilient gives the innocent tenant its own pool. Returns the innocent tenant's
// success rate.
func linkSharedRoad(resilient bool) float64 {
	const (
		totalSlots  = 30
		authMs      = 20 * time.Millisecond
		waitMs      = 50 * time.Millisecond
		innocentRPS = 200
		floodRPS    = 3000
		settle      = 700 * time.Millisecond
		window      = 1200 * time.Millisecond
	)
	// Shared: both tenants draw from one pool. Bulkhead: split the pool in two.
	sharedPool := makePool(totalSlots)
	innocentPool := sharedPool
	floodPool := sharedPool
	if resilient {
		innocentPool = makePool(totalSlots / 2)
		floodPool = makePool(totalSlots / 2)
	}

	var innocentOK, innocentTotal atomic.Int64
	var measuring atomic.Bool

	auth := func(pool chan struct{}) bool {
		deadline := time.Now().Add(waitMs)
		for {
			select {
			case pool <- struct{}{}:
				time.Sleep(authMs)
				<-pool
				return true
			default:
				if time.Now().After(deadline) {
					return false
				}
				time.Sleep(time.Millisecond)
			}
		}
	}

	stop := make(chan struct{})
	var wg sync.WaitGroup
	runTenant := func(rps int, pool chan struct{}, isInnocent bool) {
		perTick := rps * 10 / 1000
		if perTick < 1 {
			perTick = 1
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			t := time.NewTicker(10 * time.Millisecond)
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
							meas := measuring.Load()
							ok := auth(pool)
							if meas && isInnocent {
								innocentTotal.Add(1)
								if ok {
									innocentOK.Add(1)
								}
							}
						}()
					}
				}
			}
		}()
	}
	runTenant(innocentRPS, innocentPool, true)
	runTenant(floodRPS, floodPool, false)

	time.Sleep(settle)
	measuring.Store(true)
	time.Sleep(window)
	measuring.Store(false)
	close(stop)
	wg.Wait()

	tot := innocentTotal.Load()
	if tot == 0 {
		return 0
	}
	return float64(innocentOK.Load()) / float64(tot) * 100
}

func makePool(n int) chan struct{} { return make(chan struct{}, n) }

// --- Link 5: the client (discipline) ---

// linkClient runs a client fleet against a slow endpoint. Fragile clients reissue
// on an aggressive timeout and manufacture a storm; resilient clients use a sane
// timeout and a budget. Returns the load reaching the endpoint, x baseline.
func linkClient(resilient bool) float64 {
	const (
		clients  = 500
		interval = 500 * time.Millisecond
		slowResp = 500 * time.Millisecond
		timeout  = 50 * time.Millisecond
		saneTO   = 800 * time.Millisecond
		maxTry   = 40
		settle   = 800 * time.Millisecond
		window   = 1500 * time.Millisecond
	)
	baseline := float64(clients) / interval.Seconds()
	to := timeout
	if resilient {
		to = saneTO
	}

	var attempts atomic.Int64
	var measuring atomic.Bool

	logical := func() {
		meas := measuring.Load()
		done := make(chan struct{})
		got := make(chan struct{}, 1)
		var firer sync.WaitGroup
		firer.Add(1)
		go func() {
			defer firer.Done()
			tries := 0
			tryCap := maxTry
			if resilient {
				tryCap = 2 // budget: one try plus one retry, then fail fast
			}
			for tries < tryCap {
				select {
				case <-done:
					return
				default:
				}
				tries++
				if meas {
					attempts.Add(1)
				}
				go func() {
					time.Sleep(slowResp)
					select {
					case got <- struct{}{}:
					default:
					}
				}()
				select {
				case <-done:
					return
				case <-time.After(to):
				}
			}
		}()
		select {
		case <-got:
		case <-time.After(slowResp + 2*saneTO):
		}
		close(done)
		firer.Wait()
	}

	stop := make(chan struct{})
	var fleet sync.WaitGroup
	for i := 0; i < clients; i++ {
		fleet.Add(1)
		go func() {
			defer fleet.Done()
			t := time.NewTicker(interval)
			defer t.Stop()
			for {
				select {
				case <-stop:
					return
				case <-t.C:
					go logical()
				}
			}
		}()
	}

	time.Sleep(settle)
	measuring.Store(true)
	time.Sleep(window)
	measuring.Store(false)
	time.Sleep(slowResp + 300*time.Millisecond) // drain sampled requests
	close(stop)
	fleet.Wait()

	return (float64(attempts.Load()) / window.Seconds()) / baseline
}

// --- shared helpers ---

type latencies struct{ v []time.Duration }

func (l *latencies) percentile(p float64) time.Duration {
	if len(l.v) == 0 {
		return 0
	}
	sort.Slice(l.v, func(i, j int) bool { return l.v[i] < l.v[j] })
	i := int(p * float64(len(l.v)))
	if i >= len(l.v) {
		i = len(l.v) - 1
	}
	return l.v[i]
}

func setReplicas(r *atomic.Int64, desired int64, maxR int) {
	if desired < 1 {
		desired = 1
	}
	if desired > int64(maxR) {
		desired = int64(maxR)
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

func printHeader() {
	fmt.Println()
	fmt.Println(styles.Header("3B: Bit By Byte   Chapter 7: The Fix Is Not the Lesson",
		"the same chain, the same load, broken then rebuilt"))
	fmt.Println()
	fmt.Println(styles.Sub.Render("Walk the failure chain twice under one offered load: FRAGILE (the"))
	fmt.Println(styles.Sub.Render("incident's failure mechanisms, in a small model) and RESILIENT (every"))
	fmt.Println(styles.Sub.Render("mitigation applied). Same load, different shape: the second one bends and"))
	fmt.Println(styles.Sub.Render("stays up instead of folding link by link."))
	fmt.Println()
}

func printTable(f, r *metrics) {
	fmt.Println(styles.Bold.Render(
		pad("Link (chapter)", 30) + pad("What you watch", 20) + pad("Fragile", 12) + "Resilient"))
	fmt.Println(styles.Rule(74))

	rows := []struct {
		link   string
		metric string
		fv, rv func(m *metrics) string
		bad    func(m *metrics) bool // is the fragile value the failing one?
	}{
		{"Capacity: autoscaler (ch3)", "proxy replicas",
			func(m *metrics) string { return fmt.Sprintf("%d", m.replicas) },
			func(m *metrics) string { return fmt.Sprintf("%d", m.replicas) },
			func(m *metrics) bool { return m.replicas <= 1 }},
		{"Capacity: failures (ch3)", "requests failed",
			func(m *metrics) string { return fmt.Sprintf("%.0f%%", m.failedPct) },
			func(m *metrics) string { return fmt.Sprintf("%.0f%%", m.failedPct) },
			func(m *metrics) bool { return m.failedPct >= 5 }},
		{"The queue: backpressure (ch2)", "p95 latency",
			func(m *metrics) string { return shortDur(m.p95) },
			func(m *metrics) string { return shortDur(m.p95) },
			func(m *metrics) bool { return m.p95 >= 500*time.Millisecond }},
		{"Retries: budget (ch4)", "load amplification",
			func(m *metrics) string { return fmt.Sprintf("%.1fx", m.retryAmpl) },
			func(m *metrics) string { return fmt.Sprintf("%.1fx", m.retryAmpl) },
			func(m *metrics) bool { return m.retryAmpl >= 1.5 }},
		{"Shared road: bulkhead (ch5)", "innocent success",
			func(m *metrics) string { return fmt.Sprintf("%.0f%%", m.innocentPct) },
			func(m *metrics) string { return fmt.Sprintf("%.0f%%", m.innocentPct) },
			func(m *metrics) bool { return m.innocentPct < 90 }},
		{"The client: discipline (ch6)", "fleet amplification",
			func(m *metrics) string { return fmt.Sprintf("%.1fx", m.clientAmpl) },
			func(m *metrics) string { return fmt.Sprintf("%.1fx", m.clientAmpl) },
			func(m *metrics) bool { return m.clientAmpl >= 1.5 }},
	}

	for _, row := range rows {
		fCol, rCol := "-", "-"
		if f != nil {
			fCol = row.fv(f)
		}
		if r != nil {
			rCol = row.rv(r)
		}
		line := pad(row.link, 30) + pad(row.metric, 20) + pad(fCol, 12) + rCol
		if f != nil && row.bad(f) {
			fmt.Println(styles.Err.Render(line))
		} else {
			fmt.Println(styles.OK.Render(line))
		}
	}
}

func printFooter(f, r metrics) {
	fmt.Println(styles.Rule(74))
	fmt.Println(styles.Sub.Render("Same offered load fed both stacks. Fragile folds: the autoscaler holds"))
	fmt.Println(styles.Sub.Render(fmt.Sprintf(
		"at %d replica and %.0f%% of requests fail, latency climbs to %s, retries",
		f.replicas, f.failedPct, shortDur(f.p95))))
	fmt.Println(styles.Sub.Render(fmt.Sprintf(
		"add %.1fx, the innocent path drops to %.0f%%, and the client fleet runs %.1fx.",
		f.retryAmpl, f.innocentPct, f.clientAmpl)))
	fmt.Println(styles.Sub.Render(fmt.Sprintf(
		"Resilient bends: %d replicas, %.0f%% failed, %s p95, %.1fx retries, %.0f%%",
		r.replicas, r.failedPct, shortDur(r.p95), r.retryAmpl, r.innocentPct)))
	fmt.Println(styles.Sub.Render(fmt.Sprintf(
		"innocent, %.1fx client. The load was not the whole problem. The couplings",
		r.clientAmpl)))
	fmt.Println(styles.Sub.Render("were what let it cascade."))
	fmt.Println()
}

func writeCSV(path string, f, r *metrics) error {
	var b strings.Builder
	b.WriteString("stack,replicas,failed_pct,p95_ms,retry_amplification_x,innocent_success_pct,client_amplification_x\n")
	row := func(name string, m *metrics) {
		if m == nil {
			return
		}
		b.WriteString(fmt.Sprintf("%s,%d,%.1f,%d,%.2f,%.1f,%.2f\n",
			name, m.replicas, m.failedPct, m.p95.Milliseconds(), m.retryAmpl, m.innocentPct, m.clientAmpl))
	}
	row("fragile", f)
	row("resilient", r)
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

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
