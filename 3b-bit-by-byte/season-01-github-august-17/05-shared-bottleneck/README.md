# Chapter 5 lab: The Road Everyone Shares

Companion to *3B: Bit By Byte*, Season One, Chapter 5,
"The Road Everyone Shares."

This lab reproduces the blast-radius mechanism from the August 17, 2026 GitHub
incident. GitHub reported that the original failure cascaded until four HAProxy
nodes exhausted their flow limits, which degraded the gateway authentication path
and spread authentication latency and failures across nearly every surface:
Issues, Pull Requests, the APIs, Actions, Copilot, SAML and OIDC sign-in, SCIM,
Team Sync. Those features share almost no code. What they share is one road:
before any of them does its own job, it has to authenticate. We build the
smallest system where saturating that one shared road takes down everything
behind it, then contain the damage with a bulkhead.

The point of the chapter, and this lab: independent features can still share a
single road (a gateway, an auth path), and that shared road, not the features,
governs their reliability. Saturating one shared component degrades everything
behind it. A bulkhead (a separate pool per tenant) shrinks the blast radius back
to local.

## Three ways to run it

| Version | Command | For |
|---------|---------|-----|
| **`minimal/`** | `go run ./minimal` | **Start here.** Just the shared gate: only `/profile` floods, yet all three endpoints fall together in one plain table. |
| **`.`** (full) | `go run .` | The measured artifact the essay quotes: shared vs bulkhead, side by side, plus per-endpoint bars and CSV. |
| **`tui/`** | `go run ./tui` | Interactive. Dial `/profile`'s flood and toggle shared/bulkhead live: watch all three success bars fall together, then only `/profile` fall. |

Read `minimal/main.go` first; it is the same mechanism with the bulkhead and
everything optional stripped out.

## The mechanism in one idea

Three endpoints that have nothing to do with each other, all sitting behind one
shared auth gate. Flood just one of them. Because the flood fills the shared
gate, all three fail together, even the two whose own load never moved and whose
own handlers are perfect. They are not three independent systems. They are three
tenants of one road.

Then give each endpoint its own gate. Now the flood can only starve its own lane,
and the other two stay healthy. Same load, same handlers: the only thing that
changed is the blast radius.

## The system

A real, in-process set of three independent endpoints behind a shared auth gate:

```text
/notes    (200 req/s)  \
/tasks    (200 req/s)  ->  AUTH GATE (semaphore, 30 slots, ~1500 auth/s)  ->  handler (5ms)
/profile  (3000 req/s) /
```

- **Endpoints:** `/notes`, `/tasks`, `/profile` are three separate handlers with
  zero shared logic. Each does a trivially fast slice of its own work
  (`time.Sleep`, 5ms). The handler is never the bottleneck.
- **Auth gate:** a counting semaphore of 30 slots. Every request must pass auth
  before it reaches its handler. Each auth check holds a slot for 20ms, so the
  gate's throughput ceiling is `slots / auth-ms`, about 1500 auth/s in total. A
  request that cannot get a slot within 50ms is rejected: an auth failure at the
  door.
- **Load:** open-loop, one flat generator per endpoint (a fixed batch every
  10ms), so offered rates never back off. `/notes` and `/tasks` are light and
  innocent (200 req/s each). `/profile` floods (3000 req/s), which alone is twice
  the gate's whole capacity.
- **Two arrangements:** in **shared** mode all three endpoints use the one gate;
  in **bulkhead** mode each gets its own gate of `30 / 3 = 10` slots.

## What to look for

With the defaults, a representative run:

```text
SHARED pool: all three authenticate through one gate
Endpoint   Offered  Success   Admit    p50      p95
           req/s    rate      req/s    lat      lat
/notes     200      39%       75       71ms     83ms
/tasks     200      44%       84       72ms     94ms
/profile   3000     43%       1238     72ms     86ms

BULKHEAD: each endpoint gets its own gate
/notes     200      100%      199      25ms     28ms
/tasks     200      100%      199      25ms     28ms
/profile   3000     16%       486      71ms     77ms
```

Read the two tables together. In the shared pool, `/notes` and `/tasks` offer a
tiny 200 req/s each and their handlers are perfect, yet they succeed only 39% and
44% of the time, in lockstep with the flooding `/profile` (43%), and their
latency triples (25ms healthy to ~72ms). Nothing about them changed. They simply
share `/profile`'s road, and `/profile` filled it.

Give each its own lane and the picture splits. `/notes` and `/tasks` snap back to
100% success at 25ms, because their gates were never touched. `/profile` still
tanks (16%), because it genuinely offers far more than one lane can clear, but its
failure is now its own. Same flood, same handlers. Only the blast radius changed.

In the `tui`, this is something you feel: raise `/profile`'s load on the shared
gate and watch all three success bars fall together, then press `s` and watch
`/notes` and `/tasks` jump back to full while only `/profile` stays red.

## What is measured vs modeled

Everything in this lab is measured from the real run: the offered rate, the
success rate, the admitted throughput, and the p50/p95 latency per endpoint are
all counted from real goroutines passing through a real semaphore with real auth
waits. Unlike the Chapter 2 and Chapter 3 labs, there is **no modeled number**
here. The whole result is the per-endpoint success-rate contrast between the two
arrangements, and it is measured end to end.

## Flags (full lab)

| Flag | Default | What it does |
|------|---------|--------------|
| `--auth-slots` | `30` | Total auth slots: the shared gate's limit. Bulkhead splits this three ways. |
| `--auth-ms` | `20` | How long each auth check holds a slot (sets gate throughput: `slots / auth-ms`). |
| `--auth-timeout-ms` | `50` | How long a request waits for a slot before it is rejected. |
| `--handler-ms` | `5` | Per-request work inside each endpoint's own handler (kept trivially fast). |
| `--notes-rps` | `200` | Offered load for `/notes` (light and innocent). |
| `--tasks-rps` | `200` | Offered load for `/tasks` (light and innocent). |
| `--profile-rps` | `3000` | Offered load for `/profile` (the flood). |
| `--settle-ms` | `1000` | Time to let the pipeline fill before measuring. |
| `--window-ms` | `3000` | Measurement window after settling. |
| `--mode` | `both` | Scenario: `shared` \| `bulkhead` \| `both`. |
| `--csv` | `""` | Also write the results as CSV. |

## Two variations worth running

```bash
# A bigger road. Give the shared gate more slots and the light endpoints recover
# even under the flood, because the gate now clears more than /profile can offer.
go run . --auth-slots=90

# A harder flood. Push /profile until even its own bulkhead lane cannot dent it,
# while /notes and /tasks stay perfectly healthy in their own lanes.
go run . --profile-rps=6000
```

## Run it

```bash
git clone git@github.com:moksheshd/experiments.git
cd experiments/3b-bit-by-byte/season-01-github-august-17/05-shared-bottleneck

go run ./minimal   # the shared bottleneck, simplest form
go run .           # shared vs bulkhead, side by side (what the essay quotes)
go run ./tui       # interactive: dial the flood, toggle the arrangement
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
