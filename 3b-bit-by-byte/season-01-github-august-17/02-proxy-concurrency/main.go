// Command proxyconc is the companion lab for 3B: Bit By Byte, Chapter 2,
// "The Proxy Beside Your Application."
//
// It reproduces one mechanism from the August 17, 2026 GitHub incident: a
// proxy with a concurrency limit saturates while the service behind it stays
// comfortably idle. Client pressure climbs, the client sees failures, and yet
// the service's own load is bounded by the proxy and never rises past the
// limit. Two graphs, opposite stories, same system.
//
// The system is a real, in-process Client -> Proxy -> Service:
//
//   - Service: a function that does a fixed slice of work (a real time.Sleep)
//     per request. It tracks how many requests are in flight inside it.
//   - Proxy: a counting semaphore (a buffered channel of size --limit) that
//     enforces a hard concurrency limit before forwarding to the service. This
//     is the "fake it with a semaphore" from the essay: the same constraint
//     Envoy enforces in real life, modeled with the simplest thing that has the
//     same shape.
//   - Client: a pool of goroutines that hammer the proxy in a closed loop for a
//     fixed window, at a chosen concurrency.
//
// We run the client at several concurrency levels and, for each, measure what
// the client sees (success rate, latency) and what the service sees (peak
// in-flight). Everything except one clearly-labelled number is measured from
// the real run. That one number is service CPU, which we model transparently:
// see --cpu-at-limit below.
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/moksheshd/experiments/3b-bit-by-byte/season-01-github-august-17/styles"
)

type config struct {
	limit        int           // proxy concurrency limit
	steps        []int         // client concurrency levels to sweep
	serviceDur   time.Duration // work done per request inside the service
	rejectBackto time.Duration // how long a rejected client waits before retrying
	window       time.Duration // how long each step runs
	cpuAtLimit   float64       // modeled service CPU% when service is at `limit` in flight
	queue        bool          // queue mode (block to acquire) vs reject mode (503 when full)
	csvPath      string        // optional CSV output
}

// result holds the measured (and one modeled) outcome of a single step.
type result struct {
	conc        int
	total       int64
	success     int64
	rejected    int64
	svcPeak     int64         // peak requests in flight inside the service (measured)
	proxyPeak   int64         // peak requests in flight inside the proxy (measured)
	p50         time.Duration // median successful-request latency (measured)
	p95         time.Duration // measured
	throughput  float64       // successful req/s (measured)
	serviceCPU  float64       // MODELED: see cpuAtLimit
	successRate float64       // measured
}

func main() {
	cfg := parseFlags()

	printHeader(cfg)

	var results []result
	for _, conc := range cfg.steps {
		results = append(results, runStep(cfg, conc))
	}

	printTable(results)
	fmt.Println()
	printChart("Service CPU stays low (modeled)", results, func(r result) float64 { return r.serviceCPU }, styles.OK)
	fmt.Println()
	printChart("Client success rate collapses (measured)", results, func(r result) float64 { return r.successRate }, styles.Err)
	fmt.Println()
	printFooter(cfg, results)

	if cfg.csvPath != "" {
		if err := writeCSV(cfg.csvPath, results); err != nil {
			fmt.Fprintln(os.Stderr, "could not write CSV:", err)
			os.Exit(1)
		}
		fmt.Println(styles.Sub.Render("Wrote results to " + cfg.csvPath))
	}
}

func parseFlags() config {
	limit := flag.Int("limit", 50, "proxy concurrency limit: max requests in flight through the proxy at once")
	stepsStr := flag.String("steps", "10,30,60,120", "client concurrency levels to sweep, comma-separated")
	serviceMs := flag.Int("service-ms", 20, "work per request inside the service, in milliseconds (real sleep)")
	windowMs := flag.Int("window-ms", 2000, "how long to run each step, in milliseconds")
	rejectMs := flag.Int("reject-backoff-ms", -1,
		"how long a rejected client waits before retrying; -1 means one service time (the slot won't free sooner)")
	cpuAtLimit := flag.Float64("cpu-at-limit", 35.0,
		"MODELED service CPU%% when the service is handling `limit` requests at once; CPU scales linearly below that")
	queue := flag.Bool("queue", false, "queue mode: block until a slot frees instead of rejecting (shows latency instead of failures)")
	csvPath := flag.String("csv", "", "optional path to write results as CSV")
	flag.Parse()

	var steps []int
	for _, s := range strings.Split(*stepsStr, ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		n, err := strconv.Atoi(s)
		if err != nil || n <= 0 {
			fmt.Fprintf(os.Stderr, "invalid --steps value %q: want positive integers like 10,30,60,120\n", s)
			os.Exit(1)
		}
		steps = append(steps, n)
	}
	if len(steps) == 0 {
		fmt.Fprintln(os.Stderr, "--steps must list at least one positive integer")
		os.Exit(1)
	}

	serviceDur := time.Duration(*serviceMs) * time.Millisecond
	rejectBackto := serviceDur
	if *rejectMs >= 0 {
		rejectBackto = time.Duration(*rejectMs) * time.Millisecond
	}

	return config{
		limit:        *limit,
		steps:        steps,
		serviceDur:   serviceDur,
		rejectBackto: rejectBackto,
		window:       time.Duration(*windowMs) * time.Millisecond,
		cpuAtLimit:   *cpuAtLimit,
		queue:        *queue,
		csvPath:      *csvPath,
	}
}

