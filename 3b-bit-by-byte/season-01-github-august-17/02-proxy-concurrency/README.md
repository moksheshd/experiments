# Chapter 2 lab: The Proxy Beside Your Application

Companion to *3B: Bit By Byte*, Season One, Chapter 2,
"The Proxy Beside Your Application."

This is the season's first break-it lab. It reproduces one mechanism from the
August 17, 2026 GitHub incident: a proxy with a concurrency limit saturates
while the service behind it stays comfortably idle. The client sees failures.
The service's own load never rises past the limit. Two graphs, opposite stories,
same system.

## Three ways to run it

| Version | Command | For |
|---------|---------|-----|
| **`minimal/`** | `go run ./minimal` | **Start here.** ~90 lines: just the mechanism and one colored table. |
| **`.`** (full) | `go run .` | The measured artifact the essay quotes: charts, latency percentiles, CSV, tunable flags. |
| **`tui/`** | `go run ./tui` | Interactive. Turn the client-load knob live and watch the service freeze at the limit while the client starts failing. |

If the code feels large, that is because the full `main.go` is mostly packaging.
Read `minimal/main.go` first; it is the same experiment with everything optional
stripped out.

## The mechanism, in six lines

The entire experiment is this. The proxy is a bucket of 50 slots (a buffered
channel). One request's whole life:

```go
sem := make(chan struct{}, 50) // the proxy: a bucket of 50 slots

select {
case sem <- struct{}{}: // got a slot? come in
	defer func() { <-sem }() // give it back when done
	time.Sleep(20 * time.Millisecond) // the service does its work
	return true
default:
	return false // bucket full? rejected at the door
}
```

Grab a slot, do the work, release it. If you cannot grab one, you are turned
away. **Everything else in `main.go` is measuring this and drawing charts.**

## The system

A real, in-process `Client -> Proxy -> Service`:

```text
Client (N goroutines)  ->  Proxy (semaphore, limit=50)  ->  Service (20ms/req)
```

- **Service:** a function that does a fixed slice of work (a real `time.Sleep`)
  per request, and tracks how many requests are in flight inside it.
- **Proxy:** a counting semaphore (a buffered channel of size `--limit`) that
  enforces a hard concurrency limit before forwarding. This is the "model it
  with a semaphore" from the essay: the same constraint Envoy enforces in real
  life, reproduced with the simplest thing that has the same shape. We are not
  reproducing Envoy's internals, only its capacity behavior. When the semaphore
  is full, the request is rejected at the door (or, with `--queue`, made to
  wait).
- **Client:** a pool of goroutines that drive the proxy in a closed loop for a
  fixed window, at a chosen concurrency.

## What's essential vs. scaffolding

In the full `main.go`, the mechanism is about 30 lines. The rest is packaging.
This is the map, so you know what to read and what to skip:

| Part of `main.go` | ~Lines | Is it the experiment? |
|-------------------|--------|-----------------------|
| Semaphore + `serve` + client loop | ~30 | **Yes, this is it.** |
| `parseFlags` (the `--limit`, `--steps`, ... knobs) | ~50 | No, config |
| `printTable` / `printChart` / header / footer | ~90 | No, output |
| lipgloss styling (colors, bars, padding) | scattered | No, decoration |
| `percentile`, p50/p95 latency | ~20 | No, extra metric |
| `writeCSV` | ~15 | No, export |
| `updatePeak` (atomic peak tracking) | ~10 | No, safe counting |
| `modelCPU` | ~10 | No, the one modeled number |

## Run it

```bash
git clone git@github.com:moksheshd/experiments.git
cd experiments/3b-bit-by-byte/season-01-github-august-17/02-proxy-concurrency

go run ./minimal   # the essence: mechanism + one table
go run .           # the full instrumented lab
go run ./tui       # interactive: turn the knob live
```

## What to look for

With the defaults (proxy limit 50, service 20ms/request), the client ramps
10 -> 30 -> 60 -> 120 concurrent. The two stories split exactly at the limit:

- **Below the limit** (10, 30): every request succeeds, latency is flat at one
  service time, service load rises with the client.
- **Past the limit** (60, 120): the service's in-flight count is *pinned* at 50.
  Its load, and so its CPU, cannot rise no matter how hard the client pushes.
  Meanwhile the client's success rate falls to `limit / concurrency`: about 83%
  at 60, about 42% at 120.

In the `tui`, this is something you feel rather than read: hold up the load and
watch the "Service in-flight" bar climb with it, then freeze the instant it hits
the limit while the "Client load" bar keeps growing and the success rate drops.

That is the whole lesson made physical. The service says "I am fine." The client
says "I am failing." Both are true, and the truth lives in the proxy.

## What is measured vs modeled

Everything except one number is measured from the real run: success rate,
rejections, latency percentiles, throughput, and the peak in-flight counts at
both the proxy and the service.

The one modeled number is **service CPU**. Real per-handler CPU is noisy and
machine-dependent, which would make the shipped result unreproducible. So we
model it transparently: the service is assumed to run at `--cpu-at-limit`
percent (default 35) when it is handling `limit` requests at once, and CPU
scales linearly below that. The exact percentage is illustrative. The claim the
model makes honest is structural, not numeric: because the proxy bounds how many
requests reach the service, service CPU has a ceiling that client pressure
cannot push past. That ceiling is the point, and it echoes the essay's "35% CPU
and still drowning."

## Flags (full lab)

| Flag | Default | What it does |
|------|---------|--------------|
| `--limit` | `50` | Proxy concurrency limit. |
| `--steps` | `10,30,60,120` | Client concurrency levels to sweep. |
| `--service-ms` | `20` | Work per request inside the service (real sleep). |
| `--window-ms` | `2000` | How long each step runs. |
| `--reject-backoff-ms` | `-1` | How long a rejected client waits before retrying. `-1` means one service time (the slot won't free sooner), which keeps the closed loop from spinning at CPU speed. |
| `--cpu-at-limit` | `35` | Modeled service CPU% when the service is at `limit` in flight. |
| `--queue` | `false` | Queue instead of reject: failures turn into latency. |
| `--csv` | `""` | Also write the results as CSV. |

## Two variations worth running

```bash
# Failures become latency instead of errors: nobody is rejected, but p95
# latency climbs as the client waits for a slot.
go run . --queue

# Slower service: throughput falls while the CPU ceiling holds, because
# limit = throughput x latency. The service can only clear limit/latency
# requests per second, so a latency bump downstream cuts goodput.
go run . --service-ms=60
```

In the `tui`, the left/right arrows change the proxy limit live, so you can watch
the ceiling move up and down while the load stays fixed.

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
