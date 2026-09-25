# Chapter 7 lab: The Fix Is Not the Lesson (season capstone)

Companion to *3B: Bit By Byte*, Season One, Chapter 7, "The Fix Is Not the
Lesson."

The whole season broke one `Client -> Proxy -> Service` chain on purpose, one
link at a time. This capstone walks the same chain twice under the same offered
load: once **fragile** (the August 17 incident's failure mechanisms, rebuilt in a
small model) and once **resilient** (every mitigation applied), and prints the
two outcomes side by side. The argument it makes physical: the load was not the
whole problem, the couplings were what let it cascade.

## Three ways to run it

| Version | Command | For |
|---------|---------|-----|
| **`minimal/`** | `go run ./minimal` | **Start here.** The client link alone, fragile vs resilient: the season's move in one screen (10x vs 1x). |
| **`.`** (full) | `go run .` | The measured artifact the essay quotes: the whole chain, fragile vs resilient, one before/after table. |
| **`tui/`** | `go run ./tui` | Interactive. Hold the load fixed and press one key to rebuild the client, live. |

## The mechanism, in one idea

Same offered load, two stacks. Each link of the chain is a small, self-contained,
measured experiment that reuses the mechanism from its own chapter, run once with
the incident's coupling and once with the rebuild. Nothing about the load
changes between the two. Only the couplings do.

## The system

Each row of the output is one link, measured under both configurations:

| Link | Chapter | Fragile | Resilient |
|------|---------|---------|-----------|
| Capacity: autoscaler | 3 | scales on CPU (never fires) | scales on proxy concurrency |
| Capacity: failures | 3 | requests wait out the timeout at one replica and fail | enough replicas, so nothing fails |
| The queue: backpressure | 2 | accepts and waits, so served latency climbs toward the timeout | rejects fast what it cannot serve, so served latency stays flat |
| Retries: budget | 4 | immediate, no budget (up to five tries) | backoff, jitter, and a budget of 10% of requests |
| Shared road: bulkhead | 5 | one shared auth pool | the innocent tenant gets its own lane |
| The client: discipline | 6 | reissue on an aggressive timeout | sane timeout plus a budget |

## What to look for

A representative run (defaults):

```text
Link (chapter)                What you watch      Fragile     Resilient
 Capacity: autoscaler (ch3)    proxy replicas      1           5
 Capacity: failures (ch3)      requests failed     64%         0%
 The queue: backpressure (ch2) p95 latency         1166ms      200ms
 Retries: budget (ch4)         load amplification  2.8x        1.1x
 Shared road: bulkhead (ch5)   innocent success    54%         100%
 The client: discipline (ch6)  fleet amplification 10.0x       1.0x
```

Same offered load fed both stacks. Fragile folds link by link: the autoscaler
holds at one replica and most requests wait out the one-second timeout and fail,
served latency climbs toward that timeout, retries multiply the traffic, the
innocent tenant on the shared road is starved, and the client fleet runs ten
times hot. Resilient bends: it scales so nothing fails, rejects fast what it
cannot serve so served latency stays flat, holds retries to the 10% budget,
isolates the innocent lane, and keeps the client near baseline. The system stays
up.

The queue row holds replicas at one in both stacks, so the autoscaler cannot take
the credit: the only difference is waiting versus rejecting fast. Run to run, the
numbers move a little (replicas 5 or 6, innocent success around 50-54%), but the
shape does not.

In the `tui`, hold the load fixed and press `s` to flip the client link from
fragile to resilient and watch the amplification bar collapse from ~10x to ~1x.

## What is measured vs modeled

Everything is measured from the real run: replicas, failed percentage, p95 latency,
the amplification factors, and the innocent tenant's success rate. The one
modeled number is **service CPU** in the capacity link, derived from measured
throughput exactly as in Chapter 3. The point there is directional, not numeric:
CPU only falls as latency rises, so the CPU-watching autoscaler never fires.

The `minimal` and `tui` versions use the client link, where every number is
measured (the amplification factor does not depend on the fleet's absolute size).

## Flags (full lab)

| Flag | Default | What it does |
|------|---------|--------------|
| `--mode` | `both` | Which stack to run: `both` \| `fragile` \| `resilient`. |
| `--csv` | `""` | Also write the results as CSV. |

The per-link parameters (rates, limits, latencies, pool sizes, timeouts, the retry
budget) are set
as constants inside each link function, so the composed chain stays readable. Open
`main.go` and adjust a link to see how its number moves.

## Run it

```bash
git clone git@github.com:moksheshd/experiments.git
cd experiments/3b-bit-by-byte/season-01-github-august-17/07-rebuilding-to-survive

go run ./minimal   # the client link, fragile vs resilient
go run .           # the whole chain, before/after
go run ./tui       # interactive: rebuild the client live
```

## Dependencies

Two, both from the [Charm](https://github.com/charmbracelet) family, wired
through the season's shared `styles` package so every version reads as part of
3B: Bit By Byte:

- [lipgloss](https://github.com/charmbracelet/lipgloss) for terminal styling
  (used by all three versions, via `styles`). It drops color automatically when
  the output is piped, so data stays clean in a file.
- [bubbletea](https://github.com/charmbracelet/bubbletea) for the interactive
  TUI only.

`go run` fetches whatever a given version needs automatically.
