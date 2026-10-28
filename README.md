# ctf

Manage CTF challenge folders from the terminal: create them from templates, pull in downloads, track flags and progress per event.

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/Azuyamat/ctf-tool/main/install.sh | sh
```

Installs to `~/.local/bin`. Linux and macOS, amd64 and arm64.

Then add shell setup, which gives you `ctf cd` and tab completion:

```sh
eval "$(ctf init zsh)"         # in ~/.zshrc, or `ctf init bash` in ~/.bashrc
ctf init fish | source         # in ~/.config/fish/config.fish
```

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
ctf open                       # open the folder in your editor
ctf flag 'picoCTF{...}'        # mark it solved
ctf ls                         # see progress
```

### With CTFd

```sh
ctf event picoctf --ctfd https://play.picoctf.org
ctf login                      # paste a token from <url>/settings#tokens
ctf pull                       # a folder per challenge, with files and a README
cd ~/ctf/picoctf/web/cookies
ctf info                       # points, connection, description
ctf submit 'picoCTF{...}'
ctf pull --status              # pick up your teammates' solves
```

The token is made in CTFd under **Settings → Access Tokens**. If an instance doesn't accept tokens, `ctf login` says so; use `ctf login --session` there, which logs in with your username and password like the browser does.

## Commands

| Command | Description |
|---|---|
| `ctf new <name>` | Create a challenge. Slashes make categories: `pwn/baby-rop`. |
| `ctf new <name> -t <template>` | Create from a specific template. |
| `ctf new <name> -r <connection>` | Record where the challenge runs, for `{{.Host}}` and `{{.Port}}` in templates. |
| `ctf event` | Show the current event and its progress. |
| `ctf event <name>` | Set the current event. |
| `ctf event <name> --ctfd <url>` | Set the current event and link it to a CTFd instance. `--insecure` allows `http`. |
| `ctf event --unlink` | Remove the current event's CTFd link. Also works with a name. |
| `ctf event --flag-format <regex>` | Set the flag format, checked by `submit` and `flag`. `--no-flag-format` removes it. |
| `ctf event -c` | Clear the current event. |
| `ctf login` | Log in to the current event's CTFd with an access token. `--token-stdin` reads it from stdin. |
| `ctf login --session` | Log in with a username and password instead, for instances where tokens don't work. `-u` sets the username. |
| `ctf login --cookie` | Log in with a `session` cookie copied from the browser. |
| `ctf logout` | Remove the stored login for the current event's CTFd. |
| `ctf pull [category...]` | Create and update challenge folders from the event's CTFd. |
| `ctf pull -n` | Show what `pull` would change without writing anything. |
| `ctf pull --free-hints` | Also unlock hints that cost 0 points. |
| `ctf pull --no-files` | Skip downloading challenge files. |
| `ctf pull --status` | Only refresh which challenges you and your team have solved. |
| `ctf submit <flag>` | Submit a flag to the event's CTFd. Marks the challenge solved only when the server accepts it. |
| `ctf submit <flag> -f` | Skip the checks below. |
| `ctf hint` | List the current challenge's hints. |
| `ctf hint <n>` | Show hint `n`, unlocking it first. Paid hints ask before spending points; `-y` skips the prompt. |
| `ctf ls [category]` | List challenges by category with points, solve status, time and flag. |
| `ctf ls -u` | List only unsolved challenges. |
| `ctf cd <name>` | Go to a challenge. Takes `category/name`, the folder name, the name on CTFd or part of the path. Needs `ctf init`. |
| `ctf cd` | Go to the current event's folder. |
| `ctf info` | Show the current challenge's points, tags, connection, attempts, link and description. |
| `ctf flag` | Print the current challenge's flag. |
| `ctf flag <flag>` | Set the flag and mark the challenge solved. |
| `ctf grab` | Pick downloads to move into the current challenge. |
| `ctf grab -n <count>` | Preselect the `<count>` newest downloads (default 1). |
| `ctf open` | Open the current challenge in the `editor` setting, or `$EDITOR`. |
| `ctf open -w` | Open the challenge's page on the CTFd site in the browser. |
| `ctf last` | Print the path of the last created challenge. |
| `ctf templates` | List templates and which ones override others. |
| `ctf config list` | Show all settings. |
| `ctf config get <key>` | Print a setting. |
| `ctf config set <key> <value>` | Change a setting. |
| `ctf init <shell>` | Print shell setup for `ctf cd` and completion: `bash`, `zsh` or `fish`. |
| `ctf completion <shell>` | Print only the completion script. |
| `ctf version` | Print the version. |
| `ctf completion <shell>` | Print a completion script for `bash`, `zsh` or `fish`. |