// runStep drives the client at one concurrency level for cfg.window and returns
// the measured outcome. The proxy and service live in this same process; the
// only thing standing between the client and the service is the semaphore.
func runStep(cfg config, conc int) result {
	sem := make(chan struct{}, cfg.limit)

	var (
		total, success, rejected int64
		svcInFlight, svcPeak     int64
		proxyInFlight, proxyPeak int64
	)

	// One latency slice per client goroutine, merged at the end. No shared
	// mutex on the hot path, so the measurement does not distort itself.
	lat := make([][]time.Duration, conc)

	deadline := time.Now().Add(cfg.window)
	var wg sync.WaitGroup
	wg.Add(conc)
	for i := 0; i < conc; i++ {
		go func(id int) {
			defer wg.Done()
			for time.Now().Before(deadline) {
				start := time.Now()
				ok := serve(cfg, sem, &svcInFlight, &svcPeak, &proxyInFlight, &proxyPeak)
				atomic.AddInt64(&total, 1)
				if ok {
					atomic.AddInt64(&success, 1)
					lat[id] = append(lat[id], time.Since(start))
				} else {
					atomic.AddInt64(&rejected, 1)
				}
			}
		}(i)
	}
	wg.Wait()

	// Merge and summarize latencies.
	var all []time.Duration
	for _, s := range lat {
		all = append(all, s...)
	}
	sort.Slice(all, func(i, j int) bool { return all[i] < all[j] })

	svcPeakV := atomic.LoadInt64(&svcPeak)
	successV := atomic.LoadInt64(&success)

	return result{
		conc:        conc,
		total:       atomic.LoadInt64(&total),
		success:     successV,
		rejected:    atomic.LoadInt64(&rejected),
		svcPeak:     svcPeakV,
		proxyPeak:   atomic.LoadInt64(&proxyPeak),
		p50:         percentile(all, 0.50),
		p95:         percentile(all, 0.95),
		throughput:  float64(successV) / cfg.window.Seconds(),
		serviceCPU:  modelCPU(svcPeakV, cfg.limit, cfg.cpuAtLimit),
		successRate: rate(successV, atomic.LoadInt64(&total)),
	}
}

// serve is one request's trip through the proxy and into the service. The proxy
// is the semaphore: in reject mode a full semaphore turns the request away
// immediately (like a proxy at its concurrency limit returning 503); in queue
// mode the request waits for a slot. Everything past the semaphore is the
// service doing real work.
func serve(cfg config, sem chan struct{}, svcInFlight, svcPeak, proxyInFlight, proxyPeak *int64) bool {
	// Try to enter the proxy: acquire a concurrency slot.
	if cfg.queue {
		sem <- struct{}{} // block until a slot frees
	} else {
		select {
		case sem <- struct{}{}:
		default:
			// Proxy is full: rejected at the door. A closed-loop client does
			// not instantly re-fire; it backs off, and the slot will not free
			// before roughly one service time anyway. Without this the client
			// would spin at CPU speed and the numbers would be meaningless.
			time.Sleep(cfg.rejectBackto)
			return false
		}
	}
	defer func() { <-sem }()

	// Admitted. Count it as in flight through the proxy...
	p := atomic.AddInt64(proxyInFlight, 1)
	updatePeak(proxyPeak, p)
	defer atomic.AddInt64(proxyInFlight, -1)

	// ...and the service handles it.
	s := atomic.AddInt64(svcInFlight, 1)
	updatePeak(svcPeak, s)
	time.Sleep(cfg.serviceDur) // the service's real work
	atomic.AddInt64(svcInFlight, -1)
	return true
}

// updatePeak raises *peak to v if v is larger, atomically.
func updatePeak(peak *int64, v int64) {
	for {
		old := atomic.LoadInt64(peak)
		if v <= old || atomic.CompareAndSwapInt64(peak, old, v) {
			return
		}
	}
}

// modelCPU is the one number in this lab that is modeled rather than measured.
// The claim it makes honest is structural, not numeric: the service's load is
// bounded by the proxy, so once the proxy is full the service sees at most
// `limit` requests in flight no matter how hard the client pushes. We assume
// the service runs at cpuAtLimit percent when saturated at the limit, and
// scales linearly below that. The exact percentage is illustrative; the ceiling
// is the point.
func modelCPU(svcPeak int64, limit int, cpuAtLimit float64) float64 {
	if limit <= 0 {
		return 0
	}
	frac := float64(svcPeak) / float64(limit)
	if frac > 1 {
		frac = 1
	}
	return frac * cpuAtLimit
}

