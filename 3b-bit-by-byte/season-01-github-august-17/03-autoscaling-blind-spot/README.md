# Chapter 3 lab: When CPU Says Everything Is Fine

Companion to *3B: Bit By Byte*, Season One, Chapter 3,
"When CPU Says Everything Is Fine."

This lab reproduces the autoscaling blind spot from the August 17, 2026 GitHub
incident: an autoscaler that watches the wrong signal. GitHub reported that a
sidecar reached its concurrency limit and did not scale, because the scaling
policy watched the host service rather than the sidecar's own limits. We build
the smallest system where that happens on purpose and watch it fail, then fix it
by changing the one thing that mattered: which signal the autoscaler watched.

It grows directly out of the Chapter 2 lab, with two changes:

- The client is now **open-loop**: it fires at a fixed arrival rate and never
  slows down, however long each request takes. The load stays flat all run.
- There is an **autoscaler**. It reads one metric and resizes the number of
  proxy replicas (which multiplies the concurrency limit). We run the whole
  sweep twice: once watching **service CPU** (the blind spot), once watching
  **proxy concurrency** (the fix).

## Three ways to run it

| Version | Command | For |
|---------|---------|-----|
| **`minimal/`** | `go run ./minimal` | **Start here.** Just the blind spot: flat load, rising latency, and a CPU number that only falls while requests are rejected. |
| **`.`** (full) | `go run .` | The measured artifact the essay quotes: both autoscaler signals, side by side, plus charts and CSV. |
| **`tui/`** | `go run ./tui` | Interactive. Turn the latency knob and toggle the autoscaler signal live: watch CPU-watching do nothing, then switch to concurrency and watch the rejects vanish. |

Read `minimal/main.go` first; it is the same mechanism with the autoscaler and
everything optional stripped out.

## The experiment, in one idea

Hold the arrival rate flat and make the service slower, step by step. Two numbers
move in opposite directions:

- **Concurrency climbs.** It is roughly arrival-rate times latency (Little's
  Law), so as latency rises the number of requests in flight rises with it,
  until it pins against the proxy's limit and new requests are rejected.
- **Service CPU falls.** The capped service completes *less* real work per
  second while requests pile up waiting, so its CPU drops.

An autoscaler watching CPU sees a falling, healthy-looking number and never
scales. An autoscaler watching concurrency sees the ceiling being approached and
adds replicas. Same load, same failure mode, same autoscaler: the only
difference is the signal.

## The system

A real, in-process, **open-loop** `Client -> Proxy -> Service` with an autoscaler:

```text
Client (flat 1000/s)  ->  Proxy (limit 100/replica, resized by autoscaler)  ->  Service (latency we dial up)
```

- **Client:** an open-loop generator. Every 10ms it launches a fixed batch of
  requests, so the arrival rate is flat regardless of how long requests take.
  This is the key difference from Chapter 2's closed loop: the client never backs
  off, so rising latency turns straight into rising concurrency.
- **Proxy:** an atomic in-flight counter checked against a **live** capacity
  (`replicas x limit`). Chapter 2 used a fixed-size channel; here the ceiling has
  to move while the autoscaler adds replicas, so admission is a compare-and-swap
  against the current capacity.
- **Service:** a fixed slice of real work (`time.Sleep`) per request. We raise
  that slice step by step, the way a downstream dependency degrades.
- **Autoscaler:** a control loop that every 200ms reads one metric and resizes
  replicas with the standard HPA rule,
  `desired = ceil(replicas x currentMetric / targetMetric)`. On CPU it never
  fires; on concurrency it tracks the load.

## What to look for

With the defaults (arrival rate 1000/s, proxy limit 100/replica), the service
latency ramps `50 -> 100 -> 200 -> 400` ms. A representative run:

```text
Autoscaler watches SERVICE CPU  (the blind spot)
Latency  Offered  In-flight  Replicas  Svc CPU  Reject   Thru
 50ms    50       60         1         40%      0%       993
 100ms   100      100        1         36%      8%       909
 200ms   200      100        1         19%      51%      487
 400ms   400      100        1         11%      74%      267

Autoscaler watches PROXY CONCURRENCY  (the fix)
 50ms    50       60         1         40%      0%       1000
 100ms   100      110        2         40%      0%       1007
 200ms   200      210        3         40%      0%       993
 400ms   400      410        5         40%      0%       1007
```

Read the two tables together. The offered concurrency (arrival-rate x latency)
climbs 50 -> 400 in both. Watching CPU, the in-flight count pins at the 100 limit,
CPU *falls* to 11%, the autoscaler holds at 1 replica, and 74% of requests are
shed. Watching concurrency, the autoscaler grows to 5 replicas, CPU holds near
its 40% baseline, and nothing is rejected. Same load. The signal was the whole
game.

In the `tui`, this is something you feel: raise the latency on the CPU signal and
watch the reject bar climb while the CPU bar shrinks and replicas stay at 1. Press
`s` to switch to concurrency and watch replicas jump and the rejects vanish.

## What is measured vs modeled

Everything except one number is measured from the real run: peak in-flight,
replicas, reject rate, and throughput.

The one modeled number is **service CPU**. Real per-handler CPU is noisy and
machine-dependent, which would make the shipped result unreproducible. So we
model it transparently from the measured throughput:
`CPU = (throughput / arrival-rate) x cpu-at-baseline` (baseline 40%). The claim
the model makes honest is directional, not numeric: because the capped service
completes fewer requests per second as latency rises, its CPU can only fall, so a
CPU-watching autoscaler never fires. That falling number is the "green CPU" the
chapter is named for.

## Flags (full lab)

| Flag | Default | What it does |
|------|---------|--------------|
| `--rate` | `1000` | Steady arrival rate in req/s. The load stays flat all run. |
| `--limit` | `100` | Per-replica proxy concurrency limit. |
| `--steps-ms` | `50,100,200,400` | Service latencies to sweep (the backend getting slower). |
| `--settle-ms` | `1500` | Per step: time to let the autoscaler converge before measuring. |
| `--window-ms` | `1500` | Per step: measurement window after settling. |
| `--cpu-at-baseline` | `40` | Modeled service CPU% when throughput equals the arrival rate. |
| `--target-cpu` | `70` | CPU-watching autoscaler target. |
| `--target-util` | `80` | Concurrency-watching autoscaler target. |
| `--max-replicas` | `20` | Ceiling on replicas the autoscaler may add. |
| `--scale` | `both` | Autoscaler signal: `both` \| `cpu` \| `concurrency` \| `none`. |
| `--csv` | `""` | Also write the results as CSV. |

## Two variations worth running

```bash
# No autoscaler at all: the ceiling never moves, so the CPU-signal and no-signal
# runs look the same. Proof that the CPU-watching autoscaler was doing nothing.
go run . --scale=none

# A faster arrival rate hits the limit at a lower latency, because concurrency is
# arrival-rate x latency. Watch the CPU-watched run start shedding sooner.
go run . --rate=2000
```

## Run it

```bash
git clone git@github.com:moksheshd/experiments.git
cd experiments/3b-bit-by-byte/season-01-github-august-17/03-autoscaling-blind-spot

go run ./minimal   # the blind spot, simplest form
go run .           # both signals, side by side (what the essay quotes)
go run ./tui       # interactive: dial latency, toggle the signal
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
