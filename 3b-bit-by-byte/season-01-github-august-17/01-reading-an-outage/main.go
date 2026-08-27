// Command worksheet is the companion lab for 3B: Bit By Byte, Chapter 1,
// "What Happened? Learning to Read an Outage."
//
// It is not a break-it experiment. It is a thinking tool: it walks the
// documented August 17, 2026 GitHub incident one event at a time and asks the
// only question that matters when you read a capacity related outage:
//
//	What ran out here, and what did that cause to run out next?
//
// The answers are intentionally blank in this repo on the day Chapter 1 ships.
// They fill in, one row at a time, as each later chapter is published. So the
// worksheet is the same discipline as the essay: we do not hand you the causes
// before we have earned them.
package main

import (
	"bufio"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/moksheshd/experiments/3b-bit-by-byte/season-01-github-august-17/styles"
)

//go:embed timeline.json
var timelineJSON []byte

const wrapWidth = 66

type answer struct {
	WhatRanOut string `json:"what_ran_out"`
	Mechanism  string `json:"mechanism"`
}

type event struct {
	ID         string `json:"id"`
	Time       string `json:"time"`
	Observed   string `json:"observed"`
	Source     string `json:"source"`
	Bucket     string `json:"bucket"`
	Kind       string `json:"kind"` // "context" or "causal"
	UnpackedIn int    `json:"unpacked_in"`
	Answer     answer `json:"answer"`
}

type timeline struct {
	Incident string   `json:"incident"`
	Window   string   `json:"window_utc"`
	Duration string   `json:"duration"`
	Note     string   `json:"note"`
	Sources  []string `json:"sources"`
	Events   []event  `json:"events"`
}

func main() {
	answersUpTo := flag.Int("answers", 0,
		"reveal answers for chapters <= N (0 hides all; try --answers=2 after you have read Chapter 2)")
	interactive := flag.Bool("interactive", false,
		"walk one event at a time; type your own guess, press Enter to continue")
	flag.Parse()

	var tl timeline
	if err := json.Unmarshal(timelineJSON, &tl); err != nil {
		fmt.Fprintln(os.Stderr, "could not parse embedded timeline.json:", err)
		os.Exit(1)
	}

	printHeader(tl, *answersUpTo)

	in := bufio.NewScanner(os.Stdin)
	for i, e := range tl.Events {
		printEvent(i+1, e)
		if e.Kind == "causal" {
			if *interactive {
				fmt.Print(styles.Sub.Render("    your guess > "))
				if !in.Scan() {
					fmt.Println()
					break
				}
			}
			printAnswer(e, *answersUpTo)
		}
		fmt.Println()
	}

	printFooter(*answersUpTo, len(tl.Events))
}

func printHeader(tl timeline, answersUpTo int) {
	meta := fmt.Sprintf("%s  ·  %s  ·  %s", tl.Incident, tl.Window, tl.Duration)
	fmt.Println()
	fmt.Println(styles.Header("3B: Bit By Byte   Chapter 1: Reading an Outage", meta))
	fmt.Println()
	fmt.Println(styles.Sub.Render("For each CAUSAL event, ask: what constraint was hit here, and what"))
	fmt.Println(styles.Sub.Render("did it cause to give next? \"What ran out?\" is the shorthand, as long"))
	fmt.Println(styles.Sub.Render("as you remember not every constraint is literal exhaustion."))
	if answersUpTo <= 0 {
		fmt.Println(styles.Sub.Render("Answers are hidden. Reveal what a chapter has earned with --answers=N."))
	} else {
		fmt.Println(styles.Attn.Render(fmt.Sprintf("Revealing answers unpacked through Chapter %d.", answersUpTo)))
	}
	fmt.Println()
}

func printEvent(n int, e event) {
	idx := styles.Index.Render(fmt.Sprintf("[%2d]", n))
	tag := styles.Sub.Render(e.Source)
	fmt.Printf("%s %s  %s\n", idx, styles.Bold.Render(pad(e.Time, 14)), tag)

	// Color line by line so lipgloss never pads a multi-line block to width
	// (which would leave trailing spaces). The map recedes in muted gray; the
	// causal questions stand out in the default color.
	for _, ln := range wrapLines(e.Observed, wrapWidth) {
		if e.Kind == "context" {
			fmt.Printf("     %s\n", styles.Sub.Render(ln))
		} else {
			fmt.Printf("     %s\n", ln)
		}
	}
	if e.Kind == "causal" {
		fmt.Printf("     %s\n", styles.Attn.Render("what ran out? ______________"))
	}
}

func printAnswer(e event, answersUpTo int) {
	// A filled answer is the signal that its chapter has shipped (each row is
	// filled the day its chapter publishes). Until then, the blank is the hook.
	filled := strings.TrimSpace(e.Answer.WhatRanOut) != ""

	if !filled {
		ch := styles.Index.Render(fmt.Sprintf("Chapter %d", e.UnpackedIn))
		fmt.Printf("     %s%s%s\n",
			styles.Sub.Render("-> "),
			ch,
			styles.Sub.Render(" answers this. Not shipped yet. Follow along and this blank fills itself in."))
		return
	}

	if answersUpTo < e.UnpackedIn {
		fmt.Printf("     %s\n", styles.Sub.Render(
			fmt.Sprintf("-> Chapter %d covers this. Read it, then rerun with --answers=%d.", e.UnpackedIn, e.UnpackedIn)))
		return
	}

	fmt.Printf("     %s\n", styles.OK.Render("-> ran out: "+e.Answer.WhatRanOut))
	if m := strings.TrimSpace(e.Answer.Mechanism); m != "" {
		fmt.Printf("     %s\n", styles.OK.Render("-> mechanism: "+m))
	}
	fmt.Printf("     %s\n", styles.Sub.Render(fmt.Sprintf("(Chapter %d)", e.UnpackedIn)))
}

func printFooter(answersUpTo, total int) {
	fmt.Println(styles.Rule(72))
	fmt.Println(styles.Sub.Render(fmt.Sprintf("%d events, and every blank above is a promise.", total)))
	fmt.Printf("%s%s%s\n",
		styles.Sub.Render("Follow "),
		styles.Title.Render("3B: Bit By Byte"),
		styles.Sub.Render(" and each answer fills in as its chapter ships."))
	fmt.Println(styles.Sub.Render("Read a chapter, then rerun with --answers=<n>. Walk it with --interactive."))
	fmt.Println()
}

// pad right-pads s to width columns.
func pad(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// wrapLines re-flows s into lines no wider than width columns. Lines carry no
// indentation; the caller adds it, so each returned line is single-line and
// safe to pass to a lipgloss style without triggering block padding.
func wrapLines(s string, width int) []string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{s}
	}
	var lines []string
	var cur strings.Builder
	for i, w := range words {
		if cur.Len() > 0 && cur.Len()+1+len(w) > width {
			lines = append(lines, cur.String())
			cur.Reset()
		} else if i > 0 && cur.Len() > 0 {
			cur.WriteString(" ")
		}
		cur.WriteString(w)
	}
	if cur.Len() > 0 {
		lines = append(lines, cur.String())
	}
	return lines
}
