// Command minimal is the read-this-first version of the Chapter 7 capstone.
//
// The full lab (../main.go) walks the entire failure chain twice, fragile and
// resilient, and prints a before/after table across all five links. This minimal
// version keeps just the last and most visceral link, the client, because it
// shows the whole season's move in one screen: take the exact same load, flip one
// mitigation, and watch the shape change.
//
// The setup is a fleet of clients hitting a slow endpoint. Fragile clients have
// the reissue-on-timeout bug and manufacture a tenfold storm. Resilient clients
// use a sane timeout and a small retry budget, so the same fleet against the same
// slow endpoint stays near its baseline. Same load, different couplings.
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
	clients  = 500
	interval = 500 * time.Millisecond
	slowResp = 500 * time.Millisecond
	buggyTO  = 50 * time.Millisecond  // aggressive timeout: the bug
	saneTO   = 800 * time.Millisecond // longer than a slow response: the fix
	maxTry   = 40
	settle   = 800 * time.Millisecond
	window   = 1500 * time.Millisecond
)

func main() {
	fmt.Println()
	fmt.Println(styles.Header(
		"3B: Bit By Byte   Chapter 7: The Fix Is Not the Lesson",
		"the client link, fragile vs resilient  ·  minimal, read-first version"))
	fmt.Println()

	baseline := float64(clients) / interval.Seconds()
	fmt.Println(styles.Bold.Render(fmt.Sprintf("Same load both runs: fleet of %d, baseline %.0f req/s, endpoint slow (%s)",
		clients, baseline, slowResp)))
	fmt.Println()
	fmt.Println(styles.Bold.Render(pad("Client behavior", 36) + pad("Load reaching endpoint", 24) + "Amplification"))
	fmt.Println(styles.Rule(64))

	for _, s := range []struct {
		name      string
		resilient bool
	}{
		{"Fragile (reissue on timeout)", false},
		{"Resilient (sane timeout + budget)", true},
	} {
		rps := runClient(s.resilient, baseline)
		ampl := rps / baseline
		row := pad(s.name, 36) + pad(fmt.Sprintf("%.0f req/s", rps), 24) + fmt.Sprintf("%.1fx", ampl)
		if ampl >= 1.5 {
			fmt.Println(styles.Err.Render(row))
		} else {
			fmt.Println(styles.OK.Render(row))
		}
	}

	fmt.Println()
	fmt.Println(styles.Sub.Render("Same fleet, same slow endpoint, same offered load. The only difference is"))
	fmt.Println(styles.Sub.Render("client discipline, and it is the difference between a tenfold storm and a"))
	fmt.Println(styles.Sub.Render("flat baseline. ../main.go applies every mitigation to every link and shows"))
	fmt.Println(styles.Sub.Render("the whole chain fold or bend under one load."))
	fmt.Println()
}

// runClient spins up the fleet under one policy and returns the attempts/sec
// reaching the endpoint. Fragile reissues on the aggressive timeout with no cap;
// resilient waits past a slow response and caps retries at one.
func runClient(resilient bool, baseline float64) float64 {
	to := buggyTO
	tryCap := maxTry
	if resilient {
		to = saneTO
		tryCap = 2 // one try plus one retry, then fail fast: the budget
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
			for tries := 0; tries < tryCap; tries++ {
				select {
				case <-done:
					return
				default:
				}
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

	return float64(attempts.Load()) / window.Seconds()
}

func pad(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
