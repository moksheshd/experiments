// Command autoscaleblind is the companion lab for 3B: Bit By Byte, Chapter 3,
// "When CPU Says Everything Is Fine."
//
// It reproduces the autoscaling blind spot from the August 17, 2026 GitHub
// incident: an autoscaler that watches the wrong signal. GitHub reported that a
// sidecar reached its concurrency limit and did not scale, because the scaling
// policy watched the host service rather than the sidecar's own limits. Here we
// build the smallest system where that happens on purpose and watch it fail.
//
// The system is the same Client -> Proxy -> Service from Chapter 2, with two
// changes that make this chapter's point:
//
//   - The client is OPEN-loop. It fires at a fixed arrival rate and never slows
//     down, no matter how long requests take. That is the whole setup: the
//     arrival rate is held flat while the backend gets slower.
//   - There is an AUTOSCALER. Each step it reads one metric and adjusts the
//     number of proxy replicas (which multiplies the concurrency limit). We run
//     the whole sweep twice: once with the autoscaler watching service CPU (the
//     blind spot), once watching proxy concurrency (the fix).
//
// The experiment holds the arrival rate steady and makes the service slower step
// by step, the way a real downstream dependency degrades. Watch what happens:
//
//   - Arrival rate: flat. It never moves. That is the point.
//   - Latency: climbing, because we make it climb.
//   - Concurrency (in flight at the proxy): climbing with latency, because
//     concurrency is roughly arrival-rate times latency (Little's Law), until it
//     presses against the limit.
//   - Service CPU: calm, even falling, because the capped service completes less
//     real work per second while requests pile up waiting.
//   - The CPU-watching autoscaler: silent. CPU looks great, so it never scales.
//   - Rejections: climbing, as the concurrency ceiling is hit.
//
// Point the same autoscaler at proxy concurrency instead and it finally reacts:
// replicas climb with the load and the rejections disappear.
//
// Everything except one clearly-labelled number is measured from the real run.
// That one number is service CPU, which we model transparently from measured
// throughput: see --cpu-at-baseline below.
package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/moksheshd/experiments/3b-bit-by-byte/season-01-github-august-17/styles"
)

type config struct {
	ratePerSec  int             // steady arrival rate, requests/sec (open loop)
	limit       int             // per-replica proxy concurrency limit
	steps       []time.Duration // service latencies to sweep (the backend getting slower)
	settle      time.Duration   // per step: let the autoscaler converge before measuring
	window      time.Duration   // per step: measurement window after settling
	cpuBaseline float64         // modeled service CPU% when throughput == arrival rate
	targetCPU   float64         // CPU-watching autoscaler: scale to keep CPU near this
	targetUtil  float64         // concurrency-watching autoscaler: scale to keep util near this
	maxReplicas int             // ceiling on replicas the autoscaler may add
	scale       string          // "both" | "cpu" | "concurrency" | "none"
	csvPath     string
}

// result holds the measured (and one modeled) outcome of a single latency step.
type result struct {
	latency      time.Duration
	offeredConc  int     // arrival-rate x latency: the concurrency the load implies
	peakInFlight int64   // measured peak requests in flight at the proxy
	replicas     int64   // replicas the autoscaler settled on
	capacity     int64   // replicas x limit: the effective concurrency ceiling
	serviceCPU   float64 // MODELED from measured throughput: see modelCPU
	rejectRate   float64 // measured
	throughput   float64 // measured successful req/s
}

const genInterval = 10 * time.Millisecond      // load generator tick granularity
const controlInterval = 200 * time.Millisecond // how often the autoscaler reacts

