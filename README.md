# ctf

Manage CTF challenge folders from the terminal: create them from templates, pull in downloads, track flags and progress per event.

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/Azuyamat/ctf-tool/main/install.sh | sh
```

Installs to `~/.local/bin`. Linux and macOS, amd64 and arm64.

| Variable | Default | |
|---|---|---|
| `CTF_VERSION` | `latest` | Release to install, e.g. `v0.1.0` |
| `CTF_INSTALL_DIR` | `~/.local/bin` | Where to put the binary |

From source (Go 1.27+):

```sh
git clone https://github.com/Azuyamat/ctf-tool && cd ctf-tool
go build -o ~/.local/bin/ctf ./cmd/ctf
```

## Quick start

```sh
ctf event picoctf              # challenges now go under ~/ctf/picoctf
ctf new web/cookies            # create ~/ctf/picoctf/web/cookies
cd ~/ctf/picoctf/web/cookies
ctf grab                       # move the newest download in here
ctf open                       # open the folder in $EDITOR
ctf flag 'picoCTF{...}'        # mark it solved
ctf ls                         # see progress
```

## Commands

| Command | Description |
|---|---|
| `ctf new <name>` | Create a challenge. Slashes make categories: `pwn/baby-rop`. |
| `ctf new <name> -t <template>` | Create from a specific template. |
| `ctf event` | Show the current event and its progress. |
| `ctf event <name>` | Set the current event. |
| `ctf event -c` | Clear the current event. |
| `ctf ls` | List challenges by category with solve status, time and flag. |
| `ctf flag` | Print the current challenge's flag. |
| `ctf flag <flag>` | Set the flag and mark the challenge solved. |
| `ctf grab` | Pick downloads to move into the current challenge. |
| `ctf grab -n <count>` | Preselect the `<count>` newest downloads (default 1). |
| `ctf open` | Open the current challenge in `$EDITOR`. |
| `ctf last` | Print the path of the last created challenge. |
| `ctf templates` | List templates and which ones override others. |
| `ctf config list` | Show all settings. |
| `ctf config get <key>` | Print a setting. |
| `ctf config set <key> <value>` | Change a setting. |
| `ctf version` | Print the version. |

Run `ctf` for the command list and `ctf <command> --help` for its options.

**Current challenge** is the one containing your working directory. `flag` also falls back to the last created challenge; `grab` and `open` don't.

`cd "$(ctf last)"` jumps to the challenge you just made.

## Grab

Downloads are listed newest first; hidden files and unfinished downloads (`.part`, `.crdownload`) are skipped. Files go into the challenge's `dist/` folder if it has one, otherwise the challenge folder.

| Key | Action |
|---|---|
| `↑` `↓` / `k` `j` | Move |
| `space` | Toggle selection |
| `/` | Filter by name (`enter` keeps it, `esc` clears it) |
| `enter` | Grab the selected files, or the highlighted one if none are selected |
| `q` / `esc` | Cancel |

## Templates

A template is a folder copied into every new challenge.

```
~/ctf/
├── templates/          global templates
│   └── default/        used when no -t is given
│       ├── dist/
│       └── solve.py.tmpl
└── picoctf/
    └── templates/      event templates, override global ones with the same name
```

Files ending in `.tmpl` are rendered with Go [`text/template`](https://pkg.go.dev/text/template) and saved without the suffix. Everything else is copied as is.

| Variable | Example |
|---|---|
| `{{.Name}}` | `web/cookies` |
| `{{.Event}}` | `picoctf` |

```python
# solve.py.tmpl
# {{.Event}} / {{.Name}}
from pwn import *
```

## Configuration

| Key | Default | |
|---|---|---|
| `baseDir` | `~/ctf` | Where events and challenges live |
| `downloadDir` | `~/Downloads` | Where `grab` looks for files |

| File | Location |
|---|---|
| Config | `~/.config/ctf/config.json` (macOS: `~/Library/Application Support/ctf/config.json`) |
| State (event, last challenge) | `$XDG_STATE_HOME/ctf/state.json`, default `~/.local/state/ctf/state.json` |
| Challenge data | `.ctf.json` inside each challenge folder |

Output colors turn off when piped or when `NO_COLOR` is set.
