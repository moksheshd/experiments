// Command retryamp is the companion lab for 3B: Bit By Byte, Chapter 4,
// "When Retries Become the Outage."
//
// It reproduces one mechanism from the August 17, 2026 GitHub incident: retries
// do not create capacity. GitHub reported that optimistic retry logic worsened
// the incident by overloading internal load balancers. Retries did not start the
// fire. They fed it. Here we build the smallest system where that happens on
// purpose, watch it happen, then apply the standard discipline and watch the same
// load get tamed.
//
// The system is a real, in-process open-loop Client -> Service:
//
//   - Client: an OPEN-loop generator. Every 10ms it launches a fixed batch of
//     LOGICAL requests, so the offered rate stays flat no matter how the service
//     behaves. This is the same open-loop generator as Chapter 3: the client
//     never backs off just because the service is struggling.
//   - Service: a hard concurrency limit L (default 50). Each admitted request
//     does serviceMs (default 40ms) of real work (a real time.Sleep). Its ceiling
//     on useful work is therefore L / serviceMs, about 1250 successful req/s. When
//     it is full it rejects immediately, fast and cheap. That fast "no" is server
//     backpressure: it is the service protecting itself.
//   - Retries: a LOGICAL request may make several ATTEMPTS. Every attempt that
//     reaches the service (admitted or rejected) is counted. That count, divided
//     by the offered rate, is the amplification: how much extra load the retries
//     poured onto a service that was already out of capacity.
//
// We sweep the offered logical-request rate across levels that cross the ~1250
// capacity (default 500, 1000, 2000, 4000 req/s) and run the whole sweep twice:
//
//   - NAIVE: on failure, retry immediately, up to maxAttempts, with no backoff,
//     no jitter, and no budget. As offered load passes capacity, attempts balloon
//     far past offered and the logical success rate craters.
//   - DISCIPLINED: exponential backoff plus jitter, and a retry BUDGET capped at a
//     small fraction of offered traffic (default 10%). Once the budget is spent, a
//     failed request gives up instead of retrying. Amplification stays pinned near
//     1x, goodput holds near capacity, and the service survives.
//
// Everything in this lab is measured from the real run. There are no modeled
// numbers: attempts, amplification, goodput, and success rate are all counted
// from real goroutines hitting a real concurrency-limited service.
package main

import (
	"flag"
	"fmt"
	"math/rand/v2"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/moksheshd/experiments/3b-bit-by-byte/season-01-github-august-17/styles"
)

type config struct {
	offeredSteps []int         // offered logical-request rates to sweep (req/s)
	limit        int           // service concurrency limit
	serviceDur   time.Duration // real work per admitted request
	maxAttempts  int           // attempts per logical request (1 first try + retries)
	budgetFrac   float64       // disciplined: retry budget as a fraction of offered
	backoff      time.Duration // disciplined: base backoff, doubled each retry
	settle       time.Duration // per step: let the pipeline fill before measuring
	window       time.Duration // per step: measurement window after settling
	mode         string        // "both" | "naive" | "disciplined"
	csvPath      string
}

// result holds the measured outcome of a single offered-load step. Every field is
// counted from the real run; nothing here is modeled.
type result struct {
	offered      int     // offered logical req/s (the target rate)
	attemptsRate float64 // attempts reaching the service, per second (measured)
	amplify      float64 // attempts / offered: the amplification (measured)
	goodput      float64 // successful logical req/s (measured)
	successPct   float64 // successful logical / offered logical (measured)
}

const genInterval = 10 * time.Millisecond // load generator tick granularity

func main() {
	cfg := parseFlags()
	printHeader(cfg)

	switch cfg.mode {
	case "both":
		naive := runSweep(cfg, "naive")
		disc := runSweep(cfg, "disciplined")
		printSweep("NAIVE retries  (retry immediately, no backoff, no budget)", naive)
		fmt.Println()
		printSweep("DISCIPLINED retries  (backoff + jitter + 10% budget + backpressure)", disc)
		fmt.Println()
		printChart("Amplification under NAIVE retries (attempts / offered)", naive,
			func(r result) float64 { return r.amplify }, float64(cfg.maxAttempts), "x", styles.Err)
		fmt.Println()
		printChart("Amplification under DISCIPLINED retries (attempts / offered)", disc,
			func(r result) float64 { return r.amplify }, float64(cfg.maxAttempts), "x", styles.OK)
		fmt.Println()
		printChart("Logical success rate under NAIVE retries (% of offered)", naive,
			func(r result) float64 { return r.successPct }, 100, "%", styles.Err)
		fmt.Println()
		printFooter(cfg, naive, disc)
		if cfg.csvPath != "" {
			writeAndReport(cfg.csvPath, naive, disc)
		}
	default:
		res := runSweep(cfg, cfg.mode)
		printSweep(cfg.mode+" retries", res)
		fmt.Println()
		if cfg.csvPath != "" {
			writeAndReport(cfg.csvPath, res, nil)
		}
	}
}

