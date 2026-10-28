package commands

import (
	"fmt"
	"strings"

	"github.com/azuyamat/ctf/internal/pull"
	"github.com/azuyamat/ctf/internal/ui"
	"github.com/azuyamat/gear/command"
)

func (c *cli) cmdPull() command.Command {
	return command.NewExecutableCommand("pull", "Create and update challenge folders from the event's platform").
		Group("Events").
		Long("Create a folder for each challenge on the event's platform, and bring the ones already pulled up to date, including which challenges are solved.").
		Example(
			"ctf pull",
			"# see what would change first\nctf pull web pwn --dry-run",
			"ctf pull --status",
		).
		Args(
			command.NewStringArg("categories", "Only pull these categories").AsVariadic().AsOptional().Complete(c.completeCategories),
		).
		Flags(
			command.NewBoolFlag("dry-run", "n", "Show what would change without writing anything", false),
			command.NewBoolFlag("status", "s", "Only refresh which challenges are solved", false),
			command.NewBoolFlag("free-hints", "", "Unlock hints that cost 0 points", false),
			command.NewBoolFlag("no-files", "", "Don't download challenge files", false),
			command.NewStringFlag("template", "t", "Template for new challenges (default: the defaultTemplate setting)", "").Complete(c.completeTemplates),
		).
		MutuallyExclusive("status", "free-hints").
		MutuallyExclusive("status", "no-files").
		MutuallyExclusive("status", "template").
		Handler(c.runPull)
}

func (c *cli) runPull(ctx *command.Context, args command.ValidatedArgs) error {
	ev, err := c.LinkedEvent()
	if err != nil {
		return err
	}
	p, err := c.Connect(ev)
	if err != nil {
		return err
	}
	statusOnly := command.FlagValue[bool](args, "status")
	tmpl, err := c.resolveTemplate(command.FlagValue[string](args, "template"))
	if err != nil {
		return err
	}

	dryRun := command.FlagValue[bool](args, "dry-run")
	if !dryRun && !statusOnly {
		if err := ev.Create(); err != nil {
			return err
		}
	}

	report, err := pull.Run(ctx.Context(), p, pull.Options{
		Root:       ev.Path,
		Event:      ev.Name,
		Template:   tmpl,
		Paths:      c.Config.Paths(ev),
		Categories: command.ArgValue[[]string](args, "categories"),
		DryRun:     dryRun,
		FreeHints:  command.FlagValue[bool](args, "free-hints"),
		NoFiles:    command.FlagValue[bool](args, "no-files"),
		StatusOnly: statusOnly,
	})
	if err != nil {
		return fmt.Errorf("pull from %s: %w", ev.Platform.Host(), err)
	}
	if _, err := c.Username(ctx.Context(), ev, p); err != nil {
		ui.Notef("couldn't look up your username: %v", err)
	}

	if dryRun {
		ui.Notef("dry run, nothing written")
	}
	rows := solveRows(report)
	if !statusOnly {
		rows = append(pullRows(report), rows...)
	}
	if len(rows) > 0 {
		ui.Println(ui.Table(rows))
	}
	if statusOnly {
		ui.Println(solveSummary(report, dryRun))
	} else {
		ui.Println(pullSummary(report, dryRun))
	}

	if report.SolvesErr != nil {
		return fmt.Errorf("refresh solves: %w", report.SolvesErr)
	}
	if failed := report.CountSolves(pull.SolveFailed); failed > 0 {
		return fmt.Errorf("%d solves failed to save", failed)
	}
	if failed := report.Count(pull.Failed); failed > 0 {
		return fmt.Errorf("%d challenges failed to pull", failed)
	}
	if failed, _ := report.CountFiles(pull.FileFailed); failed > 0 {
		return fmt.Errorf("%d files failed to download", failed)
	}
	return nil
}

