# matrix-rain

`ccmatrix`: a cmatrix clone in Go (Coding Challenges "Building The Matrix Falling Text Generator").
Raw ANSI escape codes only; the dependencies are `golang.org/x/term` and `golang.org/x/sys`.

## Usage

```
go build ./cmd/ccmatrix
./ccmatrix
```

Full-screen falling text: a white head followed by a short green trail, every column with its own start and speed.

| Key | Action |
| --- | --- |
| `q` | quit |
| `Ctrl-C` | quit (exit code 130) |
| anything else | ignored |

`SIGINT` and `SIGTERM` also quit (exit codes 130 and 143). On every exit path, including a panic, the terminal is
restored before the process exits: termios, cursor visible, colours reset, alternate screen left.

Status: step 1 of the plan. The flags `-u`, `-C`, `-r`, `-c`, `-s` and live resize handling come in later steps.

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
| Pseudo-terminal (Python `pty`, 80x24, Linux): `q`, `SIGINT`, `SIGTERM`, `Ctrl-C` exit and restore | OK |
| xterm, GNOME Terminal, Kitty, Alacritty, iTerm2, Windows Terminal (WSL) | not yet checked by hand |
