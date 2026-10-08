<div align="center">

# revtui

**Gerrit code review, right where you work.**

A terminal interface for browsing changes, reading patches, and checking out revisions without leaving your workflow.

[Getting started](#getting-started) · [Features](#features) · [Keyboard shortcuts](#keyboard-shortcuts)

</div>

---

## Features

| | |
| --- | --- |
| **Find your next review** | Browse changes in a compact list or a grid grouped by review status. See the owner, project, branch, and work-in-progress or conflict flags at a glance. |
| **Read the patch** | Open a change to see its commit message and file diffs. Diffs use a unified layout in narrower terminals and a side-by-side layout when there is room. |
| **Stay in your flow** | Refresh changes, open the selected change in your browser, or check out its current revision in your local Git repository. |

revtui currently connects to **Gerrit**. It is built with Go and [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Getting started

### 1. Build

Install [Go 1.26.4](https://go.dev/dl/) or newer, then build from the repository root:

```bash
go build -o revtui ./cmd/
```

Place the resulting `revtui` binary on your `PATH`, or run it as `./revtui` from this directory. revtui has been tested on Linux and Mac.

### 2. Connect to Gerrit

Set your Gerrit host and username. Use the host name only, without `https://`:

```bash
export GERRIT_HOST="gerrit.example.com"
export GERRIT_USER="your-username"
```

Generate an **HTTP password** in your Gerrit account settings, then sign in:

```bash
revtui login "your-http-password"
```

The password is saved to your OS keyring for subsequent runs. Since `login` takes the password as a command-line argument, it may also appear in your shell history; clear that entry if needed.

### 3. Launch

```bash
revtui
```

Run `revtui` inside a local Git checkout if you plan to use the checkout shortcut. It fetches the selected revision from that repository's `origin` remote and checks out `FETCH_HEAD`.

## Keyboard shortcuts

| Key | Action |
| --- | --- |
| `j` / `k` or `↑` / `↓` | Move through changes, files, or diff lines |
| `h` / `l` or `←` / `→` | Move between columns in grid view |
| `m` | Switch between list and grid views |
| `Enter` | Open a change; focus the diff from the file list |
| `Esc` | Return to the file list or change list |
| `Page Up` / `Page Down` | Scroll the open diff |
| `gg` / `G` | Jump to the top or bottom of the open diff |
| `r` | Refresh changes and the open patch |
| `o` | Open the selected change in a browser |
| `c` | Check out the selected revision |
| `q` / `Ctrl+C` | Quit |

## Other commands

| Command | Action |
| --- | --- |
| `revtui changes` | Launch the TUI (same as `revtui`) |
| `revtui me` | Show the signed-in Gerrit account |
| `revtui logout` | Remove credentials for the current host and user from the OS keyring |
| `revtui nuke` | Remove all revtui credentials from the OS keyring |

If your Gerrit server uses a self-signed certificate, `GERRIT_SKIP_VERIFY_TLS=1` disables TLS certificate verification for revtui. Use it only for a server you trust.

---

<div align="center">

For the days when opening another browser tab feels like one tab too many.

</div>
