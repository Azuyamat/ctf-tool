package commands

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/azuyamat/ctf/internal/app"
	"github.com/azuyamat/ctf/internal/credentials"
	"github.com/azuyamat/ctf/internal/event"
	"github.com/azuyamat/ctf/internal/platform"
	"github.com/azuyamat/ctf/internal/platform/ctfd"
	"github.com/azuyamat/ctf/internal/ui"
	"github.com/azuyamat/gear/command"
	"github.com/charmbracelet/x/term"
)

const maxTokenSize = 4096

func (c *cli) cmdLogin() command.Command {
	return command.NewExecutableCommand("login", "Log in to the current event's platform with an access token").
		Group("Events").
		Example(
			"ctf login",
			"# for instances where tokens don't work\nctf login --session -u alice",
			"echo \"$CTFD_TOKEN\" | ctf login --token-stdin",
		).
		Flags(
			command.NewBoolFlag("token-stdin", "", "Read the token, cookie or password from stdin instead of prompting", false),
			command.NewBoolFlag("session", "", "Log in with a username and password, for instances where tokens don't work", false),
			command.NewStringFlag("user", "u", "Username for --session", "").Requires("session"),
			command.NewBoolFlag("cookie", "", "Use a session cookie copied from the browser", false),
		).
		MutuallyExclusive("session", "cookie").
		Handler(c.runLogin)
}

func (c *cli) cmdLogout() command.Command {
	return command.NewExecutableCommand("logout", "Remove the stored login for the current event's platform").
		Group("Events").
		Handler(c.runLogout)
}

func (c *cli) runLogin(ctx *command.Context, args command.ValidatedArgs) error {
	ev, err := c.LinkedEvent()
	if err != nil {
		return err
	}

	session, cookie := command.FlagValue[bool](args, "session"), command.FlagValue[bool](args, "cookie")
	fromStdin := command.FlagValue[bool](args, "token-stdin")

	var auth platform.Auth
	var readErr error
	switch {
	case session:
		auth.Session, readErr = sessionLogin(ctx.Context(), ev, command.FlagValue[string](args, "user"), fromStdin, c.Version)
	case cookie:
		auth.Session, readErr = readSecret(fmt.Sprintf("Session cookie (the value of the session cookie for %s): ", ev.Platform.Host()), "cookie", fromStdin)
		auth.Session = strings.TrimPrefix(auth.Session, "session=")
	default:
		auth.Token, readErr = readSecret(fmt.Sprintf("Access token (create one at %s/settings#tokens): ", ev.Platform.URL), "token", fromStdin)
	}
	if readErr != nil {
		return readErr
	}

	p, err := c.Platform(ev, auth)
	if err != nil {
		return err
	}
	me, err := p.Me(ctx.Context())
	if err != nil {
		return redact(fmt.Errorf("log in to %s: %w", ev.Platform.Host(), err), auth.Secret())
	}

	creds, err := credentials.Load()
	if err != nil {
		return err
	}
	creds.Set(ev.Platform.URL, auth)
	creds.SetUser(ev.Platform.URL, me.Name)
	if err := creds.Save(); err != nil {
		return err
	}

	var details []string
	if me.Team != "" {
		details = append(details, "team "+me.Team)
	}
	if auth.Session != "" {
		details = append(details, "browser session")
	}
	detail := ""
	if len(details) > 0 {
		detail = "  " + ui.Dim(strings.Join(details, " · "))
	}
	ui.Successf("logged in to %s as %s%s", ui.Bold(ev.Platform.Host()), ui.Bold(me.Name), detail)
	return nil
}

func (c *cli) runLogout(ctx *command.Context, args command.ValidatedArgs) error {
	ev, err := c.LinkedEvent()
	if err != nil {
		return err
	}

	creds, err := credentials.Load()
	if err != nil {
		return err
	}
	if !creds.Delete(ev.Platform.URL) {
		ui.Notef("not logged in to %s", ev.Platform.Host())
	} else if err := creds.Save(); err != nil {
		return err
	} else {
		ui.Successf("logged out of %s", ui.Bold(ev.Platform.Host()))
	}

	if os.Getenv(app.TokenEnv) != "" {
		ui.Notef("%s is still set and is used instead of stored tokens", app.TokenEnv)
	}
	return nil
}

func sessionLogin(ctx context.Context, ev *event.Event, user string, fromStdin bool, version string) (string, error) {
	if user == "" {
		if fromStdin || !term.IsTerminal(os.Stdin.Fd()) {
			return "", errors.New("pass the username with --user when stdin is not a terminal")
		}
		ui.Promptf("Username or email for %s: ", ev.Platform.Host())
		line, err := bufio.NewReader(io.LimitReader(os.Stdin, maxTokenSize)).ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return "", fmt.Errorf("read username: %w", err)
		}
		user = strings.TrimSpace(line)
		if user == "" {
			return "", errors.New("no username given")
		}
	}

	password, err := readSecret("Password: ", "password", fromStdin)
	if err != nil {
		return "", err
	}
	session, err := ctfd.Login(ctx, ev.Platform.URL, user, password, version)
	if err != nil {
		return "", redact(fmt.Errorf("log in to %s: %w", ev.Platform.Host(), err), password)
	}
	return session, nil
}

func readSecret(prompt, what string, fromStdin bool) (string, error) {
	var raw []byte
	var err error
	switch {
	case fromStdin:
		raw, err = io.ReadAll(io.LimitReader(os.Stdin, maxTokenSize))
	case term.IsTerminal(os.Stdin.Fd()):
		ui.Promptf("%s", prompt)
		raw, err = term.ReadPassword(os.Stdin.Fd())
		ui.Promptf("\n")
	default:
		return "", fmt.Errorf("stdin is not a terminal, pass the %s with --token-stdin", what)
	}
	if err != nil {
		return "", fmt.Errorf("read %s: %w", what, err)
	}

	secret := strings.TrimRight(string(raw), "\r\n")
	if what != "password" {
		secret = strings.TrimSpace(secret)
	}
	if secret == "" {
		return "", fmt.Errorf("no %s given", what)
	}
	return secret, nil
}

func redact(err error, token string) error {
	msg := err.Error()
	if token == "" || !strings.Contains(msg, token) {
		return err
	}
	return errors.New(strings.ReplaceAll(msg, token, "[redacted]"))
}
