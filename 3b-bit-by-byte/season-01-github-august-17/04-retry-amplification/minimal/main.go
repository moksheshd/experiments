// Command minimal is the read-this-first version of the Chapter 4 lab.
//
// It keeps only the mechanism and prints one plain table: just the naive retry
// storm, no discipline, no flags, no charts, no CSV. If you want to understand
// what the experiment shows, read this file. If you want the disciplined run and
// the fix, read ../main.go.
//
// The whole idea, in three parts:
//
//   - The client is OPEN-loop: every 10ms it launches a fixed batch of LOGICAL
//     requests, so the offered rate stays flat however the service behaves.
//   - The service has a hard concurrency limit (50). Each admitted request does a
//     fixed slice of real work (40ms). Its ceiling on useful work is therefore
//     50 / 40ms, about 1250 successful req/s. When it is full it rejects
//     instantly: fast, cheap backpressure.
//   - On failure, the client retries IMMEDIATELY, up to maxAttempts (5), with no
//     backoff, no jitter, and no budget. Every attempt that reaches the service
//     (admitted or rejected) is counted.
//
// Push the offered rate past the ~1250 the service can clear and watch the
// attempts reaching the service balloon far past the offered rate. That gap is
// the amplification: retries do not create capacity, they just pile more load
// onto a service that was already out of room. Notice the goodput does not rise
// with all those extra attempts. The service was never going to clear more than
// its ceiling; the retries were pure waste, aimed at the thing already drowning.
package main

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/moksheshd/experiments/3b-bit-by-byte/season-01-github-august-17/styles"
)

const (
	limit       = 50                     // service concurrency limit
	serviceDur  = 40 * time.Millisecond  // real work per admitted request
	maxAttempts = 5                      // attempts per logical request (naive: immediate)
	settle      = 800 * time.Millisecond // let the pipeline fill before measuring
	window      = 3000 * time.Millisecond
)

func main() {
	fmt.Println()
	fmt.Println(styles.Header(
		"3B: Bit By Byte   Chapter 4: When Retries Become the Outage",
		"Client (open loop) -> Service (limit 50, ~1250 req/s)  ·  minimal, naive retries only"))
	fmt.Println()

	fmt.Println(styles.Bold.Render(pad("Offered", 10) + pad("Attempts", 11) + pad("Amplify", 11) + pad("Goodput", 10) + "Success"))
	fmt.Println(styles.Bold.Render(pad("req/s", 10) + pad("@service/s", 11) + pad("x offered", 11) + pad("req/s", 10) + "rate"))
	fmt.Println(styles.Rule(52))

	for _, offered := range []int{500, 1000, 2000, 4000} {
		attemptsRate, amp, goodput, successPct := runStep(offered)
		row := pad(fmt.Sprintf("%d", offered), 10) +
			pad(fmt.Sprintf("%.0f", attemptsRate), 11) +
			pad(fmt.Sprintf("%.1fx", amp), 11) +
			pad(fmt.Sprintf("%.0f", goodput), 10) +
			fmt.Sprintf("%.0f%%", successPct)
		// Green while retries stay honest; red once amplification takes off.
		if amp >= 1.5 {
			fmt.Println(styles.Err.Render(row))
		} else {
			fmt.Println(styles.OK.Render(row))
		}
	}

	fmt.Println()
	fmt.Println(styles.Sub.Render("The offered rate crosses the service's ~1250 req/s capacity between rows 2"))
	fmt.Println(styles.Sub.Render("and 3. Past it, attempts balloon past offered while goodput stays pinned at"))
	fmt.Println(styles.Sub.Render("the ceiling: the retries added load, not capacity. ../main.go adds the"))
	fmt.Println(styles.Sub.Render("disciplined run (backoff + jitter + budget + backpressure) and the fix."))
	fmt.Println()
}

// runStep holds the offered rate flat at one level and returns the measured
// attempts/sec at the service, the amplification (attempts / offered), the
// goodput (successful logical req/s), and the logical success rate. The service is
// an atomic in-flight counter checked against the fixed limit; a full service
// rejects instantly and the naive client just tries again.
func runStep(offered int) (attemptsRate, amp, goodput, successPct float64) {
	var (
		inFlight     atomic.Int64
		offeredCount atomic.Int64
		attempts     atomic.Int64
		success      atomic.Int64
	)
	var measuring atomic.Bool

	// One attempt at the service: admit if below the limit, else reject instantly.
	call := func() bool {
		for {
			n := inFlight.Load()
			if n >= limit {
				return false // service full: rejected fast
			}
			if inFlight.CompareAndSwap(n, n+1) {
				break
			}
		}
		time.Sleep(serviceDur) // the service's real work
		inFlight.Add(-1)
		return true
	}

	stop := make(chan struct{})
	var wg sync.WaitGroup

	// Open-loop generator: a fixed batch of logical requests every 10ms.
	perTick := offered * 10 / 1000
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
						if measuring.Load() {
							offeredCount.Add(1)
						}
						// Naive retries: on rejection, try again immediately.
						for attempt := 1; attempt <= maxAttempts; attempt++ {
							meas := measuring.Load()
							if meas {
								attempts.Add(1)
							}
							if call() {
								if meas {
									success.Add(1)
								}
								return
							}
						}
					}()
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

	off := offeredCount.Load()
	att := attempts.Load()
	good := success.Load()
	attemptsRate = float64(att) / window.Seconds()
	goodput = float64(good) / window.Seconds()
	if off > 0 {
		amp = float64(att) / float64(off)
		successPct = float64(good) / float64(off) * 100
	}
	return attemptsRate, amp, goodput, successPct
}

func pad(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
