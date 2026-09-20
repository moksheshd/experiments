// Command retrystorm is the companion lab for 3B: Bit By Byte, Chapter 6,
// "The Copilot Retry Storm."
//
// It reproduces the self-inflicted flood from the August 17, 2026 GitHub
// incident: a fleet of clients that turns a slow endpoint into roughly ten times
// its normal load, all by itself. GitHub reported the Copilot Token Service went
// from about 7,000-9,000 requests per second to about 70,000-100,000, and
// attributed it to a latent retry bug in VS Code where delayed responses caused
// about tenfold request amplification. Here we build the smallest system where
// that happens on purpose and watch the amplification emerge.
//
// The system is a fleet of many identical clients hitting one token endpoint on a
// steady heartbeat. Everything about the storm is measured: the one thing that is
// illustrative, not literal, is the scale. A fleet of a couple thousand clients
// stands in for GitHub's 7-9K/s world. The AMPLIFICATION FACTOR (attempts
// reaching the endpoint, divided by the steady baseline) is the measured payload,
// and it does not depend on the absolute scale.
//
// We run a progression of scenarios and watch the factor move:
//
//   - Healthy baseline: the endpoint answers fast, so one beat is one request.
//   - Buggy client + slow endpoint: on a timeout the client reissues immediately,
//     without canceling the in-flight original, so a slow response spawns a stack
//     of duplicates. The factor climbs toward tenfold.
//   - Failover: point the fleet at a fresh, healthy-region server. The factor does
//     not move, because the clients never changed. The herd follows the target.
//   - Client-side discipline (sane timeout, backoff, jitter, retry budget): each
//     control bends the curve, and the budget caps amplification for good.
//
// The lesson the numbers make physical: the fix for a client storm lives in the
// client, not the server.
package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/moksheshd/experiments/3b-bit-by-byte/season-01-github-august-17/styles"
)

type config struct {
	clients  int           // fleet size
	interval time.Duration // per-client heartbeat: one logical request per interval
	fastResp time.Duration // healthy endpoint response time
	slowResp time.Duration // faulted endpoint response time (delayed, past the buggy timeout)
	capacity int           // endpoint concurrency limit (requests it can serve at once)
	timeout  time.Duration // buggy client's aggressive timeout
	saneTO   time.Duration // disciplined client's timeout (longer than a healthy response)
	backoff  time.Duration // disciplined base backoff between attempts
	budget   float64       // disciplined retry budget as a fraction of baseline (e.g. 0.10)
	maxTry   int           // hard safety cap on attempts per logical request
	settle   time.Duration // warm-up before measuring
	window   time.Duration // measurement window
	csvPath  string
}

// policy is one client behavior under test.
type policy struct {
	name     string
	slow     bool          // is the endpoint faulted (delayed responses)?
	timeout  time.Duration // how long the client waits before reissuing
	backoff  time.Duration // base wait between attempts (0 = immediate reissue = the bug)
	jitter   bool          // randomize the waits to break the herd
	budgeted bool          // cap retries with a budget and then fail fast
	note     string        // one-line explanation for the footer
}

// result is the measured outcome of one scenario.
type result struct {
	name     string
	attempts int64   // total attempts that reached the endpoint during the window
	logical  int64   // total logical requests the fleet intended during the window
	success  int64   // logical requests that got a good answer
	rps      float64 // attempts per second reaching the endpoint (measured)
	ampl     float64 // rps / baseline: the amplification factor
	okRate   float64 // logical success rate
}