func main() {
	cfg := parseFlags()
	printHeader(cfg)

	switch cfg.scale {
	case "both":
		cpuRes := runSweep(cfg, "cpu")
		concRes := runSweep(cfg, "concurrency")
		printSweep("Autoscaler watches SERVICE CPU  (the blind spot)", cfg, cpuRes)
		fmt.Println()
		printSweep("Autoscaler watches PROXY CONCURRENCY  (the fix)", cfg, concRes)
		fmt.Println()
		printChart("Service CPU while scaling on CPU (modeled): it only falls", cpuRes,
			func(r result) float64 { return r.serviceCPU }, styles.OK)
		fmt.Println()
		printChart("Requests rejected while scaling on CPU (measured)", cpuRes,
			func(r result) float64 { return r.rejectRate }, styles.Err)
		fmt.Println()
		printChart("Requests rejected while scaling on CONCURRENCY (measured)", concRes,
			func(r result) float64 { return r.rejectRate }, styles.OK)
		fmt.Println()
		printFooter(cfg, cpuRes, concRes)
		if cfg.csvPath != "" {
			writeAndReport(cfg.csvPath, cpuRes, concRes)
		}
	default:
		res := runSweep(cfg, cfg.scale)
		printSweep("Autoscaler signal: "+cfg.scale, cfg, res)
		fmt.Println()
		if cfg.csvPath != "" {
			writeAndReport(cfg.csvPath, res, nil)
		}
	}
}

func parseFlags() config {
	rate := flag.Int("rate", 1000, "steady arrival rate in requests/sec (the load stays flat all run)")
	limit := flag.Int("limit", 100, "per-replica proxy concurrency limit: max requests in flight per replica")
	stepsStr := flag.String("steps-ms", "50,100,200,400", "service latencies to sweep, in ms, comma-separated (the backend getting slower)")
	settleMs := flag.Int("settle-ms", 1500, "per step: time to let the autoscaler converge before measuring")
	windowMs := flag.Int("window-ms", 1500, "per step: measurement window after settling")
	cpuBaseline := flag.Float64("cpu-at-baseline", 40.0,
		"MODELED service CPU%% when throughput equals the arrival rate; CPU scales with measured throughput below that")
	targetCPU := flag.Float64("target-cpu", 70.0, "CPU-watching autoscaler scales to keep CPU near this %%")
	targetUtil := flag.Float64("target-util", 80.0, "concurrency-watching autoscaler scales to keep proxy utilization near this %%")
	maxReplicas := flag.Int("max-replicas", 20, "ceiling on replicas the autoscaler may add")
	scale := flag.String("scale", "both", "autoscaler signal: both | cpu | concurrency | none")
	csvPath := flag.String("csv", "", "optional path to write results as CSV")
	flag.Parse()

	var steps []time.Duration
	for _, s := range strings.Split(*stepsStr, ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		n, err := strconv.Atoi(s)
		if err != nil || n <= 0 {
			fmt.Fprintf(os.Stderr, "invalid --steps-ms value %q: want positive integers like 50,100,200,400\n", s)
			os.Exit(1)
		}
		steps = append(steps, time.Duration(n)*time.Millisecond)
	}
	if len(steps) == 0 {
		fmt.Fprintln(os.Stderr, "--steps-ms must list at least one positive integer")
		os.Exit(1)
	}
	switch *scale {
	case "both", "cpu", "concurrency", "none":
	default:
		fmt.Fprintf(os.Stderr, "invalid --scale %q: want both | cpu | concurrency | none\n", *scale)
		os.Exit(1)
	}

	return config{
		ratePerSec:  *rate,
		limit:       *limit,
		steps:       steps,
		settle:      time.Duration(*settleMs) * time.Millisecond,
		window:      time.Duration(*windowMs) * time.Millisecond,
		cpuBaseline: *cpuBaseline,
		targetCPU:   *targetCPU,
		targetUtil:  *targetUtil,
		maxReplicas: *maxReplicas,
		scale:       *scale,
		csvPath:     *csvPath,
	}
}

// runSweep runs the whole latency sweep once, under one autoscaler signal. The
// replica count is carried forward across steps: latency only rises, so a real
// autoscaler would already be holding the capacity it grew for the previous step.
func runSweep(cfg config, mode string) []result {
	var replicas atomic.Int64
	replicas.Store(1)
	var out []result
	for _, lat := range cfg.steps {
		out = append(out, runStep(cfg, mode, lat, &replicas))
	}
	return out
}

