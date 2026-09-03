// Command minimal is the read-this-first version of the Chapter 3 lab.
//
// It keeps only the mechanism and prints one plain table: no autoscaler, no
// flags, no charts, no CSV. If you want to understand what the experiment shows,
// read this file. If you want the autoscaler and the fix, read ../main.go.
//
// The whole idea, in three parts:
//
//   - The client is OPEN-loop: it fires at a fixed arrival rate and never slows
//     down, however long each request takes. The load is held flat all run.
//   - The proxy has a fixed concurrency limit (100). To reach the service you
//     must be one of at most 100 requests in flight; beyond that you are rejected.
//   - The service does a fixed slice of real work per request, and we make that
//     slice slower step by step, the way a downstream dependency degrades.
//
// Hold the arrival rate flat and raise the latency, and two things move in
// opposite directions. Concurrency climbs (it is roughly arrival-rate times
// latency) until it pins against the limit and requests start getting rejected.
// Meanwhile service CPU falls, because the capped service completes less real
// work per second. An autoscaler watching CPU would see a falling number and
// never scale, while the service quietly sheds most of its traffic.
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
	ratePerSec  = 1000                   // steady arrival rate (open loop, never backs off)
	limit       = 100                    // proxy concurrency limit
	cpuBaseline = 40.0                   // modeled service CPU% when throughput == arrival rate
	settle      = 500 * time.Millisecond // let the pipeline fill before measuring
	window      = 1500 * time.Millisecond
)

func main() {
	fmt.Println()
	fmt.Println(styles.Header(
		"3B: Bit By Byte   Chapter 3: When CPU Says Everything Is Fine",
		"Client (flat 1000/s) -> Proxy (limit 100) -> Service  ·  minimal, read-first version"))
	fmt.Println()

	fmt.Println(styles.Bold.Render(pad("Latency", 9) + pad("Offered", 9) + pad("In-flight", 11) + pad("Svc CPU", 9) + "Reject"))
	fmt.Println(styles.Bold.Render(pad("(svc)", 9) + pad("conc", 9) + pad("peak", 11) + pad("(model)", 9) + "rate"))
	fmt.Println(styles.Rule(45))

	for _, latency := range []time.Duration{50 * time.Millisecond, 100 * time.Millisecond, 200 * time.Millisecond, 400 * time.Millisecond} {
		peak, cpu, rejectRate := runStep(latency)
		offered := ratePerSec * int(latency/time.Millisecond) / 1000
		row := pad(shortDur(latency), 9) +
			pad(fmt.Sprintf("%d", offered), 9) +
			pad(fmt.Sprintf("%d", peak), 11) +
			pad(fmt.Sprintf("%.0f%%", cpu), 9) +
			fmt.Sprintf("%.0f%%", rejectRate)
		// Green while the proxy admits everyone; red once it starts rejecting.
		if rejectRate >= 1 {
			fmt.Println(styles.Err.Render(row))
		} else {
			fmt.Println(styles.OK.Render(row))
		}
	}

	fmt.Println()
	fmt.Println(styles.Sub.Render("The arrival rate never changed. Latency did, so concurrency climbed to the"))
	fmt.Println(styles.Sub.Render("limit and requests got rejected, while service CPU only fell. An autoscaler"))
	fmt.Println(styles.Sub.Render("set to scale up when CPU > 70% would never fire. ../main.go adds that"))
	fmt.Println(styles.Sub.Render("autoscaler, then points it at proxy concurrency instead and watches it react."))
	fmt.Println()
}

// runStep holds the arrival rate flat at one service latency and returns the peak
// in-flight count, the modeled service CPU, and the reject rate, all measured over
// a window after the pipeline fills. The proxy is an atomic in-flight counter
// checked against the fixed limit; that gate is the entire trick.
func runStep(latency time.Duration) (peak int64, cpu, rejectRate float64) {
	var (
		inFlight atomic.Int64
		peakV    atomic.Int64
		success  atomic.Int64
		rejected atomic.Int64
	)
	var measuring atomic.Bool

	stop := make(chan struct{})
	var wg sync.WaitGroup

	// Open-loop generator: a fixed batch of arrivals every 10ms, so the rate
	// stays flat no matter how slow the service gets.
	perTick := ratePerSec * 10 / 1000
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
						// The proxy door: admit only if in flight < limit.
						for {
							n := inFlight.Load()
							if n >= limit {
								if meas {
									rejected.Add(1)
								}
								return // rejected at the door
							}
							if inFlight.CompareAndSwap(n, n+1) {
								if meas {
									updatePeak(&peakV, n+1)
								}
								break
							}
						}
						time.Sleep(latency) // the service's work
						inFlight.Add(-1)
						if meas {
							success.Add(1)
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

	succ := success.Load()
	rej := rejected.Load()
	thru := float64(succ) / window.Seconds()
	cpu = thru / float64(ratePerSec) * cpuBaseline
	if succ+rej > 0 {
		rejectRate = float64(rej) / float64(succ+rej) * 100
	}
	return peakV.Load(), cpu, rejectRate
}

// updatePeak raises *peak to v if v is larger, safely across goroutines.
func updatePeak(peak *atomic.Int64, v int64) {
	for {
		old := peak.Load()
		if v <= old || peak.CompareAndSwap(old, v) {
			return
		}
	}
}

func shortDur(d time.Duration) string { return fmt.Sprintf("%dms", d.Milliseconds()) }

func pad(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
