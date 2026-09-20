// Command minimal is the read-this-first version of the Chapter 6 lab.
//
// It keeps only the mechanism and prints one plain table: no failover, no
// client-side fixes, no flags, no charts, no CSV. If you want to understand what
// the experiment shows, read this file. If you want failover and the fixes, read
// ../main.go.
//
// The whole idea, in three parts:
//
//   - A fleet of identical clients hits one endpoint on a steady heartbeat. When
//     the endpoint is healthy, that is a calm baseline: one beat, one request.
//   - The endpoint is faulted so its responses are slow, past the client timeout.
//   - Each client has the bug: on a timeout it reissues immediately, without
//     canceling the request still in flight. A slow response spawns a stack of
//     duplicates instead of replacing one request with another.
//
// Watch the attempts reaching the endpoint climb toward ten times the baseline,
// with nothing actually failing: the clients are simply generating ten times the
// load off one slow response. Point that firehose at a real, capacity-limited
// service and it falls over. That is what hit the Copilot Token Service.
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
	clients  = 1000                    // fleet size
	interval = 500 * time.Millisecond  // per-client heartbeat: one logical request per interval
	slowResp = 500 * time.Millisecond  // faulted endpoint response time (past the timeout)
	fastResp = 20 * time.Millisecond   // healthy endpoint response time
	timeout  = 50 * time.Millisecond   // the buggy client's aggressive timeout
	maxTry   = 40                      // safety cap on attempts per logical request
	settle   = 800 * time.Millisecond  // warm-up before measuring
	window   = 2000 * time.Millisecond // measurement window
)

func main() {
	fmt.Println()
	fmt.Println(styles.Header(
		"3B: Bit By Byte   Chapter 6: The Copilot Retry Storm",
		"Fleet of 1000 clients -> one endpoint  ·  minimal, read-first version"))
	fmt.Println()

	baseline := float64(clients) / interval.Seconds()
	fmt.Println(styles.Bold.Render(fmt.Sprintf("Baseline (steady, healthy world): %.0f req/s", baseline)))
	fmt.Println()
	fmt.Println(styles.Bold.Render(pad("Scenario", 30) + pad("Endpoint", 11) + "Amplify"))
	fmt.Println(styles.Bold.Render(pad("", 30) + pad("req/s", 11) + "x base"))
	fmt.Println(styles.Rule(48))

	for _, s := range []struct {
		name string
		slow bool
	}{
		{"Healthy endpoint", false},
		{"Slow endpoint + buggy retries", true},
	} {
		rps := runScenario(s.slow)
		ampl := rps / baseline
		row := pad(s.name, 30) + pad(fmt.Sprintf("%.0f", rps), 11) + fmt.Sprintf("%.1fx", ampl)
		if ampl >= 2 {
			fmt.Println(styles.Err.Render(row))
		} else {
			fmt.Println(styles.OK.Render(row))
		}
	}

	fmt.Println()
	fmt.Println(styles.Sub.Render("A slow response with a timeout of one tenth its length spawns about ten"))
	fmt.Println(styles.Sub.Render("attempts per request. ../main.go adds a fresh-region failover (which does"))
	fmt.Println(styles.Sub.Render("nothing, because the clients are the load) and the client-side fixes that"))
	fmt.Println(styles.Sub.Render("actually flatten the storm."))
	fmt.Println()
}

// runScenario spins up the fleet and returns the measured attempts/sec reaching
// the endpoint. When slow is true, the endpoint is faulted and the clients run the
// reissue-on-timeout bug; that is the entire trick.
func runScenario(slow bool) float64 {
	respTime := fastResp
	if slow {
		respTime = slowResp
	}
	var attempts atomic.Int64
	var measuring atomic.Bool

	// The endpoint: it simply takes respTime to answer. No capacity limit here, so
	// the only fault is latency and the payload is pure amplification.
	serve := func() { time.Sleep(respTime) }

	// One logical request: fire an attempt, and if it does not answer within the
	// timeout, reissue immediately (the bug), until one attempt answers.
	logical := func() {
		meas := measuring.Load()
		done := make(chan struct{})
		got := make(chan struct{}, 1)
		var firer sync.WaitGroup
		firer.Add(1)
		go func() {
			defer firer.Done()
			for tries := 0; tries < maxTry; tries++ {
				select {
				case <-done:
					return
				default:
				}
				if meas {
					attempts.Add(1)
				}
				go func() {
					serve()
					select {
					case got <- struct{}{}:
					default:
					}
				}()
				select {
				case <-done:
					return
				case <-time.After(timeout): // wait, then reissue: the bug
				}
			}
		}()
		select {
		case <-got:
		case <-time.After(slowResp + 2*time.Second):
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

	return float64(attempts.Load()) / window.Seconds()
}

func pad(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