// runStep holds the arrival rate flat at one service latency, runs the autoscaler
// live for settle+window, and measures the steady state over the final window.
//
// The proxy is an atomic in-flight counter checked against a live capacity
// (replicas x limit). Unlike Chapter 2's fixed-size channel, this ceiling has to
// move while the autoscaler adds replicas, so we admit with a compare-and-swap
// against the current capacity instead of a buffered channel.
func runStep(cfg config, mode string, latency time.Duration, replicas *atomic.Int64) result {
	var (
		inFlight  atomic.Int64 // requests in flight at the proxy right now
		peak      atomic.Int64 // peak in flight during the measurement window
		completed atomic.Int64 // lifetime completions, sampled by the autoscaler for throughput
		success   atomic.Int64 // completions during the measurement window
		rejected  atomic.Int64 // rejections during the measurement window
	)
	var measuring atomic.Bool

	capacity := func() int64 { return replicas.Load() * int64(cfg.limit) }

	stop := make(chan struct{})
	var wg sync.WaitGroup

	// Load generator: open loop. Every genInterval it launches a fixed batch of
	// requests, so the arrival rate stays flat no matter how long each request
	// takes. This is the difference that makes the chapter: the client never
	// backs off, so rising latency turns straight into rising concurrency.
	perTick := cfg.ratePerSec * int(genInterval/time.Millisecond) / 1000
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
						ok, n := admit(&inFlight, capacity)
						meas := measuring.Load()
						if !ok {
							if meas {
								rejected.Add(1)
							}
							return
						}
						if meas {
							updatePeak(&peak, n)
						}
						time.Sleep(latency) // the service's work (downstream-bound)
						inFlight.Add(-1)
						completed.Add(1)
						if meas {
							success.Add(1)
						}
					}()
				}
			}
		}
	}()

	// Autoscaler: every controlInterval it reads its one metric and resizes the
	// replica count with the standard HPA rule,
	// desired = ceil(replicas x currentMetric / targetMetric). The only
	// difference between the two runs is which metric it reads.
	wg.Add(1)
	go func() {
		defer wg.Done()
		t := time.NewTicker(controlInterval)
		defer t.Stop()
		lastCompleted := completed.Load()
		lastTime := time.Now()
		for {
			select {
			case <-stop:
				return
			case now := <-t.C:
				c := completed.Load()
				dt := now.Sub(lastTime).Seconds()
				lastTime = now
				thru := 0.0
				if dt > 0 {
					thru = float64(c-lastCompleted) / dt
				}
				lastCompleted = c

				r := replicas.Load()
				switch mode {
				case "cpu":
					// The blind spot. CPU is modeled from throughput and can only
					// fall as latency rises, so this desired is never above r.
					cpu := modelCPU(thru, cfg.ratePerSec, cfg.cpuBaseline)
					desired := int64(math.Ceil(float64(r) * cpu / cfg.targetCPU))
					setReplicas(replicas, desired, cfg.maxReplicas)
				case "concurrency":
					// The fix. Utilization rises as concurrency climbs toward the
					// ceiling, so this desired climbs with the load.
					util := float64(inFlight.Load()) / float64(r*int64(cfg.limit)) * 100
					desired := int64(math.Ceil(float64(r) * util / cfg.targetUtil))
					setReplicas(replicas, desired, cfg.maxReplicas)
				case "none":
					// No autoscaler: the ceiling never moves.
				}
			}
		}
	}()

	// Let the autoscaler converge, then measure the steady state.
	time.Sleep(cfg.settle)
	measuring.Store(true)
	time.Sleep(cfg.window)
	measuring.Store(false)
	close(stop)
	wg.Wait()

	succ := success.Load()
	rej := rejected.Load()
	thru := float64(succ) / cfg.window.Seconds()
	rejRate := 0.0
	if succ+rej > 0 {
		rejRate = float64(rej) / float64(succ+rej) * 100
	}

	return result{
		latency:      latency,
		offeredConc:  cfg.ratePerSec * int(latency/time.Millisecond) / 1000,
		peakInFlight: peak.Load(),
		replicas:     replicas.Load(),
		capacity:     replicas.Load() * int64(cfg.limit),
		serviceCPU:   modelCPU(thru, cfg.ratePerSec, cfg.cpuBaseline),
		rejectRate:   rejRate,
		throughput:   thru,
	}
}