func parseFlags() config {
	stepsStr := flag.String("offered-steps", "500,1000,2000,4000",
		"offered logical-request rates to sweep, in req/s, comma-separated (should cross capacity)")
	limit := flag.Int("limit", 50, "service concurrency limit: max requests in flight at once")
	serviceMs := flag.Int("service-ms", 40, "real work per admitted request, in milliseconds")
	maxAttempts := flag.Int("max-attempts", 5, "attempts per logical request (1 first try plus retries)")
	budgetFrac := flag.Float64("budget", 0.10, "disciplined: retry budget as a fraction of offered traffic")
	backoffMs := flag.Int("backoff-ms", 50, "disciplined: base backoff in ms, doubled each retry, with full jitter")
	settleMs := flag.Int("settle-ms", 800, "per step: time to let the pipeline fill before measuring")
	windowMs := flag.Int("window-ms", 3000, "per step: measurement window after settling")
	mode := flag.String("mode", "both", "which run(s) to do: both | naive | disciplined")
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
			fmt.Fprintf(os.Stderr, "invalid --offered-steps value %q: want positive integers like 500,1000,2000,4000\n", s)
			os.Exit(1)
		}
		steps = append(steps, n)
	}
	if len(steps) == 0 {
		fmt.Fprintln(os.Stderr, "--offered-steps must list at least one positive integer")
		os.Exit(1)
	}
	switch *mode {
	case "both", "naive", "disciplined":
	default:
		fmt.Fprintf(os.Stderr, "invalid --mode %q: want both | naive | disciplined\n", *mode)
		os.Exit(1)
	}

	return config{
		offeredSteps: steps,
		limit:        *limit,
		serviceDur:   time.Duration(*serviceMs) * time.Millisecond,
		maxAttempts:  *maxAttempts,
		budgetFrac:   *budgetFrac,
		backoff:      time.Duration(*backoffMs) * time.Millisecond,
		settle:       time.Duration(*settleMs) * time.Millisecond,
		window:       time.Duration(*windowMs) * time.Millisecond,
		mode:         *mode,
		csvPath:      *csvPath,
	}
}

func runSweep(cfg config, mode string) []result {
	var out []result
	for _, offered := range cfg.offeredSteps {
		out = append(out, runStep(cfg, mode, offered))
	}
	return out
}

