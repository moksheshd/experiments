// Command minimal is the read-this-first version of the Chapter 2 lab.
//
// It keeps only the mechanism and prints one plain table: no flags, no charts,
// no CSV, no latency percentiles. If you want to understand what the experiment
// does, read this file. If you want the full instrumented run, read ../main.go
// instead.
//
// The only dependency is the season's shared `styles` package (lipgloss), so the
// output is colored to match the rest of 3B: Bit By Byte. lipgloss drops color
// automatically when the output is piped, so the table stays clean in a file.
//
// The whole idea, in three parts:
//
//   - The proxy is a bucket of 50 slots (a buffered channel). To reach the
//     service you must take a slot; if the bucket is full you are turned away.
//   - The service does a fixed 20ms of real work per request and counts how many
//     requests are inside it at once.
//   - The client fires requests as fast as it can, at a chosen concurrency, for
//     a fixed window.
//
// Ramp the client past the proxy's limit and two stories split: the service's
// in-flight count pins at 50 and never climbs, while the client's success rate
// falls to roughly limit/concurrency. The service looks idle; the client is
// failing; both are true.
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
	limit      = 50                    // proxy concurrency limit
	serviceDur = 20 * time.Millisecond // work per request inside the service
	window     = 2 * time.Second       // how long each step runs
)

func main() {
	fmt.Println()
	fmt.Println(styles.Header(
		"3B: Bit By Byte   Chapter 2: The Proxy Beside Your Application",
		"Client -> Proxy (limit 50) -> Service (20ms/req)  ·  minimal, read-first version"))
	fmt.Println()

	fmt.Println(styles.Bold.Render(pad("Client", 8) + pad("Svc peak", 11) + pad("Proxy", 9) + "Success"))
	fmt.Println(styles.Bold.Render(pad(" conc", 8) + pad("in-flight", 11) + pad("reject", 9) + "rate"))
	fmt.Println(styles.Rule(36))

	for _, conc := range []int{10, 30, 60, 120} {
		svcPeak, rejected, successRate := runStep(conc)
		row := pad(fmt.Sprintf(" %d", conc), 8) +
			pad(fmt.Sprintf("%d", svcPeak), 11) +
			pad(fmt.Sprintf("%d", rejected), 9) +
			fmt.Sprintf("%.0f%%", successRate)
		// Green while the proxy admits everyone; red once it starts rejecting.
		if rejected > 0 {
			fmt.Println(styles.Err.Render(row))
		} else {
			fmt.Println(styles.OK.Render(row))
		}
	}

	fmt.Println()
	fmt.Println(styles.Sub.Render("The full lab (../main.go) adds a modeled \"service CPU\" column: CPU ="))
	fmt.Println(styles.Sub.Render("(svcPeak / limit) * 35%. That number is modeled, not measured, but the"))
	fmt.Println(styles.Sub.Render("point is structural: the proxy caps the load, so service CPU has a"))
	fmt.Println(styles.Sub.Render("ceiling that client pressure cannot push past."))
	fmt.Println()
}

// runStep drives `conc` clients against the proxy for one window and returns what
// the service saw (peak in-flight), how many requests the proxy rejected, and the
// client's success rate. The proxy is the semaphore; that is the entire trick.
func runStep(conc int) (svcPeak, rejected int64, successRate float64) {
	sem := make(chan struct{}, limit) // the proxy: a bucket of `limit` slots

	var total, success, svcInFlight int64

	deadline := time.Now().Add(window)
	var wg sync.WaitGroup
	wg.Add(conc)
	for i := 0; i < conc; i++ {
		go func() {
			defer wg.Done()
			for time.Now().Before(deadline) {
				atomic.AddInt64(&total, 1)
				if serve(sem, &svcInFlight, &svcPeak) {
					atomic.AddInt64(&success, 1)
				} else {
					atomic.AddInt64(&rejected, 1)
				}
			}
		}()
	}
	wg.Wait()

	successRate = float64(success) / float64(total) * 100
	return svcPeak, rejected, successRate
}

// serve is one request's whole life. Try to take a proxy slot; if the bucket is
// full, we are rejected at the door. Otherwise the service does its work and we
// give the slot back.
func serve(sem chan struct{}, svcInFlight, svcPeak *int64) bool {
	select {
	case sem <- struct{}{}: // got a slot
	default:
		// Proxy full: rejected. Back off one service time before this client
		// tries again (the slot won't free sooner), so the loop doesn't spin.
		time.Sleep(serviceDur)
		return false
	}
	defer func() { <-sem }() // release the slot on the way out

	n := atomic.AddInt64(svcInFlight, 1)
	updatePeak(svcPeak, n)
	time.Sleep(serviceDur) // the service's real work
	atomic.AddInt64(svcInFlight, -1)
	return true
}

// updatePeak raises *peak to v if v is larger, safely across goroutines.
func updatePeak(peak *int64, v int64) {
	for {
		old := atomic.LoadInt64(peak)
		if v <= old || atomic.CompareAndSwapInt64(peak, old, v) {
			return
		}
	}
}

func pad(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
