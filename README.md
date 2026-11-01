# matrix-rain

`ccmatrix` is a [cmatrix](https://github.com/abishekvashok/cmatrix) clone written in Go. It talks to the terminal
directly with raw mode and ANSI escape codes, with no TUI libraries.

```
 7        k      $
 e   h    9   #  +
 <   2    @   =  a
 a   x    f   &  <
 #   %        5  >
```

A white head falls down each column, followed by a fading trail. Every column has its own start, speed and trail length.

## Install

```
go install github.com/andersonmrodrigues/matrix-rain/cmd/ccmatrix@latest
```

Or build from source:

```
git clone https://github.com/andersonmrodrigues/matrix-rain.git
cd matrix-rain
go build ./cmd/ccmatrix
./ccmatrix
```

## Usage

```
ccmatrix [-s] [-u 0-10] [-C colour] [-r] [-c]
```

| Flag | Meaning |
| --- | --- |
| `-s` | screensaver mode: exit on any key, mouse click or scroll |
| `-u N` | update delay, an integer from 0 to 10 (default 4). The pause between frames is `N x 10ms`, so `0` is fastest |
| `-C colour` | trail colour: `green` (default), `red`, `blue`, `white`, `yellow`, `cyan`, `magenta` or `black`. Heads stay white |
| `-r` | rainbow trails, cycling through colours by position. Overrides `-C` |
| `-c` | half-width katakana (U+FF66..U+FF9D) instead of ASCII |

Examples:

```
ccmatrix                     # green rain, default speed
ccmatrix -u 0 -C cyan        # as fast as possible, cyan trails
ccmatrix -r -c               # rainbow katakana
ccmatrix -s -u 2             # screensaver, any key, click or scroll exits
```

Invalid values (`-u 11`, `-C purple`) print an error to stderr and exit with code 2 before the terminal is touched.

Live keys:

| Key | Action |
| --- | --- |
| `0`-`9` | set the speed, same scale as `-u` |
| `q` | quit |
| `Ctrl-C` | quit (exit code 130) |
| any key, mouse click or scroll | quit, with `-s` only (without `-s` other keys are ignored) |

`SIGINT` and `SIGTERM` also quit (exit codes 130 and 143). Resizing the window redraws the rain for the new size.

## How it works

| Package | Role |
| --- | --- |
| `internal/config` | parses and validates flags without touching the terminal |
| `internal/rain` | the falling-text engine: a grid of columns advanced by `Tick`, rebuilt by `Resize` |
| `internal/render` | turns an engine frame into ANSI escape codes |
| `internal/terminal` | raw mode, alternate screen, cursor, mouse reporting, input and size |
| `internal/app` | the event loop that wires the others together |

- **Pure, seeded engine.** The engine has no terminal knowledge and draws its randomness from an injected seeded RNG,
  so its behaviour is deterministic and table-testable.
- **Renderer writes to an `io.Writer`.** Each row is placed with an absolute cursor move and holds exactly the grid
  width, so nothing is drawn outside the window.
- **Restore guarantees.** `terminal.Session` puts stdin in raw mode and switches to the alternate screen with a hidden
  cursor. `Restore` is idempotent and runs on every exit path: `q`, `Ctrl-C`, `SIGINT`, `SIGTERM`, errors and panics. It
  resets colours, shows the cursor, leaves the alternate screen, turns mouse reporting off and restores the original
  termios.
- **Signals.** `SIGINT`, `SIGTERM` and `SIGWINCH` are handled in the main loop. They are registered before the terminal
  is touched, so an early signal cannot leave it in raw mode.
- **Resize.** On `SIGWINCH` the size is read again, the engine rebuilds its columns, and the screen is cleared. Zero
  and 1x1 sizes are handled without drawing out of bounds.
- **Mouse.** Mouse reporting (`ESC[?1000h`, `ESC[?1006h`) is enabled only with `-s`, and its disable sequence is
  always written on cleanup.

## Testing

```
go test ./...
go vet ./...
gofmt -l .        # prints nothing
```

Engine, renderer, flag and key handling are covered by table-driven tests. On Linux, pty-based tests run the real
program on a pseudo-terminal and check that termios and the screen are restored after `q`, `Ctrl-C`, `SIGINT`,
`SIGTERM` and a panic, that mouse reporting is on only with `-s` and always turned off, and that the process survives
window size changes.

## Limitations

- Linux and macOS only. Windows is not supported.
- A `black` trail is invisible on a black or dark background; only the white heads show.
- Katakana alignment depends on the font. If it draws these glyphs wider than one cell, columns can look uneven.
- Mouse support varies by terminal. In `-s`, terminals without mouse reporting still exit on any key. Only clicks and
  scroll are reported (`ESC[?1000h`); plain pointer movement does not exit.

### Mouse compatibility

`-s` uses standard X10/SGR mouse reporting, which these terminals implement. They have not been verified by hand.

| Terminal | Mouse reporting |
| --- | --- |
| xterm | standard X10/SGR |
| GNOME Terminal / VTE | standard X10/SGR |
| Kitty | standard X10/SGR |
| Alacritty | standard X10/SGR |
| iTerm2 | standard X10/SGR |
| macOS Terminal.app | standard X10/SGR |
| tmux | standard X10/SGR (when its `mouse` option is on, tmux consumes events itself) |