// admit is the proxy's door. It grabs an in-flight slot only if the current
// count is below the live capacity, with a compare-and-swap so a moving ceiling
// is never overshot. A closed door returns false: rejected, like a proxy at its
// concurrency limit returning 503.
func admit(inFlight *atomic.Int64, capacity func() int64) (ok bool, now int64) {
	for {
		n := inFlight.Load()
		if n >= capacity() {
			return false, n
		}
		if inFlight.CompareAndSwap(n, n+1) {
			return true, n + 1
		}
	}
}

func updatePeak(peak *atomic.Int64, v int64) {
	for {
		old := peak.Load()
		if v <= old || peak.CompareAndSwap(old, v) {
			return
		}
	}
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

// modelCPU is the one number in this lab that is modeled rather than measured.
// It says service CPU tracks the real work the service actually completes:
// CPU == (throughput / arrival-rate) x cpuBaseline. When the proxy caps
// concurrency, throughput falls (the capped service clears fewer requests per
// second while the rest pile up waiting), so CPU falls with it. The exact
// baseline percentage is illustrative; the direction is the point: rising
// latency never raises CPU, so a CPU-watching autoscaler never fires.
func modelCPU(throughput float64, ratePerSec int, cpuBaseline float64) float64 {
	if ratePerSec <= 0 {
		return 0
	}
	cpu := throughput / float64(ratePerSec) * cpuBaseline
	if cpu < 0 {
		cpu = 0
	}
	if cpu > 100 {
		cpu = 100
	}
	return cpu
}

func printHeader(cfg config) {
	meta := fmt.Sprintf("arrival rate %d/s (flat)  ·  proxy limit %d/replica  ·  latency %s",
		cfg.ratePerSec, cfg.limit, stepsLabel(cfg.steps))
	fmt.Println()
	fmt.Println(styles.Header("3B: Bit By Byte   Chapter 3: When CPU Says Everything Is Fine", meta))
	fmt.Println()
	fmt.Println(styles.Sub.Render("Client (open loop, flat rate) -> Proxy (concurrency limit) -> Service"))
	fmt.Println(styles.Sub.Render("Hold the arrival rate steady and make the service slower step by step."))
	fmt.Println(styles.Sub.Render("Watch concurrency climb while CPU falls, and see which signal the"))
	fmt.Println(styles.Sub.Render("autoscaler needed to be watching."))
	fmt.Println()
}

func printSweep(title string, cfg config, rs []result) {
	fmt.Println(styles.Bold.Render(title))
	fmt.Println(styles.Bold.Render(
		pad("Latency", 9) + pad("Offered", 9) + pad("In-flight", 11) +
			pad("Replicas", 10) + pad("Svc CPU", 9) + pad("Reject", 9) + "Thru"))
	fmt.Println(styles.Bold.Render(
		pad("(svc)", 9) + pad("conc", 9) + pad("peak", 11) +
			pad("(scaled)", 10) + pad("(model)", 9) + pad("rate", 9) + "req/s"))
	fmt.Println(styles.Rule(64))
	for _, r := range rs {
		row := pad(shortDur(r.latency), 9) +
			pad(fmt.Sprintf("%d", r.offeredConc), 9) +
			pad(fmt.Sprintf("%d", r.peakInFlight), 11) +
			pad(fmt.Sprintf("%d", r.replicas), 10) +
			pad(fmt.Sprintf("%.0f%%", r.serviceCPU), 9) +
			pad(fmt.Sprintf("%.0f%%", r.rejectRate), 9) +
			fmt.Sprintf("%.0f", r.throughput)
		// Green while the system keeps its promises, red once it sheds load.
		if r.rejectRate >= 1 {
			fmt.Println(styles.Err.Render(row))
		} else {
			fmt.Println(styles.OK.Render(row))
		}
	}
}

// printChart draws a labelled row of horizontal bars, one per step, scaled to a
// fixed 0-100 range so charts line up visually.
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
			styles.Sub.Render(pad(shortDur(r.latency), 8)),
			bar,
			styles.Bold.Render(fmt.Sprintf("%.0f%%", v)))
	}
}

