# 3B: Bit By Byte, Season One labs

Runnable experiments for **Inside GitHub's August 17 Outage**. Each chapter of
the series ships with a small lab in its own folder. Together they grow a single
`Client -> Proxy -> Service` system that we break on purpose to watch real
distributed-systems failure mechanics happen: concurrency limits, retry
amplification, cascades, and the fixes that contain them.

The labs are written in **Go**. They use
[lipgloss](https://github.com/charmbracelet/lipgloss) for terminal output and
otherwise stick to the standard library. All labs live in one Go module (this
folder) and share a `styles` package, so the whole season looks consistent in
the terminal. Each lab still runs with a single `go run .`.

## Labs

| Chapter | Folder | What it shows |
|---------|--------|---------------|
| 1 | [`01-reading-an-outage/`](01-reading-an-outage/) | A thinking tool: read the incident by asking "what ran out?" |
| 2 | [`02-proxy-concurrency/`](02-proxy-concurrency/) | A proxy saturates at its concurrency limit while the service behind it stays idle. |

More labs appear here as later chapters publish.

## Install Go

You need **Go 1.23 or newer**. The authoritative, always-current instructions
are Go's own: <https://go.dev/doc/install>. Downloads: <https://go.dev/dl/>.

Quick paths per platform:

### macOS

```bash
# Option A: Homebrew
brew install go

# Option B: official package
# Download the .pkg from https://go.dev/dl/ and run the installer.
```

### Linux

```bash
# Official tarball (recommended; replace the version with the current one).
# See https://go.dev/doc/install for the exact latest command.
curl -LO https://go.dev/dl/go1.23.0.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.23.0.linux-amd64.tar.gz
# Add Go to PATH (add this line to ~/.profile or ~/.bashrc):
export PATH=$PATH:/usr/local/go/bin
```

Your distribution's package manager also works (for example
`sudo apt install golang-go` or `sudo dnf install golang`), but it can lag
behind the current release.

### Windows

Download the `.msi` installer from <https://go.dev/dl/> and run it. It adds Go
to your `PATH` automatically. Then use PowerShell or a new terminal.

### Verify

```bash
go version
```

You should see `go version go1.23` or newer.

## Run a lab

```bash
git clone git@github.com:moksheshd/experiments.git
cd experiments/3b-bit-by-byte/season-01-github-august-17/01-reading-an-outage
go run .
```

Each lab's own `README.md` explains its flags and what to look for.