func main() {
	cfg := parseFlags()
	printHeader(cfg)

	// The baseline is the steady, healthy world: one beat, one request.
	baseline := float64(cfg.clients) / cfg.interval.Seconds()

	scenarios := []policy{
		{name: "Healthy baseline", slow: false, timeout: cfg.timeout, backoff: 0,
			note: "endpoint answers fast, so one heartbeat is one request"},
		{name: "Buggy client + slow endpoint", slow: true, timeout: cfg.timeout, backoff: 0,
			note: "reissue immediately on timeout, original left in flight: the bug"},
		{name: "Same clients, failover", slow: true, timeout: cfg.timeout, backoff: 0,
			note: "fresh server, unchanged clients: the herd follows the target"},
		{name: "Fix: sane timeout", slow: true, timeout: cfg.saneTO, backoff: 0,
			note: "wait longer than a healthy response before giving up"},
		{name: "Fix: backoff + jitter", slow: true, timeout: cfg.timeout, backoff: cfg.backoff, jitter: true,
			note: "space retries out and desynchronize the herd"},
		{name: "Fix: retry budget", slow: true, timeout: cfg.timeout, backoff: cfg.backoff, jitter: true, budgeted: true,
			note: "hard cap: retries may add only a small fraction, then fail fast"},
	}

	var results []result
	for _, p := range scenarios {
		results = append(results, runScenario(cfg, p, baseline))
	}

	printTable(results, baseline)
	fmt.Println()
	printChart("Amplification: attempts reaching the endpoint, x baseline", results)
	fmt.Println()
	printFooter(cfg, scenarios, results, baseline)

	if cfg.csvPath != "" {
		if err := writeCSV(cfg.csvPath, results, baseline); err != nil {
			fmt.Fprintln(os.Stderr, "could not write CSV:", err)
			os.Exit(1)
		}
		fmt.Println(styles.Sub.Render("Wrote results to " + cfg.csvPath))
	}
}

func parseFlags() config {
	clients := flag.Int("clients", 1000, "fleet size: number of identical clients")
	intervalMs := flag.Int("interval-ms", 500, "per-client heartbeat: one logical request per this interval")
	fastMs := flag.Int("fast-ms", 20, "healthy endpoint response time in ms")
	slowMs := flag.Int("slow-ms", 500, "faulted endpoint response time in ms (delayed, past the buggy timeout)")
	capacity := flag.Int("capacity", 50000, "endpoint concurrency limit; the default is effectively unlimited, so the injected fault is latency (delayed responses), not capacity")
	timeoutMs := flag.Int("timeout-ms", 50, "buggy client's aggressive timeout in ms")
	saneMs := flag.Int("sane-timeout-ms", 800, "disciplined client's timeout in ms (longer than a healthy response)")
	backoffMs := flag.Int("backoff-ms", 50, "disciplined base backoff between attempts in ms")
	budget := flag.Float64("budget", 0.10, "disciplined retry budget as a fraction of baseline traffic")
	maxTry := flag.Int("max-attempts", 40, "hard safety cap on attempts per logical request")
	settleMs := flag.Int("settle-ms", 800, "warm-up before measuring, in ms")
	windowMs := flag.Int("window-ms", 2000, "measurement window, in ms")
	csvPath := flag.String("csv", "", "optional path to write results as CSV")
	flag.Parse()

	return config{
		clients:  *clients,
		interval: time.Duration(*intervalMs) * time.Millisecond,
		fastResp: time.Duration(*fastMs) * time.Millisecond,
		slowResp: time.Duration(*slowMs) * time.Millisecond,
		capacity: *capacity,
		timeout:  time.Duration(*timeoutMs) * time.Millisecond,
		saneTO:   time.Duration(*saneMs) * time.Millisecond,
		backoff:  time.Duration(*backoffMs) * time.Millisecond,
		budget:   *budget,
		maxTry:   *maxTry,
		settle:   time.Duration(*settleMs) * time.Millisecond,
		window:   time.Duration(*windowMs) * time.Millisecond,
		csvPath:  *csvPath,
	}
}