func pullRows(report *pull.Report) [][]string {
	var rows [][]string
	for _, res := range report.Results {
		switch res.Kind {
		case pull.New:
			rows = append(rows, []string{ui.Green("+"), res.Path, ui.Dim(fmt.Sprintf("%d pts", res.Points))})
		case pull.Linked:
			rows = append(rows, []string{ui.Accent("↳"), res.Path, ui.Dim("linked existing folder")})
		case pull.Changed:
			rows = append(rows, []string{ui.Accent("~"), res.Path, ui.Dim(strings.Join(res.Changes, ", "))})
		case pull.Gone:
			rows = append(rows, []string{ui.Red("!"), res.Path, ui.Dim("no longer on the server")})
		case pull.Failed:
			rows = append(rows, []string{ui.Red("✗"), res.Path, res.Err.Error()})
		}
		rows = append(rows, fileRows(res.Files)...)
	}
	return rows
}

func solveRows(report *pull.Report) [][]string {
	var rows [][]string
	for _, s := range report.Solves {
		switch s.Kind {
		case pull.SolvedRemotely:
			who := "solved by you on the site"
			if !s.Mine {
				who = "solved by " + s.Solver
			}
			rows = append(rows, []string{ui.Green("✓"), s.Path, ui.Dim(who)})
		case pull.LocalOnly:
			rows = append(rows, []string{ui.Red("?"), s.Path, ui.Dim("solved locally but not on the server, kept")})
		case pull.SolveFailed:
			rows = append(rows, []string{ui.Red("✗"), s.Path, s.Err.Error()})
		}
	}
	return rows
}

func solveSummary(report *pull.Report, dryRun bool) string {
	n := report.CountSolves(pull.SolvedRemotely)
	switch {
	case n == 0:
		return "solves are up to date"
	case dryRun:
		return fmt.Sprintf("would mark %d solved", n)
	default:
		return fmt.Sprintf("marked %d solved", n)
	}
}

func fileRows(files []pull.FileResult) [][]string {
	var rows [][]string
	for _, f := range files {
		switch f.Kind {
		case pull.Planned:
			rows = append(rows, []string{ui.Dim("↓"), f.Path, ui.Dim("would download")})
		case pull.Fetched:
			rows = append(rows, []string{ui.Green("↓"), f.Path, ui.Dim(ui.Size(f.Bytes))})
		case pull.Conflict:
			rows = append(rows, []string{ui.Red("!"), f.Path, ui.Dim("changed on the server, your copy was kept")})
		case pull.FileFailed:
			rows = append(rows, []string{ui.Red("✗"), f.Path, f.Err.Error()})
		}
	}
	return rows
}

func pullSummary(report *pull.Report, dryRun bool) string {
	verb := "pulled"
	if dryRun {
		verb = "would pull"
	}
	parts := []string{
		fmt.Sprintf("%d new", report.Count(pull.New)+report.Count(pull.Linked)),
		fmt.Sprintf("%d changed", report.Count(pull.Changed)),
		fmt.Sprintf("%d unchanged", report.Count(pull.Unchanged)),
	}
	extra := ""
	fetched, bytes := report.CountFiles(pull.Fetched)
	conflicts, conflictBytes := report.CountFiles(pull.Conflict)
	planned, _ := report.CountFiles(pull.Planned)
	if files := fetched + conflicts + planned; files > 0 {
		noun := "files"
		if files == 1 {
			noun = "file"
		}
		extra += fmt.Sprintf(" · %d %s", files, noun)
		if total := bytes + conflictBytes; total > 0 {
			extra += " (" + ui.Size(total) + ")"
		}
	}
	if report.Locked > 0 {
		extra += ui.Dim(fmt.Sprintf(" · %d locked", report.Locked))
	}
	if n := report.CountSolves(pull.SolvedRemotely); n > 0 {
		extra += fmt.Sprintf(" · %d newly solved", n)
	}
	return fmt.Sprintf("%s %s%s", verb, strings.Join(parts, ", "), extra)
}
