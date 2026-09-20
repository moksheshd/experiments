// Command minimal is the read-this-first version of the Chapter 5 lab.
//
// It keeps only the mechanism and prints one plain table: just the shared-pool
// scenario, no bulkhead, no flags, no charts, no CSV. If you want to understand
// what the experiment shows, read this file. If you want the fix, read ../main.go.
//
// The whole idea, in three parts:
//
//   - Three endpoints (/notes, /tasks, /profile) are completely independent. Each
//     has its own handler that does a trivially fast slice of work (5ms) and
//     shares no logic with the others.
//   - In front of all three sits one shared AUTH gate: a semaphore with 30 slots.
//     Every request must pass auth before it reaches its handler. Each auth check
//     holds a slot for 20ms, so the gate clears about 30/0.02 = 1500 auth/s in
//     total. A request that cannot get a slot within 50ms is rejected at the door.
//   - The load is open-loop and flat. /notes and /tasks are light and innocent
//     (200 req/s each). /profile floods (3000 req/s).
//
// /profile's flood alone is twice the gate's whole capacity, so it keeps the
// shared slots full. /notes and /tasks then fail too, even though their own
// handlers are perfect and their own load is tiny: they are not three independent
// systems, they are three tenants of one road. Watch all three success rates fall
// together. ../main.go gives each endpoint its own gate and contains the damage.
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
	authSlots   = 30                    // shared gate concurrency limit
	authWork    = 20 * time.Millisecond // each auth check holds a slot this long
	authTimeout = 50 * time.Millisecond // give up waiting for a slot after this
	handlerWork = 5 * time.Millisecond  // each endpoint's own (fast, healthy) work
	settle      = 1000 * time.Millisecond
	window      = 3000 * time.Millisecond
)

type endpoint struct {
	name string
	rps  int
}

func main() {
	fmt.Println()
	fmt.Println(styles.Header(
		"3B: Bit By Byte   Chapter 5: The Road Everyone Shares",
		"Three endpoints, one shared auth gate (30 slots)  ·  minimal, read-first version"))
	fmt.Println()

	eps := []endpoint{
		{"/notes", 200},    // light, innocent
		{"/tasks", 200},    // light, innocent
		{"/profile", 3000}, // the flood
	}

	// One shared gate for all three: the road everyone shares.
	gate := make(chan struct{}, authSlots)

	// Per-endpoint counters, keyed by index.
	success := make([]atomic.Int64, len(eps))
	reject := make([]atomic.Int64, len(eps))
	var measuring atomic.Bool

	stop := make(chan struct{})
	var wg sync.WaitGroup

	// One open-loop generator per endpoint: a fixed batch every 10ms, so each
	// offered rate stays flat no matter how the gate behaves.
	for i, ep := range eps {
		perTick := ep.rps * 10 / 1000
		if perTick < 1 {
			perTick = 1
		}
		wg.Add(1)
		go func(idx, batch int) {
			defer wg.Done()
			t := time.NewTicker(10 * time.Millisecond)
			defer t.Stop()
			for {
				select {
				case <-stop:
					return
				case <-t.C:
					for j := 0; j < batch; j++ {
						wg.Add(1)
						go func() {
							defer wg.Done()
							handle(gate, &success[idx], &reject[idx], &measuring)
						}()
					}
				}
			}
		}(i, perTick)
	}

	time.Sleep(settle)
	measuring.Store(true)
	time.Sleep(window)
	measuring.Store(false)
	close(stop)
	wg.Wait()

	fmt.Println(styles.Bold.Render(pad("Endpoint", 11) + pad("Offered", 9) + pad("Success", 10) + "Admit"))
	fmt.Println(styles.Bold.Render(pad("", 11) + pad("req/s", 9) + pad("rate", 10) + "req/s"))
	fmt.Println(styles.Rule(38))
	for i, ep := range eps {
		succ := success[i].Load()
		rej := reject[i].Load()
		pct := 0.0
		if succ+rej > 0 {
			pct = float64(succ) / float64(succ+rej) * 100
		}
		row := pad(ep.name, 11) +
			pad(fmt.Sprintf("%d", ep.rps), 9) +
			pad(fmt.Sprintf("%.0f%%", pct), 10) +
			fmt.Sprintf("%.0f", float64(succ)/window.Seconds())
		// Green while the endpoint keeps its promises, red once it sheds load.
		if pct >= 90 {
			fmt.Println(styles.OK.Render(row))
		} else {
			fmt.Println(styles.Err.Render(row))
		}
	}

	fmt.Println()
	fmt.Println(styles.Sub.Render("Only /profile floods, but all three fall together: they share one auth gate,"))
	fmt.Println(styles.Sub.Render("so the flood starves /notes and /tasks even though their handlers are perfect"))
	fmt.Println(styles.Sub.Render("and their load is tiny. ../main.go gives each endpoint its own gate and shows"))
	fmt.Println(styles.Sub.Render("/notes and /tasks snapping back to ~100% while only /profile stays down."))
	fmt.Println()
}

// handle is one request's whole life: authenticate at the shared gate, and only
// if that succeeds run the endpoint's own fast handler. The handler is never the
// bottleneck; the gate is. Acquire a slot within authTimeout or be rejected.
func handle(gate chan struct{}, success, reject *atomic.Int64, measuring *atomic.Bool) {
	t := time.NewTimer(authTimeout)
	select {
	case gate <- struct{}{}:
		t.Stop()
		time.Sleep(authWork) // the auth check, holding a shared slot
		<-gate
	case <-t.C:
		if measuring.Load() {
			reject.Add(1) // never got a slot: auth failure at the door
		}
		return
	}
	time.Sleep(handlerWork) // the endpoint's own work, healthy and quick
	if measuring.Load() {
		success.Add(1)
	}
}

func pad(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