Run `ctf` for the command list and `ctf <command> --help` for its options and examples.

### Shell completion

Completion covers commands, flags, config keys, template names and the categories in the current event.

```bash
source <(ctf completion bash)   # add to ~/.bashrc; for zsh use "zsh" and ~/.zshrc
ctf completion fish | source    # add to ~/.config/fish/config.fish
```

**Current challenge** is the one containing your working directory. `flag`, `info`, `submit` and `hint` also fall back to the last created challenge; `grab` and `open` don't.

`cd "$(ctf last)"` jumps to the challenge you just made.

`ctf cd` looks in the current event. An exact match on the path, folder name or CTFd name wins; otherwise any challenge whose path contains the text does, as long as only one does. Without `ctf init` it can't change your shell's directory, so it prints the path instead.

## Grab

Downloads are listed newest first; hidden files and unfinished downloads (`.part`, `.crdownload`) are skipped. Files go into the challenge's `dist/` folder (the `filesDir` setting) if it has one, otherwise the challenge folder.

| Key | Action |
|---|---|
| `↑` `↓` / `k` `j` | Move |
| `space` | Toggle selection |
| `/` | Filter by name (`enter` keeps it, `esc` clears it) |
| `enter` | Grab the selected files, or the highlighted one if none are selected |
| `q` / `esc` | Cancel |

## Pull

`ctf pull` creates a folder for each challenge on the event's CTFd at `<category>/<name>`, using the default template like `ctf new`. When two names would get the same folder, the second gets `-<id>` added. A folder you made yourself with `ctf new` at that path is linked instead.

Each run reports what happened:

| | |
|---|---|
| `+` | New challenge |
| `↳` | Linked to a folder you already had |
| `~` | Name, category, description, connection, files or hints changed |
| `!` | No longer on the server. The folder is kept. |

Points, solve counts, tags and attempts update without being reported, so dynamic scoring doesn't flag every challenge. Locked challenges are counted and created once they unlock.

Challenge files are downloaded into the challenge's `dist/` folder (the `filesDir` setting), or the challenge folder if it has none. A file is only downloaded again when the server's copy changes. If you already have a different version, the new one is saved as `<name>.new` and yours is kept. An interrupted download leaves a `.part` file that the next pull replaces.

### Solves

Pull also asks CTFd which challenges your team has solved. A challenge solved on the site, by you or a teammate, is marked solved with the time it was solved. `ctf ls` shows teammates' solves as `✓ team`, and `ctf info` names who solved it. `ctf pull --status` does only this part, so it's quick to run often.

A local solve is never undone. If you recorded a flag with `ctf flag` and the server doesn't have it as solved, pull marks it with `?` and keeps it.

### What pull writes

Apart from downloaded files, pull only writes `.ctf.json` and `README.md`. `README.md` is regenerated from the server, so keep notes in another file, or set `readmeName` to have pull write somewhere else, like `CHALLENGE.md`. Everything else, like `solve.py`, is never touched after the folder is created.

`README.md` links to the challenge on the CTFd site; `ctf open -w` opens that link.

Pull never spends points. Hint text only comes from CTFd after an unlock, so the README shows hints you or a teammate have unlocked. `ctf hint <n>` unlocks one.

## Submit

`ctf submit` runs inside a pulled challenge and uses the CTFd of the event the challenge belongs to. `ctf flag` stays local and never contacts the server.

Before submitting, it refuses a flag that:

- has leading or trailing whitespace
- doesn't match the event's `--flag-format` (the whole flag must match)
- was already submitted and rejected

