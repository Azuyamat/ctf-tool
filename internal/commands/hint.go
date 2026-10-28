package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/azuyamat/ctf/internal/challenge"
	"github.com/azuyamat/ctf/internal/platform"
	"github.com/azuyamat/ctf/internal/ui"
	"github.com/azuyamat/gear/command"
	"github.com/charmbracelet/x/term"
)

func (c *cli) cmdHint() command.Command {
	return command.NewExecutableCommand("hint", "List the current challenge's hints, or show and unlock one").
		Group("Challenges").
		Args(
			command.NewIntArg("number", "Hint to show or unlock, as listed").AsOptional(),
		).
		Flags(
			command.NewBoolFlag("yes", "y", "Unlock a paid hint without asking", false),
		).
		Handler(c.runHint)
}

func (c *cli) runHint(ctx *command.Context, args command.ValidatedArgs) error {
	ch, err := c.Challenge(true)
	if err != nil {
		return err
	}
	if ch.Remote == nil {
		return fmt.Errorf("%s wasn't pulled from a platform, run ctf pull", c.shortPath(ch.Path))
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
	hints := challenge.HintsFrom(d.Hints)
	if len(hints) == 0 {
		ui.Notef("%s has no hints", c.shortPath(ch.Path))
		return c.saveHints(ch, hints)
	}

	if !args.Has("number") {
		rows := make([][]string, len(hints))
		for i, h := range hints {
			status := ui.Dim("locked")
			if h.Unlocked {
				status = ui.Green("unlocked")
			}
			rows[i] = []string{ui.Bold(fmt.Sprint(i + 1)), h.CostLabel(), h.Label(i + 1), status}
		}
		ui.Println(ui.Table(rows))
		return c.saveHints(ch, hints)
	}

	n := command.ArgValue[int](args, "number")
	if n < 1 || n > len(hints) {
		return fmt.Errorf("no hint %d, %s has %d", n, c.shortPath(ch.Path), len(hints))
	}
	h := hints[n-1]

	if !h.Unlocked {
		if h.Cost > 0 && !command.FlagValue[bool](args, "yes") {
			me, err := p.Me(ctx.Context())
			if err != nil {
				return err
			}
			if !term.IsTerminal(os.Stdin.Fd()) {
				return fmt.Errorf("hint %d costs %d points, pass --yes to unlock it", n, h.Cost)
			}
			scoreLabel := "Your score"
			if me.Team != "" {
				scoreLabel = "Team score"
			}
			if !confirm(fmt.Sprintf("Unlock %q for %d points? %s: %d [y/N] ", h.Label(n), h.Cost, scoreLabel, me.Score)) {
				ui.Notef("hint %d not unlocked", n)
				return c.saveHints(ch, hints)
			}
		}

		unlocked, err := p.UnlockHint(ctx.Context(), h.ID)
		if err != nil {
			return fmt.Errorf("unlock hint %d: %w", n, err)
		}
		hints[n-1] = challenge.HintsFrom([]platform.Hint{unlocked})[0]
		h = hints[n-1]
	}

	if err := c.saveHints(ch, hints); err != nil {
		return err
	}
	ui.Println(ui.Bold(h.Label(n)) + "  " + ui.Dim(h.CostLabel()))
	ui.Println(h.Content)
	return nil
}

func (c *cli) saveHints(ch *challenge.Challenge, hints []challenge.Hint) error {
	ch.Remote.Hints = hints
	if err := ch.Save(); err != nil {
		return err
	}
	_, err := ch.WriteReadme()
	return err
}

func confirm(prompt string) bool {
	ui.Promptf("%s", prompt)
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes"
}
