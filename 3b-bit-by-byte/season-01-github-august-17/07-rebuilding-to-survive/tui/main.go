// Command tui is the interactive version of the Chapter 7 capstone.
//
// The full lab (../main.go) walks the whole chain fragile vs resilient in one
// batch run. This interactive version keeps the client link live, because it is
// the capstone's move in miniature: hold the load fixed and flip the mitigation.
//
// A fleet hits a slow endpoint. Start fragile and the reissue-on-timeout bug
// drives the load reaching the endpoint to about ten times baseline. Press s to
// rebuild the client (a sane timeout and a retry budget) and watch the same fleet
// against the same slow endpoint fall back to about baseline. Same load, different
// couplings.
//
// Controls: s (toggle fragile/resilient), q to quit.
package main

import (
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/moksheshd/experiments/3b-bit-by-byte/season-01-github-august-17/styles"
)

const (
	clients  = 500
	interval = 500 * time.Millisecond
	slowResp = 500 * time.Millisecond
	buggyTO  = 50 * time.Millisecond
	saneTO   = 800 * time.Millisecond
	maxTry   = 40
	barWidth = 40
	scaleMax = 12.0
	refresh  = 200 * time.Millisecond
)

type engine struct {
	resilient   atomic.Bool
	winAttempts atomic.Int64
}

func (e *engine) logical() {
	to := buggyTO
	tryCap := maxTry
	if e.resilient.Load() {
		to = saneTO
		tryCap = 2
	}
	done := make(chan struct{})
	got := make(chan struct{}, 1)
	go func() {
		for tries := 0; tries < tryCap; tries++ {
			select {
			case <-done:
				return
			default:
			}
			e.winAttempts.Add(1)
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
}

func (e *engine) run() {
	for i := 0; i < clients; i++ {
		go func() {
			t := time.NewTicker(interval)
			defer t.Stop()
			for range t.C {
				go e.logical()
			}
		}()
	}
}

func (e *engine) snapshot() float64 {
	return float64(e.winAttempts.Swap(0)) / refresh.Seconds()
}

// --- TUI ---

var (
	dim  = styles.Sub
	okS  = styles.OK
	warn = styles.Attn
	bad  = styles.Err
)

type tickMsg time.Time

type model struct {
	e        *engine
	rps      float64
	baseline float64
}

func (m model) Init() tea.Cmd { return tick() }

func tick() tea.Cmd {
	return tea.Tick(refresh, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "s", " ":
			m.e.resilient.Store(!m.e.resilient.Load())
		}
	case tickMsg:
		m.rps = m.e.snapshot()
		return m, tick()
	}
	return m, nil
}

func (m model) View() string {
	ampl := m.rps / m.baseline
	resilient := m.e.resilient.Load()
	mode := "FRAGILE (reissue on timeout)"
	if resilient {
		mode = "RESILIENT (sane timeout + budget)"
	}

	var b strings.Builder
	b.WriteString(styles.Header(
		"3B: Bit By Byte   Chapter 7: The Fix Is Not the Lesson",
		fmt.Sprintf("the client link, live  ·  fleet %d  ·  baseline %.0f req/s  ·  endpoint slow", clients, m.baseline)) + "\n\n")

	b.WriteString(dim.Render("Client: "+mode) + "\n\n")

	style := okS
	if ampl >= 2 {
		style = bad
	} else if ampl >= 1.5 {
		style = warn
	}
	b.WriteString(fmt.Sprintf("  %s %s  %s\n",
		dim.Render("Load reaching endpoint "),
		bar(ampl, scaleMax, style),
		style.Render(fmt.Sprintf("%.1fx  (%.0f req/s)", ampl, m.rps))))

	b.WriteString("\n")
	if resilient {
		b.WriteString(okS.Render("Rebuilt. Same fleet, same slow endpoint, same load, near baseline.") + "\n")
	} else {
		b.WriteString(warn.Render("The storm is running. Same load will bend to ~1x once you rebuild the") + "\n")
		b.WriteString(warn.Render("client. Press s.") + "\n")
	}

	b.WriteString("\n")
	b.WriteString(dim.Render("s: toggle fragile/resilient   q: quit"))
	return b.String()
}

func bar(v, max float64, style lipgloss.Style) string {
	frac := v / max
	if frac > 1 {
		frac = 1
	}
	if frac < 0 {
		frac = 0
	}
	filled := int(frac * barWidth)
	return style.Render(strings.Repeat("█", filled)) + dim.Render(strings.Repeat("·", barWidth-filled))
}

func main() {
	e := &engine{}
	go e.run()
	baseline := float64(clients) / interval.Seconds()
	p := tea.NewProgram(model{e: e, baseline: baseline})
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
