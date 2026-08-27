# Chapter 1 lab: Reading an Outage

Companion to *3B: Bit By Byte*, Season One, Chapter 1,
"What Happened? Learning to Read an Outage."

This is not a break-it experiment. It is a thinking tool. Chapters 2 onward
reproduce real failure mechanics (a saturating proxy, retry amplification,
cascades); this one reproduces the *method* the whole season is built on:
reading an incident by asking, at each step, what ran out, and what that pushed
over next.

## What it is

- `timeline.json` is the August 17, 2026 GitHub incident as data. Every
  `observed` line is drawn only from GitHub's incident report or status page
  (the two `sources` in the file). Nothing is inferred.
- `main.go` walks that timeline and, for each *causal* event, prompts you with
  the only question that matters: **what ran out here?**

## Run it

```bash
git clone git@github.com:moksheshd/experiments.git
cd experiments/3b-bit-by-byte/season-01-github-august-17/01-reading-an-outage
go run .
```

Walk it one event at a time and write your own answers:

```bash
go run . --interactive
```

## Why the answers are blank

On the day Chapter 1 ships, every answer in `timeline.json` is empty on purpose.
Chapter 1 is the map, not the explanation. The essay deliberately refuses to
hand you the causal chain in one paragraph, and so does this lab.

The answers fill in **one row at a time, as each later chapter publishes**. Once
you have read Chapter *N*, reveal what it earned:

```bash
go run . --answers=2   # after Chapter 2
go run . --answers=7   # the whole chain, once the season is complete
```

An answer records two things: the resource that ran out (if any) and the
`mechanism`. Not every link is exhaustion. Some are a misconfiguration or an
amplification, and the worksheet will say so rather than force every failure
into "something ran out." That distinction is the point of Chapter 1.

## Dependencies

One: [lipgloss](https://github.com/charmbracelet/lipgloss) for terminal styling,
shared across the season through the `styles` package one level up. `timeline.json`
is embedded at build time, so the program runs from anywhere, and `go run .`
fetches what it needs automatically.
