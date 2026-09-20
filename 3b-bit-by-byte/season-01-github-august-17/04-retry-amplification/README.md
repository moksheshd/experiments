# Chapter 4 lab: When Retries Become the Outage

Companion to *3B: Bit By Byte*, Season One, Chapter 4,
"When Retries Become the Outage."

This lab reproduces one mechanism from the August 17, 2026 GitHub incident:
retries do not create capacity. GitHub reported that optimistic retry logic
worsened the incident by overloading internal load balancers. The retries did not
start the fire. They fed it. We build the smallest system where that happens on
purpose, watch the load balloon, then apply the standard discipline (backoff,
jitter, a retry budget, and server backpressure) and watch the same load get
tamed.

It grows directly out of the Chapter 3 lab, reusing its open-loop generator:

- The client is **open-loop**: every 10ms it launches a fixed batch of logical
  requests, so the offered rate stays flat however the service behaves. The client
  never backs off just because the service is struggling.
- Each logical request may make several **attempts**. We run the whole sweep
  twice: once with **naive** retries (retry immediately, no discipline), once with
  **disciplined** retries (backoff + jitter + budget), and count every attempt
  that reaches the service.

## Three ways to run it

| Version | Command | For |
|---------|---------|-----|
| **`minimal/`** | `go run ./minimal` | **Start here.** Just the naive storm: flat offered load, rising past capacity, and attempts ballooning far past offered while goodput sits still. |
| **`.`** (full) | `go run .` | The measured artifact the essay quotes: naive and disciplined runs side by side, plus charts and CSV. |
| **`tui/`** | `go run ./tui` | Interactive. Dial the offered load and toggle naive/disciplined live: watch the attempts bar lift off from the offered bar, then collapse back. |

Read `minimal/main.go` first; it is the same mechanism with the discipline and
everything optional stripped out.

## The mechanism in one idea

A retry is a good idea when the failure is transient: a dropped packet, one node
hiccuping. But when the failure is that the dependency is *out of capacity*, a
retry is not a healthy request getting a second chance. It is more load, aimed at
the exact thing that was already drowning. And it compounds: the struggling state
is precisely what generates the extra requests.

Hold the offered rate flat and push it past what the service can clear. Under
naive retries, every rejection is retried immediately, so the attempts reaching
the service balloon far past the offered rate: amplification climbs to several
times the load. Crucially, the goodput does not climb with it. The service was
never going to clear more than its ceiling. The retries were pure waste, poured
onto a service that was already out of room. Add backoff, jitter, and a retry
budget, and amplification collapses back near 1x for the same goodput.

## The system

A real, in-process, **open-loop** `Client -> Service`:

```text
Client (open loop, flat offered rate)  ->  Service (concurrency limit 50, ~1250 req/s)
```

- **Client:** an open-loop generator. Every 10ms it launches a fixed batch of
  logical requests, so the offered rate is flat regardless of how the service
  behaves. Same generator as Chapter 3.
- **Service:** a hard concurrency limit (default 50). Each admitted request does a
  fixed slice of real work (`time.Sleep`, default 40ms), so its ceiling on useful
  work is `limit / service-time`, about **1250 successful req/s**. When it is full
  it rejects immediately, fast and cheap. That fast "no" is **server
  backpressure**: the service protecting itself.
- **Retries:** a logical request may make several attempts. Every attempt that
  reaches the service (admitted or rejected) is counted. That count over the
  offered rate is the **amplification**.
  - **Naive:** on rejection, retry immediately, up to `max-attempts` (default 5),
    with no backoff, no jitter, no budget.
  - **Disciplined:** exponential backoff (base 50ms, doubled each retry) with full
    jitter, and a retry **budget** (a token bucket refilled at `budget` x offered,
    default 10%). Once the budget is spent, a failed request gives up instead of
    retrying.

## What to look for

With the defaults (service limit 50, 40ms/req, so about 1250 req/s of capacity),
the offered rate sweeps `500 -> 1000 -> 2000 -> 4000` req/s, crossing capacity
between the second and third rows. A representative run:

```text
NAIVE retries  (retry immediately, no backoff, no budget)
Offered   Attempts   Amplify    Goodput   Success
req/s     @service/s x offered  req/s     rate
500       500        1.0x       500       100%
1000      1000       1.0x       1000      100%
2000      5866       2.9x       1034      52%
4000      15685      3.9x       1062      27%

DISCIPLINED retries  (backoff + jitter + 10% budget + backpressure)
500       498        1.0x       498       100%
1000      1010       1.0x       994       100%
2000      2195       1.1x       1065      53%
4000      4400       1.1x       1105      28%
```

Read the two tables together. The offered rate is identical in both. Below
capacity (rows 1 and 2) nothing is rejected, so both runs sit at 1.0x. Past
capacity the naive run's attempts lift off: at 4000 offered it pushed about 15,700
attempts/s at the service, nearly 4x the load, and shed almost three of every four
requests. The disciplined run carried the same 4000 offered at about 1.1x, roughly
4,400 attempts/s. And the goodput is the same in both, near the ~1250 ceiling.
That is the whole point: **retries did not create capacity.** The naive run just
poured several times the load onto a service that was already out of room, which is
exactly how a local slowdown becomes a shared-infrastructure outage.

In the `tui`, this is something you feel: raise the offered load on naive retries
and watch the attempts bar lift off from the offered bar. Press `s` to switch to
disciplined and watch it collapse back down, same load, same goodput.

## What is measured vs modeled

Everything in this lab is **measured** from the real run. There are no modeled
numbers. Attempts, amplification, goodput, and success rate are all counted from
real goroutines hitting a real concurrency-limited service that does real
`time.Sleep` work and rejects instantly when full. (Chapter 3 had one modeled
number, service CPU; this chapter needs none.)

Numbers vary a little run to run, as any real concurrent system does: the
amplification shape (about 1x below capacity, climbing to several-x under naive
overload, pinned near 1.1x under discipline) is stable.

## Flags (full lab)

| Flag | Default | What it does |
|------|---------|--------------|
| `--offered-steps` | `500,1000,2000,4000` | Offered logical-request rates to sweep, in req/s (should cross capacity). |
| `--limit` | `50` | Service concurrency limit: max requests in flight at once. |
| `--service-ms` | `40` | Real work per admitted request, in ms. Capacity is `limit / service-ms`. |
| `--max-attempts` | `5` | Attempts per logical request (1 first try plus retries). |
| `--budget` | `0.10` | Disciplined: retry budget as a fraction of offered traffic. |
| `--backoff-ms` | `50` | Disciplined: base backoff, doubled each retry, with full jitter. |
| `--settle-ms` | `800` | Per step: time to let the pipeline fill before measuring. |
| `--window-ms` | `3000` | Per step: measurement window after settling. |
| `--mode` | `both` | Which run(s): `both` \| `naive` \| `disciplined`. |
| `--csv` | `""` | Also write the results as CSV. |

## Two variations worth running

```bash
# Raise the naive ceiling. More attempts means more amplification is possible, so
# the naive run's attempts climb even higher for the same goodput.
go run . --max-attempts=10

# Tighten or loosen the disciplined budget. At 0.5 the budget stops capping
# amplification hard; at 0.02 the disciplined run is even calmer.
go run . --budget=0.02
```

## Run it

```bash
git clone git@github.com:moksheshd/experiments.git
cd experiments/3b-bit-by-byte/season-01-github-august-17/04-retry-amplification

go run ./minimal   # the naive storm, simplest form
go run .           # naive vs disciplined, side by side (what the essay quotes)
go run ./tui       # interactive: dial offered load, toggle the discipline
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