func rate(n, d int64) float64 {
	if d == 0 {
		return 0
	}
	return float64(n) / float64(d) * 100
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
	mode := "reject when full"
	if cfg.queue {
		mode = "queue when full"
	}
	meta := fmt.Sprintf("proxy limit %d  ·  service %s/req  ·  %s each  ·  %s",
		cfg.limit, cfg.serviceDur, cfg.window, mode)
	fmt.Println()
	fmt.Println(styles.Header("3B: Bit By Byte   Chapter 2: The Proxy Beside Your Application", meta))
	fmt.Println()
	fmt.Println(styles.Sub.Render("Client -> Proxy (concurrency limit) -> Service (healthy, fast)"))
	fmt.Println(styles.Sub.Render("Ramp the client and watch the two stories split: the service stays"))
	fmt.Println(styles.Sub.Render("bounded by the proxy while the client starts failing at the door."))
	fmt.Println()
}

func printTable(rs []result) {
	fmt.Println(styles.Bold.Render(
		pad("Client", 8) + pad("Svc CPU", 9) + pad("Svc peak", 10) +
			pad("Proxy", 8) + pad("Success", 9) + pad("p50", 9) + "p95"))
	fmt.Println(styles.Bold.Render(
		pad(" conc", 8) + pad("(model)", 9) + pad("in-flight", 10) +
			pad("reject", 8) + pad("rate", 9) + pad("lat", 9) + "lat"))
	fmt.Println(styles.Rule(60))
	for _, r := range rs {
		row := pad(fmt.Sprintf(" %d", r.conc), 8) +
			pad(fmt.Sprintf("%.0f%%", r.serviceCPU), 9) +
			pad(fmt.Sprintf("%d", r.svcPeak), 10) +
			pad(fmt.Sprintf("%d", r.rejected), 8) +
			pad(fmt.Sprintf("%.0f%%", r.successRate), 9) +
			pad(shortDur(r.p50), 9) +
			shortDur(r.p95)
		// Color the row by how the client fared: green while healthy, red once
		// the proxy starts rejecting.
		if r.rejected > 0 {
			fmt.Println(styles.Err.Render(row))
		} else {
			fmt.Println(styles.OK.Render(row))
		}
	}
}

// printChart draws a labelled row of horizontal bars, one per step, scaled to a
// fixed 0-100 range so the two charts line up visually.
func printChart(title string, rs []result, val func(result) float64, barStyle interface{ Render(...string) string }) {
	fmt.Println(styles.Bold.Render(title))
	const width = 40
	for _, r := range rs {
		v := val(r)
		if v < 0 {
			v = 0
		}
		if v > 100 {
			v = 100
		}
		filled := int(v / 100 * width)
		bar := barStyle.Render(strings.Repeat("█", filled)) +
			styles.Sub.Render(strings.Repeat("·", width-filled))
		fmt.Printf("  %s  %s  %s\n",
			styles.Sub.Render(pad(fmt.Sprintf("conc %d", r.conc), 9)),
			bar,
			styles.Bold.Render(fmt.Sprintf("%.0f%%", v)))
	}
}

func printFooter(cfg config, rs []result) {
	fmt.Println(styles.Rule(60))
	// Find the point where the client started failing.
	var brokeAt int
	for _, r := range rs {
		if r.rejected > 0 {
			brokeAt = r.conc
			break
		}
	}
	if brokeAt > 0 {
		fmt.Println(styles.Sub.Render(fmt.Sprintf(
			"The client began failing at concurrency %d (proxy limit %d). Past that",
			brokeAt, cfg.limit)))
		fmt.Println(styles.Sub.Render(
			"the service's in-flight count is pinned at the limit: its load, and so"))
		fmt.Println(styles.Sub.Render(
			"its CPU, cannot rise no matter how hard the client pushes. Green service,"))
		fmt.Println(styles.Sub.Render("failing client, same system."))
	}
	fmt.Println()
	fmt.Println(styles.Sub.Render("Try it: --queue turns failures into latency instead of errors; raise"))
	fmt.Println(styles.Sub.Render("--service-ms and throughput falls while the CPU ceiling holds, because"))
	fmt.Println(styles.Sub.Render("limit = throughput x latency (the service can only clear limit/latency)."))
	fmt.Println()
}

func writeCSV(path string, rs []result) error {
	var b strings.Builder
	b.WriteString("client_conc,service_cpu_model_pct,service_peak_inflight,proxy_peak_inflight,rejected,success_rate_pct,p50_ms,p95_ms,throughput_rps\n")
	for _, r := range rs {
		b.WriteString(fmt.Sprintf("%d,%.1f,%d,%d,%d,%.1f,%.2f,%.2f,%.1f\n",
			r.conc, r.serviceCPU, r.svcPeak, r.proxyPeak, r.rejected,
			r.successRate, ms(r.p50), ms(r.p95), r.throughput))
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