// runScenario spins up the fleet under one policy and measures, over a steady
// window, how many attempts reach the endpoint versus how many logical requests
// the fleet actually intended. The ratio is the amplification factor.
func runScenario(cfg config, p policy, baseline float64) result {
	respTime := cfg.fastResp
	if p.slow {
		respTime = cfg.slowResp
	}

	var (
		inFlight atomic.Int64 // requests inside the endpoint right now
		attempts atomic.Int64 // attempts reaching the endpoint (measured in window)
		logical  atomic.Int64 // logical requests intended (measured in window)
		success  atomic.Int64 // logical requests answered (measured in window)
		// A fleet-wide retry budget: retries allowed in the window on top of baseline.
		retryTokens atomic.Int64
	)
	// Seed the budget for the whole window: baseline logicals times the fraction.
	if p.budgeted {
		retryTokens.Store(int64(baseline*cfg.window.Seconds()*cfg.budget) + 1)
	}
	var measuring atomic.Bool

	// serve is the endpoint: admit if below capacity (fast reject otherwise), then
	// take respTime to answer. Rejections are counted as attempts too, because they
	// still reached and touched the endpoint.
	serve := func() bool {
		for {
			n := inFlight.Load()
			if n >= int64(cfg.capacity) {
				return false // saturated: fast reject
			}
			if inFlight.CompareAndSwap(n, n+1) {
				break
			}
		}
		time.Sleep(respTime)
		inFlight.Add(-1)
		return true
	}

	// logicalRequest is one intended request from one client, with retries per the
	// policy. It fires attempts and completes on the first successful answer.
	logicalRequest := func(seed int64) {
		// Each logical request gets its own RNG: several run concurrently per
		// client, and a shared *rand.Rand is not safe for concurrent use.
		rng := rand.New(rand.NewSource(seed))
		meas := measuring.Load()
		if meas {
			logical.Add(1)
		}
		done := make(chan struct{})
		got := make(chan struct{}, 1)
		var tries int

		var firer sync.WaitGroup
		firer.Add(1)
		go func() {
			defer firer.Done()
			wait := p.backoff
			for tries < cfg.maxTry {
				select {
				case <-done:
					return
				default:
				}
				// A retry (any attempt past the first) must draw a budget token if
				// the policy is budgeted; when the budget is spent, stop and fail.
				if tries > 0 && p.budgeted {
					if retryTokens.Add(-1) < 0 {
						return
					}
				}
				tries++
				if meas {
					attempts.Add(1)
				}
				go func() {
					if serve() {
						select {
						case got <- struct{}{}:
						default:
						}
					}
				}()

				// How long before the client fires again: the timeout for an
				// immediate-reissue bug, or a growing backoff for a disciplined client.
				step := p.timeout
				if p.backoff > 0 {
					step = wait
					if p.jitter {
						step = time.Duration(float64(wait) * (0.5 + rng.Float64()))
					}
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
			if meas {
				success.Add(1)
			}
		case <-time.After(logicalDeadline(cfg)):
			// Give up: this logical request never got a good answer.
		}
		close(done)
		firer.Wait()
	}

	// The fleet: each client beats on its own interval, phase-spread so they do not
	// all fire on the same tick at start.
	stop := make(chan struct{})
	var fleet sync.WaitGroup
	var seeds atomic.Int64
	for i := 0; i < cfg.clients; i++ {
		fleet.Add(1)
		go func(id int) {
			defer fleet.Done()
			rng := rand.New(rand.NewSource(int64(id) + 1))
			// Phase offset so the herd is not artificially synchronized by start time.
			time.Sleep(time.Duration(rng.Int63n(int64(cfg.interval))))
			t := time.NewTicker(cfg.interval)
			defer t.Stop()
			for {
				select {
				case <-stop:
					return
				case <-t.C:
					go logicalRequest(seeds.Add(1))
				}
			}
		}(i)
	}

	time.Sleep(cfg.settle)
	measuring.Store(true)
	time.Sleep(cfg.window)
	// Stop sampling new logical requests, then drain: keep the fleet running long
	// enough for the sampled requests to finish and record their outcome, so a
	// request that started late in the window is not miscounted as a failure just
	// because its (slow) answer would land after the window closed.
	measuring.Store(false)
	time.Sleep(logicalDeadline(cfg) + 100*time.Millisecond)
	close(stop)
	fleet.Wait()

	att := attempts.Load()
	log := logical.Load()
	suc := success.Load()
	rps := float64(att) / cfg.window.Seconds()
	okRate := 0.0
	if log > 0 {
		okRate = float64(suc) / float64(log) * 100
	}
	return result{
		name:     p.name,
		attempts: att,
		logical:  log,
		success:  suc,
		rps:      rps,
		ampl:     rps / baseline,
		okRate:   okRate,
	}
}

func printHeader(cfg config) {
	meta := fmt.Sprintf("fleet %d  ·  heartbeat %s  ·  slow response %s  ·  buggy timeout %s",
		cfg.clients, cfg.interval, cfg.slowResp, cfg.timeout)
	fmt.Println()
	fmt.Println(styles.Header("3B: Bit By Byte   Chapter 6: The Copilot Retry Storm", meta))
	fmt.Println()
	fmt.Println(styles.Sub.Render("A fleet of clients on a steady heartbeat -> one token endpoint."))
	fmt.Println(styles.Sub.Render("Make the endpoint slow, give the clients a reissue-on-timeout bug, and"))
	fmt.Println(styles.Sub.Render("watch the fleet amplify its own load. Then try to fix it two ways."))
	fmt.Println()
}

func printTable(rs []result, baseline float64) {
	fmt.Println(styles.Bold.Render(fmt.Sprintf("Baseline (steady, healthy world): %.0f req/s", baseline)))
	fmt.Println()
	fmt.Println(styles.Bold.Render(
		pad("Scenario", 32) + pad("Endpoint", 11) + pad("Amplify", 10) + "Success"))
	fmt.Println(styles.Bold.Render(
		pad("", 32) + pad("req/s", 11) + pad("x base", 10) + "rate"))
	fmt.Println(styles.Rule(64))
	for _, r := range rs {
		row := pad(r.name, 32) +
			pad(fmt.Sprintf("%.0f", r.rps), 11) +
			pad(fmt.Sprintf("%.1fx", r.ampl), 10) +
			fmt.Sprintf("%.0f%%", r.okRate)
		// Green when amplification is tame, red once the storm is running.
		if r.ampl >= 2 {
			fmt.Println(styles.Err.Render(row))
		} else {
			fmt.Println(styles.OK.Render(row))
		}
	}
}

// printChart draws one horizontal bar per scenario, scaled so the tenfold storm
// fills the row and the tamed factors sit near the left.
func printChart(title string, rs []result) {
	fmt.Println(styles.Bold.Render(title))
	const width = 40
	var maxAmpl float64 = 1
	for _, r := range rs {
		if r.ampl > maxAmpl {
			maxAmpl = r.ampl
		}
	}
	for _, r := range rs {
		frac := r.ampl / maxAmpl
		if frac > 1 {
			frac = 1
		}
		filled := int(frac * width)
		style := styles.OK
		if r.ampl >= 2 {
			style = styles.Err
		}
		bar := style.Render(strings.Repeat("█", filled)) +
			styles.Sub.Render(strings.Repeat("·", width-filled))
		fmt.Printf("  %s  %s  %s\n",
			styles.Sub.Render(pad(short(r.name), 26)),
			bar,
			styles.Bold.Render(fmt.Sprintf("%.1fx", r.ampl)))
	}
}

func printFooter(cfg config, ps []policy, rs []result, baseline float64) {
	fmt.Println(styles.Rule(64))
	// Pull the three scenarios the essay leans on.
	var storm, failover, budget result
	for i, p := range ps {
		switch {
		case p.name == "Buggy client + slow endpoint":
			storm = rs[i]
		case p.name == "Same clients, failover":
			failover = rs[i]
		case p.budgeted:
			budget = rs[i]
		}
	}
	fmt.Println(styles.Sub.Render(fmt.Sprintf(
		"The bug turned a steady %.0f req/s into %.0f (%.1fx). Failover to a fresh",
		baseline, storm.rps, storm.ampl)))
	fmt.Println(styles.Sub.Render(fmt.Sprintf(
		"server left it at %.1fx: the clients generate the load, so moving the",
		failover.ampl)))
	fmt.Println(styles.Sub.Render(fmt.Sprintf(
		"server does nothing. A retry budget brought it back to %.1fx.", budget.ampl)))
	fmt.Println()
	fmt.Println(styles.Sub.Render("The fix for a client storm lives in the client, not the server."))
	fmt.Println()
	fmt.Println(styles.Sub.Render("Try it: --slow-ms and --timeout-ms set the amplification (roughly their"))
	fmt.Println(styles.Sub.Render("ratio); --budget moves the hard cap; --clients scales the fleet."))
	fmt.Println()
}

func writeCSV(path string, rs []result, baseline float64) error {
	var b strings.Builder
	b.WriteString("scenario,baseline_rps,endpoint_rps,amplification_x,logical,success,success_rate_pct\n")
	for _, r := range rs {
		b.WriteString(fmt.Sprintf("%q,%.1f,%.1f,%.2f,%d,%d,%.1f\n",
			r.name, baseline, r.rps, r.ampl, r.logical, r.success, r.okRate))
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

// logicalDeadline is how long one logical request keeps trying before it gives
// up. It must exceed a slow response plus the disciplined timeout, so a request
// that would eventually succeed is not cut off early.
func logicalDeadline(cfg config) time.Duration {
	return cfg.slowResp + cfg.saneTO + 300*time.Millisecond
}

func short(name string) string {
	name = strings.TrimPrefix(name, "Fix: ")
	if len(name) > 26 {
		return name[:26]
	}
	return name
}

func pad(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