func printFooter(cfg config, cpuRes, concRes []result) {
	fmt.Println(styles.Rule(64))
	last := len(cfg.steps) - 1
	cpuWorst := cpuRes[last]
	concWorst := concRes[last]
	fmt.Println(styles.Sub.Render(fmt.Sprintf(
		"At %s per request, the arrival rate never moved. Concurrency did:", shortDur(cfg.steps[last]))))
	fmt.Println(styles.Sub.Render(fmt.Sprintf(
		"the load implies about %d in flight against a %d/replica limit.",
		cpuWorst.offeredConc, cfg.limit)))
	fmt.Println()
	fmt.Println(styles.Sub.Render(fmt.Sprintf(
		"Watching CPU: it fell to %.0f%%, stayed under the %.0f%% target, so the",
		cpuWorst.serviceCPU, cfg.targetCPU)))
	fmt.Println(styles.Sub.Render(fmt.Sprintf(
		"autoscaler held at %d replica(s) and shed %.0f%% of requests.",
		cpuWorst.replicas, cpuWorst.rejectRate)))
	fmt.Println(styles.Sub.Render(fmt.Sprintf(
		"Watching concurrency: it scaled to %d replicas and shed %.0f%%.",
		concWorst.replicas, concWorst.rejectRate)))
	fmt.Println()
	fmt.Println(styles.Sub.Render("Same load, same failure mode, same autoscaler. The only difference is"))
	fmt.Println(styles.Sub.Render("the signal it watched. You cannot scale on a limit you never measured."))
	fmt.Println()
	fmt.Println(styles.Sub.Render("Try it: --scale=none removes the autoscaler; --rate and --steps-ms"))
	fmt.Println(styles.Sub.Render("reshape the load; --target-util moves the concurrency trigger."))
	fmt.Println()
}

func writeAndReport(path string, cpuRes, concRes []result) {
	if err := writeCSV(path, cpuRes, concRes); err != nil {
		fmt.Fprintln(os.Stderr, "could not write CSV:", err)
		os.Exit(1)
	}
	fmt.Println(styles.Sub.Render("Wrote results to " + path))
}

func writeCSV(path string, cpuRes, concRes []result) error {
	var b strings.Builder
	b.WriteString("signal,latency_ms,offered_conc,peak_inflight,replicas,capacity,service_cpu_model_pct,reject_rate_pct,throughput_rps\n")
	writeRows := func(signal string, rs []result) {
		for _, r := range rs {
			b.WriteString(fmt.Sprintf("%s,%d,%d,%d,%d,%d,%.1f,%.1f,%.0f\n",
				signal, r.latency.Milliseconds(), r.offeredConc, r.peakInFlight,
				r.replicas, r.capacity, r.serviceCPU, r.rejectRate, r.throughput))
		}
	}
	writeRows("cpu", cpuRes)
	if concRes != nil {
		writeRows("concurrency", concRes)
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func stepsLabel(steps []time.Duration) string {
	parts := make([]string, len(steps))
	for i, s := range steps {
		parts[i] = shortDur(s)
	}
	return strings.Join(parts, " -> ")
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