// runStep holds the offered rate flat at one level, runs the client and service
// live for settle+window, and measures the steady state over the final window.
// The service is an atomic in-flight counter checked against the fixed limit; a
// full service rejects instantly. That fast rejection is the backpressure both
// runs get. The only thing that changes between runs is how the client reacts to
// it.
func runStep(cfg config, mode string, offered int) result {
	var (
		inFlight     atomic.Int64 // requests in flight at the service right now
		offeredCount atomic.Int64 // logical requests started during the window
		attempts     atomic.Int64 // attempts reaching the service during the window
		goodput      atomic.Int64 // logical requests that succeeded during the window
	)
	var measuring atomic.Bool

	// call is one attempt at the service: admit if in flight is below the limit,
	// otherwise reject instantly (backpressure). Admitted attempts do real work.
	call := func() bool {
		for {
			n := inFlight.Load()
			if n >= int64(cfg.limit) {
				return false // service full: rejected fast and cheap
			}
			if inFlight.CompareAndSwap(n, n+1) {
				break
			}
		}
		time.Sleep(cfg.serviceDur) // the service's real work
		inFlight.Add(-1)
		return true
	}

	// The retry budget (disciplined only): a token bucket refilled at
	// budgetFrac x offered tokens/sec. A retry must spend a token; when the bucket
	// is empty the request gives up instead of retrying. This caps amplification no
	// matter how bad things get.
	var budget bucket
	if mode == "disciplined" {
		budget.cap = int64(cfg.budgetFrac*float64(offered)*0.25) + 1
	}

	// logical is the whole life of one logical request: try, and on rejection
	// retry per the policy, up to maxAttempts.
	logical := func() {
		if measuring.Load() {
			offeredCount.Add(1)
		}
		for attempt := 1; attempt <= cfg.maxAttempts; attempt++ {
			if attempt > 1 && mode == "disciplined" {
				// Disciplined retry: spend a budget token or give up, then wait a
				// backed-off, jittered interval before trying again.
				if !budget.take() {
					return // budget spent: fail fast instead of piling on
				}
				time.Sleep(backoffDur(cfg.backoff, attempt-1))
			}
			// Naive retry just loops immediately: no token, no wait.
			meas := measuring.Load()
			if meas {
				attempts.Add(1)
			}
			if call() {
				if meas {
					goodput.Add(1)
				}
				return
			}
		}
		// Fell through all attempts without success: the logical request failed.
	}

	stop := make(chan struct{})
	var wg sync.WaitGroup

	// Open-loop generator: a fixed batch of logical requests every genInterval, so
	// the offered rate stays flat no matter how the service behaves.
	perTick := offered * int(genInterval/time.Millisecond) / 1000
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
						logical()
					}()
				}
			}
		}
	}()

	// Budget refiller (disciplined only): a single goroutine tops the bucket up at
	// the configured rate, carrying the fractional remainder itself so no float is
	// shared across goroutines.
	if mode == "disciplined" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			const refill = 20 * time.Millisecond
			perSec := cfg.budgetFrac * float64(offered)
			carry := 0.0
			t := time.NewTicker(refill)
			defer t.Stop()
			for {
				select {
				case <-stop:
					return
				case <-t.C:
					carry += perSec * refill.Seconds()
					whole := int64(carry)
					if whole > 0 {
						carry -= float64(whole)
						budget.add(whole)
					}
				}
			}
		}()
	}

	time.Sleep(cfg.settle)
	measuring.Store(true)
	time.Sleep(cfg.window)
	measuring.Store(false)
	close(stop)
	wg.Wait()

	off := offeredCount.Load()
	att := attempts.Load()
	good := goodput.Load()
	winSec := cfg.window.Seconds()

	amp := 0.0
	if off > 0 {
		amp = float64(att) / float64(off)
	}
	successPct := 0.0
	if off > 0 {
		successPct = float64(good) / float64(off) * 100
	}
	return result{
		offered:      offered,
		attemptsRate: float64(att) / winSec,
		amplify:      amp,
		goodput:      float64(good) / winSec,
		successPct:   successPct,
	}
}

// bucket is a lock-free token bucket for the retry budget. take spends a token if
// one is available; add returns tokens, capped so an idle bucket cannot hoard a
// huge burst.
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

// backoffDur returns a full-jittered exponential backoff: a uniform random wait in
// (0, base x 2^(retry-1)]. Backoff gives the struggling service room to breathe;
// the jitter spreads a wall of simultaneous retries out across time so they do not
// arrive as one synchronized spike.
func backoffDur(base time.Duration, retry int) time.Duration {
	shift := retry - 1
	if shift > 10 {
		shift = 10 // cap the exponent so the wait cannot run away
	}
	span := int64(base) << shift
	if span <= 0 {
		return 0
	}
	return time.Duration(rand.Int64N(span) + 1)
}

func capacityRPS(cfg config) float64 {
	if cfg.serviceDur <= 0 {
		return 0
	}
	return float64(cfg.limit) / cfg.serviceDur.Seconds()
}

func capacityLabel(cfg config) string {
	return fmt.Sprintf("%.0f", capacityRPS(cfg))
}

func printHeader(cfg config) {
	meta := fmt.Sprintf("service limit %d  ·  %s/req  ·  capacity about %s req/s  ·  offered %s",
		cfg.limit, cfg.serviceDur, capacityLabel(cfg), stepsLabel(cfg.offeredSteps))
	fmt.Println()
	fmt.Println(styles.Header("3B: Bit By Byte   Chapter 4: When Retries Become the Outage", meta))
	fmt.Println()
	fmt.Println(styles.Sub.Render("Client (open loop, flat offered rate) -> Service (hard concurrency limit)"))
	fmt.Println(styles.Sub.Render("Push the offered rate past what the service can clear, and watch the"))
	fmt.Println(styles.Sub.Render("attempts reaching the service lift off from the offered rate. Then add"))
	fmt.Println(styles.Sub.Render("backoff, jitter, and a retry budget, and watch the same load get tamed."))
	fmt.Println()
}