On a challenge with limited attempts it asks before using the last one, and refuses when none are left. `-f` skips all of these. Every submission and its result is kept under `submissions` in `.ctf.json`. Incorrect flags exit with an error, so `submit` works in scripts.

| Result | |
|---|---|
| correct | Marks the challenge solved and shows the points. |
| already solved | You or your team solved it already. Saves your flag if none was recorded. |
| incorrect | Shows the attempts left. |
| no attempts left / rate limited | Told apart by the attempt count; wait and retry when rate limited. |
| paused | The CTF is paused. |

## Templates

A template is a folder copied into every new challenge.

```
~/ctf/
├── templates/          global templates
│   └── default/        used when no -t is given (the defaultTemplate setting)
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
| `{{.Category}}` | `web` (empty without a category) |
| `{{.Connection}}` | `nc chall.ctf.io 1337`, as given |
| `{{.Host}}` | `chall.ctf.io` |
| `{{.Port}}` | `1337`, or `443`/`80` for an `https`/`http` URL |
| `{{.URL}}` | `https://web.ctf.io`, only for URL connections |
| `{{.Points}}` | `100`, for challenges from CTFd |

The connection comes from `ctf new <name> --remote <connection>`, which accepts `host:port`, `nc host port` or a URL. Variables it can't fill are empty.

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
| `editor` | empty | Command `ctf open` runs, e.g. `code` or `nvim`. Empty uses `$EDITOR` |
| `defaultTemplate` | `default` | Template for `new` and `pull` when no `-t` is given. Only `default` may be missing |
| `readmeName` | `README.md` | File `pull` writes the challenge description to |
| `filesDir` | `dist` | Folder in a challenge that `grab` and `pull` put files into |

Changing `readmeName` leaves old READMEs in place; the next pull writes the new file next to them. Changing `filesDir` doesn't move files: ones already pulled are tracked by where they were saved and aren't downloaded again.

| File | Location |
|---|---|
| Config | `~/.config/ctf/config.json` (macOS: `~/Library/Application Support/ctf/config.json`) |
| State (event, last challenge) | `$XDG_STATE_HOME/ctf/state.json`, default `~/.local/state/ctf/state.json` |
| Logins (tokens and session cookies) | `$XDG_STATE_HOME/ctf/credentials.json`, readable only by you. `CTF_TOKEN` overrides it. Passwords are never stored. |
| Event data (CTFd link) | `.event.json` inside each event folder |
| Challenge data | `.ctf.json` inside each challenge folder |

Output colors turn off when piped or when `NO_COLOR` is set.

## Development

`testdata/ctfd` runs a seeded CTFd 3.8 instance in Docker for trying CTFd features locally.

```sh
docker compose -f testdata/ctfd/compose.yaml up -d --build
ctf event dev --ctfd http://localhost:8000 --insecure
```

| | |
|---|---|
| URL | `http://localhost:8000` (`CTFD_PORT` to change) |
| Player token | `ctfd_dev_player_token` (team `hashbrowns`) |
| Logins | `player` / `player`, `teammate` / `teammate`, `admin` / `admin` |
| Admin token | `ctfd_dev_admin_token` |

The seed covers the cases pull and submit have to handle: files and a zip, a connection string, an attempt limit, a challenge your teammate already solved, a hidden and a locked challenge, a name that needs slugging, and two names that slug the same.

To try `ctf login --session`, start it with API tokens turned off. Token requests are then redirected to the login page, like on instances that block them:

```sh
CTFD_DISABLE_TOKENS=1 docker compose -f testdata/ctfd/compose.yaml up -d --force-recreate
```

To try an event without teams, set `CTFD_USER_MODE=users`. Everyone then plays alone, and the challenge the teammate solved is solved by `player` instead, as if from the browser:

```sh
CTFD_USER_MODE=users docker compose -f testdata/ctfd/compose.yaml up -d --force-recreate
```

The data lives inside the container. `docker compose -f testdata/ctfd/compose.yaml up -d --force-recreate` resets it, and `down` removes it.
