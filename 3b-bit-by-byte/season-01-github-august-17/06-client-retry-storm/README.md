# Chapter 6 lab: The Copilot Retry Storm

Companion to *3B: Bit By Byte*, Season One, Chapter 6, "The Copilot Retry Storm."

This lab reproduces the self-inflicted flood from the August 17, 2026 GitHub
incident: a fleet of clients that turns a slow endpoint into roughly ten times
its normal load, all by itself. GitHub reported the Copilot Token Service went
from about 7,000-9,000 requests per second to about 70,000-100,000, and
attributed it to a latent retry bug in VS Code where delayed responses caused
about tenfold request amplification. We build the smallest system where that
happens on purpose, watch the amplification emerge, then show that failover does
nothing and client-side discipline is what flattens it.

## Three ways to run it

| Version | Command | For |
|---------|---------|-----|
| **`minimal/`** | `go run ./minimal` | **Start here.** Baseline vs the storm: one table, ~1x then ~10x. |
| **`.`** (full) | `go run .` | The measured artifact the essay quotes: baseline, storm, failover, and each client-side fix, with a chart and CSV. |
| **`tui/`** | `go run ./tui` | Interactive. Toggle the fault, fail over (and watch it not help), and cycle the client fixes live. |

Read `minimal/main.go` first; it is the same mechanism with failover and the
fixes stripped out.

## The mechanism, in one idea

A slow response with an aggressive timeout does not replace one request with
another. It stacks them. If the endpoint takes 500ms and the client gives up
after 50ms and reissues without canceling the original, one logical request
spawns about ten attempts before the first one finally answers. Across a fleet,
that is a tenfold amplification, and it is arithmetic: the factor is roughly the
response time divided by the timeout.

## The system

```text
Fleet of N clients (steady heartbeat)  ->  one token endpoint (latency we fault)
```

- **Fleet:** N identical clients (default 1000), each issuing one logical request
  per heartbeat (default every 500ms). That steady baseline (default ~2000 req/s)
  is the calm 7-9K world, scaled down.
- **Endpoint:** answers in a fixed time. Healthy is fast (20ms); the fault makes
  it slow (500ms), past the client timeout. The default capacity is effectively
  unlimited, so the injected fault is latency, not capacity, and the payload is
  pure amplification.
- **Client retry policy:** the bug is reissue-immediately-on-timeout with no
  backoff, jitter, or budget. The fixes are a sane timeout, exponential backoff,
  jitter, and a retry budget.

## What to look for

A representative run (defaults):

```text
Baseline (steady, healthy world): 2000 req/s

Scenario                        Endpoint   Amplify   Success
 Healthy baseline                2000       1.0x      100%
 Buggy client + slow endpoint    20002      10.0x     100%
 Same clients, failover          20002      10.0x     100%
 Fix: sane timeout               2000       1.0x      100%
 Fix: backoff + jitter           8046       4.0x      100%
 Fix: retry budget               2000       1.0x      100%
```

Two rows carry the chapter. **Failover leaves the amplification unchanged**
(10.0x -> 10.0x): the clients generate the load, so moving the server just points
the same storm at a fresh one. And the **retry budget** brings it back to 1.0x,
a hard cap that holds no matter how slow the endpoint gets. Note the success rate
is 100% throughout: nothing is failing here. The clients are simply generating
ten times the necessary load off one slow response, and that firehose is what
takes down a real, capacity-limited service.

In the `tui`, press `f` to fault the endpoint and watch the bar jump to ~10x,
`o` to fail over and watch it not move, then `s` to add client discipline and
watch it collapse.

## What is measured vs modeled

Everything is measured from the real run: the attempts reaching the endpoint,
the amplification factor, and the success rate. The one thing that is
illustrative rather than literal is the **scale**: a fleet of a thousand clients
at ~2000 req/s stands in for GitHub's 7-9K/s world. The amplification factor is
the measured payload, and it does not depend on the absolute scale.

## Flags (full lab)

| Flag | Default | What it does |
|------|---------|--------------|
| `--clients` | `1000` | Fleet size. |
| `--interval-ms` | `500` | Per-client heartbeat: one logical request per interval. |
| `--fast-ms` | `20` | Healthy endpoint response time. |
| `--slow-ms` | `500` | Faulted endpoint response time (past the buggy timeout). |
| `--capacity` | `50000` | Endpoint concurrency limit (default effectively unlimited). |
| `--timeout-ms` | `50` | The buggy client's aggressive timeout. |
| `--sane-timeout-ms` | `800` | The disciplined client's timeout. |
| `--backoff-ms` | `50` | Disciplined base backoff between attempts. |
| `--budget` | `0.10` | Retry budget as a fraction of baseline traffic. |
| `--max-attempts` | `40` | Safety cap on attempts per logical request. |
| `--csv` | `""` | Also write the results as CSV. |

## Two variations worth running

```bash
# The amplification tracks the ratio of response time to timeout. Halve the
# timeout and the storm roughly doubles.
go run . --timeout-ms=25

# A tighter budget caps amplification harder, at the cost of more fast failures.
go run . --budget=0.05
```

## Run it

```bash
git clone git@github.com:moksheshd/experiments.git
cd experiments/3b-bit-by-byte/season-01-github-august-17/06-client-retry-storm

go run ./minimal   # baseline vs the storm
go run .           # every scenario, with the chart
go run ./tui       # interactive: fault, fail over, fix
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
