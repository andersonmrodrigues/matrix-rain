# matrix-rain

`ccmatrix`: a cmatrix clone in Go (Coding Challenges "Building The Matrix Falling Text Generator").
Raw ANSI escape codes only; the dependencies are `golang.org/x/term` and `golang.org/x/sys`.

## Install

```
go install github.com/andersonmrodrigues/matrix-rain/cmd/ccmatrix@latest
```

## Usage

```
go build ./cmd/ccmatrix
./ccmatrix [-u 0-10] [-C colour]
```

| Flag | Meaning |
| --- | --- |
| `-u N` | update speed, an integer from 0 to 10 (default 4). The delay between frames is `N x 10ms`, so `0` is fastest (no deliberate sleep) and `10` is 100ms |
| `-C colour` | trail colour: `green` (default), `red`, `blue`, `white`, `yellow`, `cyan`, `magenta` or `black`. The head always stays white |

Invalid values (`-u 11`, `-u fast`, `-C purple`) print an error to stderr, exit with code 2 and write nothing to the
terminal: flags are validated before the terminal is set up. `-C purple` lists the supported colours.

A `black` trail is invisible on a black or dark background; only the white heads show.

Full-screen falling text: a white head followed by a short green trail, every column with its own start and speed.

| Key | Action |
| --- | --- |
| `0`-`9` | set the speed live, same mapping as `-u` (`0` fastest) |
| `q` | quit |
| `Ctrl-C` | quit like `SIGINT` (exit code 130); raw mode disables the tty's own signal, so the `0x03` byte is handled by the app |
| anything else | ignored |

`SIGINT` and `SIGTERM` also quit (exit codes 130 and 143). On every exit path, including a panic, the terminal is
restored before the process exits: termios, cursor visible, colours reset, alternate screen left.

Status: step 2 of the plan. The flags `-r`, `-c`, `-s` and live resize handling come in later steps.

## Development

```
go test ./...
go vet ./...
gofmt -l .        # must print nothing
go build ./cmd/ccmatrix
```

## Tested terminals

| Terminal | Result |
| --- | --- |
| Linux pseudo-terminal (80x24), automated in `internal/app/run_linux_test.go`: `q`, `Ctrl-C` byte, `SIGINT`, `SIGTERM` and a forced panic restore termios (incl. `ECHO`, `ICANON`) and emit the cleanup sequence; `a`/`x` are ignored; non-tty stdout fails with an error | OK (`go test`) |
| xterm, GNOME Terminal, Kitty, Alacritty, iTerm2, Windows Terminal (WSL) | not checked by hand; the pty test does not emulate a terminal's rendering |
