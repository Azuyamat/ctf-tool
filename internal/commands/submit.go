package commands

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/azuyamat/ctf/internal/challenge"
	"github.com/azuyamat/ctf/internal/event"
	"github.com/azuyamat/ctf/internal/platform"
	"github.com/azuyamat/ctf/internal/ui"
	"github.com/azuyamat/gear/command"
	"github.com/charmbracelet/x/term"
)

func (c *cli) cmdSubmit() command.Command {
	return command.NewExecutableCommand("submit", "Submit a flag for the current challenge to the event's platform").
		Group("Challenges").
		Args(
			command.NewStringArg("flag", "Flag to submit"),
		).
		Flags(
			command.NewBoolFlag("force", "f", "Skip the checks before submitting", false),
		).
		Example("ctf submit 'picoCTF{s0me_fl4g}'").
		Handler(c.runSubmit)
}

func (c *cli) runSubmit(ctx *command.Context, args command.ValidatedArgs) error {
	flag := command.ArgValue[string](args, "flag")
	if flag == "" {
		return errors.New("flag cannot be empty")
	}

	ch, err := c.Challenge(true)
	if err != nil {
		return err
	}
	name := c.shortPath(ch.Path)
	if ch.Remote == nil {
		return fmt.Errorf("%s wasn't pulled from a platform, use ctf flag to record the flag locally", name)
	}

	ev, err := c.LinkedEventFor(ch)
	if err != nil {
		return err
	}
	p, err := c.Connect(ev)
	if err != nil {
		return err
	}

	d, err := p.Challenge(ctx.Context(), ch.Remote.ID)
	if err != nil {
		return err
	}
	if !command.FlagValue[bool](args, "force") {
		if err := checkSubmission(ev, ch, d, flag); err != nil {
			return err
		}
	}

	res, err := p.Submit(ctx.Context(), ch.Remote.ID, flag)
	if err != nil {
		return err
	}
	ch.Record(flag, string(res.Status))
	if res.Status == platform.StatusIncorrect {
		ch.Local.Attempts = d.Attempts + 1
	}

	var me string
	if res.Status == platform.StatusCorrect {
		if me, err = c.Username(ctx.Context(), ev, p); err != nil {
			ui.Notef("couldn't look up your username: %v", err)
		}
	}
	outcome := submitOutcome(ch, d, flag, res, me)
	if err := ch.Save(); err != nil {
		return err
	}
	return outcome(name)
}

func submitOutcome(ch *challenge.Challenge, d platform.Detail, flag string, res platform.SubmitResult, me string) func(string) error {
	switch res.Status {
	case platform.StatusCorrect:
		ch.SetFlag(flag)
		ch.MarkSolved(me, time.Now())
		return func(name string) error {
			ui.Successf("correct: %s solved  %s", ui.Bold(name), ui.Green(fmt.Sprintf("+%d", d.Points)))
			return nil
		}
	case platform.StatusAlreadySolved:
		if ch.Flag == "" {
			ch.SetFlag(flag)
		}
		return func(name string) error {
			ui.Successf("%s is already solved", ui.Bold(name))
			return nil
		}
	case platform.StatusIncorrect:
		return func(name string) error {
			if d.MaxAttempts > 0 {
				return fmt.Errorf("incorrect: %s", attemptsLeft(d.MaxAttempts-d.Attempts-1))
			}
			return errors.New("incorrect")
		}
	case platform.StatusRateLimited:
		return func(name string) error {
			if d.MaxAttempts > 0 && d.Attempts >= d.MaxAttempts {
				return fmt.Errorf("no attempts left for %s", name)
			}
			return fmt.Errorf("rate limited: %s", res.Message)
		}
	case platform.StatusPaused:
		return func(name string) error {
			return fmt.Errorf("the CTF is paused: %s", res.Message)
		}
	default:
		return func(name string) error {
			return fmt.Errorf("unexpected response %q: %s", res.Status, res.Message)
		}
	}
}

func checkSubmission(ev *event.Event, ch *challenge.Challenge, d platform.Detail, flag string) error {
	if flag != strings.TrimSpace(flag) {
		return fmt.Errorf("flag has leading or trailing whitespace: %q, use --force to submit it anyway", flag)
	}
	if !ev.MatchesFlagFormat(flag) {
		return fmt.Errorf("flag doesn't match the event's format %s, use --force to submit it anyway", ev.FlagFormat)
	}
	if ch.WasRejected(flag) {
		return fmt.Errorf("%s was already submitted and rejected, use --force to submit it again", flag)
	}
	if d.MaxAttempts == 0 || d.Solved {
		return nil
	}

	left := d.MaxAttempts - d.Attempts
	switch {
	case left <= 0:
		return errors.New("no attempts left for this challenge")
	case left == 1:
		if !term.IsTerminal(os.Stdin.Fd()) {
			return errors.New("this is your last attempt, use --force to submit without confirming")
		}
		if !confirm(fmt.Sprintf("This is your last attempt for %s. Submit %s? [y/N] ", d.Name, flag)) {
			return errors.New("not submitted")
		}
	}
	return nil
}

func attemptsLeft(n int) string {
	if n == 1 {
		return "1 attempt left"
	}
	return fmt.Sprintf("%d attempts left", n)
}
