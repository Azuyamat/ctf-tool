package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/azuyamat/ctf/internal/challenge"
	"github.com/azuyamat/ctf/internal/ui"
	"github.com/azuyamat/gear/command"
)

const cdEnv = "CTF_CD"

func (c *cli) cmdCd() command.Command {
	return command.NewExecutableCommand("cd", "Go to a challenge by name, or to the event folder").
		Group("Challenges").
		Args(
			command.NewStringArg("name", "Challenge name, folder or category/name").AsOptional().Complete(c.completeChallenges),
		).
		Example(
			"ctf cd cookies",
			"ctf cd web/cookies",
			"ctf cd",
		).
		Handler(c.runCd)
}

func (c *cli) runCd(ctx *command.Context, args command.ValidatedArgs) error {
	dir := c.ChallengeRoot()
	if args.Has("name") {
		ch, err := c.findChallenge(command.ArgValue[string](args, "name"))
		if err != nil {
			return err
		}
		dir = ch.Path
	}
	fmt.Println(dir)
	if os.Getenv(cdEnv) == "" {
		ui.Notef("to cd directly, add to your shell config: eval \"$(ctf init %s)\"", shellName())
	}
	return nil
}

func (c *cli) findChallenge(query string) (*challenge.Challenge, error) {
	root := c.ChallengeRoot()
	challenges, err := challenge.ListChallenges(root)
	if err != nil {
		return nil, err
	}
	entries := listEntries(root, challenges)
	query = strings.Trim(filepath.ToSlash(query), "/")

	var exact, partial []listed
	for _, e := range entries {
		rel := entryPath(e)
		names := []string{rel, e.name}
		if e.ch.Remote != nil {
			names = append(names, e.ch.Remote.Name)
		}
		switch {
		case containsFold(names, query):
			exact = append(exact, e)
		case strings.Contains(strings.ToLower(rel), strings.ToLower(query)):
			partial = append(partial, e)
		}
	}

	matches := exact
	if len(matches) == 0 {
		matches = partial
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("no challenge matches %q in %s", query, c.shortPath(root))
	case 1:
		return matches[0].ch, nil
	}
	paths := make([]string, len(matches))
	for i, e := range matches {
		paths[i] = entryPath(e)
	}
	return nil, fmt.Errorf("%q matches %d challenges: %s", query, len(matches), strings.Join(paths, ", "))
}

func entryPath(e listed) string {
	if e.category == "" {
		return e.name
	}
	return e.category + "/" + e.name
}

func containsFold(list []string, s string) bool {
	for _, item := range list {
		if strings.EqualFold(item, s) {
			return true
		}
	}
	return false
}

func shellName() string {
	switch name := filepath.Base(os.Getenv("SHELL")); name {
	case "bash", "fish":
		return name
	default:
		return "zsh"
	}
}