func printSweep(title string, rs []result) {
	fmt.Println(styles.Bold.Render(title))
	fmt.Println(styles.Bold.Render(
		pad("Offered", 10) + pad("Attempts", 11) + pad("Amplify", 11) +
			pad("Goodput", 10) + "Success"))
	fmt.Println(styles.Bold.Render(
		pad("req/s", 10) + pad("@service/s", 11) + pad("x offered", 11) +
			pad("req/s", 10) + "rate"))
	fmt.Println(styles.Rule(52))
	for _, r := range rs {
		row := pad(fmt.Sprintf("%d", r.offered), 10) +
			pad(fmt.Sprintf("%.0f", r.attemptsRate), 11) +
			pad(fmt.Sprintf("%.1fx", r.amplify), 11) +
			pad(fmt.Sprintf("%.0f", r.goodput), 10) +
			fmt.Sprintf("%.0f%%", r.successPct)
		// Green while retries stay honest; red once amplification takes off.
		if r.amplify >= 1.5 {
			fmt.Println(styles.Err.Render(row))
		} else {
			fmt.Println(styles.OK.Render(row))
		}
	}
}

// printChart draws a labelled row of horizontal bars, one per step, scaled to a
// fixed max so charts line up visually.
func printChart(title string, rs []result, val func(result) float64, max float64, suffix string, barStyle interface{ Render(...string) string }) {
	fmt.Println(styles.Bold.Render(title))
	const width = 40
	if max <= 0 {
		max = 1
	}
	for _, r := range rs {
		v := val(r)
		if v < 0 {
			v = 0
		}
		frac := v / max
		if frac > 1 {
			frac = 1
		}
		filled := int(frac * width)
		bar := barStyle.Render(strings.Repeat("█", filled)) +
			styles.Sub.Render(strings.Repeat("·", width-filled))
		fmt.Printf("  %s  %s  %s\n",
			styles.Sub.Render(pad(fmt.Sprintf("%d/s", r.offered), 8)),
			bar,
			styles.Bold.Render(fmt.Sprintf("%.1f%s", v, suffix)))
	}
}

func printFooter(cfg config, naive, disc []result) {
	fmt.Println(styles.Rule(52))
	last := len(cfg.offeredSteps) - 1
	nw := naive[last]
	dw := disc[last]
	fmt.Println(styles.Sub.Render(fmt.Sprintf(
		"At %d offered req/s against about %s of capacity, the offered rate was",
		nw.offered, capacityLabel(cfg))))
	fmt.Println(styles.Sub.Render("identical in both runs. Only the retry discipline differed."))
	fmt.Println()
	fmt.Println(styles.Sub.Render(fmt.Sprintf(
		"Naive: retries pushed %.0f attempts/s at the service, %.1fx the offered load,",
		nw.attemptsRate, nw.amplify)))
	fmt.Println(styles.Sub.Render(fmt.Sprintf(
		"for %.0f goodput. Disciplined: %.0f attempts/s, %.1fx, for %.0f goodput.",
		nw.goodput, dw.attemptsRate, dw.amplify, dw.goodput)))
	fmt.Println()
	fmt.Println(styles.Sub.Render("Same goodput. Retries did not create capacity. The naive run just poured"))
	fmt.Println(styles.Sub.Render("several times the load onto a service that was already out of room, which"))
	fmt.Println(styles.Sub.Render("is exactly how a local slowdown becomes a shared-infrastructure outage."))
	fmt.Println()
	fmt.Println(styles.Sub.Render("Try it: --max-attempts changes the naive ceiling; --budget moves the"))
	fmt.Println(styles.Sub.Render("disciplined cap; --offered-steps reshapes where the load crosses capacity."))
	fmt.Println()
}

func writeAndReport(path string, naive, disc []result) {
	if err := writeCSV(path, naive, disc); err != nil {
		fmt.Fprintln(os.Stderr, "could not write CSV:", err)
		os.Exit(1)
	}
	fmt.Println(styles.Sub.Render("Wrote results to " + path))
}

func writeCSV(path string, naive, disc []result) error {
	var b strings.Builder
	b.WriteString("mode,offered_rps,attempts_per_sec,amplification,goodput_rps,success_pct\n")
	writeRows := func(mode string, rs []result) {
		for _, r := range rs {
			b.WriteString(fmt.Sprintf("%s,%d,%.0f,%.2f,%.0f,%.1f\n",
				mode, r.offered, r.attemptsRate, r.amplify, r.goodput, r.successPct))
		}
	}
	writeRows("naive", naive)
	if disc != nil {
		writeRows("disciplined", disc)
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func stepsLabel(steps []int) string {
	parts := make([]string, len(steps))
	for i, s := range steps {
		parts[i] = fmt.Sprintf("%d", s)
	}
	return strings.Join(parts, " -> ")
}

func pad(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
