# Chapter 2 lab: The Proxy Beside Your Application

Companion to *3B: Bit By Byte*, Season One, Chapter 2,
"The Proxy Beside Your Application."

This is the season's first break-it lab. It reproduces one mechanism from the
August 17, 2026 GitHub incident: a proxy with a concurrency limit saturates
while the service behind it stays comfortably idle. The client sees failures.
The service's own load never rises past the limit. Two graphs, opposite stories,
same system.

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

## Run it

```bash
git clone git@github.com:moksheshd/experiments.git
cd experiments/3b-bit-by-byte/season-01-github-august-17/02-proxy-concurrency
go run .
```

You sweep the client through several concurrency levels and, for each, see what
the client experiences and what the service experiences side by side.

## What to look for

With the defaults (proxy limit 50, service 20ms/request), the client ramps
10 -> 30 -> 60 -> 120 concurrent. The two stories split exactly at the limit:

- **Below the limit** (10, 30): every request succeeds, latency is flat at one
  service time, service load rises with the client.
- **Past the limit** (60, 120): the service's in-flight count is *pinned* at 50.
  Its load, and so its CPU, cannot rise no matter how hard the client pushes.
  Meanwhile the client's success rate falls to `limit / concurrency`: about 83%
  at 60, about 42% at 120.

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

## Flags

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

## Dependencies

One: [lipgloss](https://github.com/charmbracelet/lipgloss) for terminal styling,
shared across the season through the `styles` package one level up. `go run .`
fetches what it needs automatically.
