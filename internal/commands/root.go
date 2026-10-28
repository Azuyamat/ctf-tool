package commands

import (
	"os"
	"path/filepath"

	"github.com/azuyamat/ctf/internal/app"
	"github.com/azuyamat/ctf/internal/fsutil"
	"github.com/azuyamat/ctf/internal/ui"
	"github.com/azuyamat/gear/command"
)

type cli struct {
	*app.App
}

func Root(a *app.App) *command.RootCommand {
	c := &cli{a}
	root := command.NewRootCommand("ctf", "Manage CTF challenges effortlessly").
		Version(a.Version).
		HelpFunc(ui.Help(a.Version)).
		AddChild(c.cmdEvent()).
		AddChild(c.cmdPull()).
		AddChild(c.cmdLogin()).
		AddChild(c.cmdLogout()).
		AddChild(c.cmdNew()).
		AddChild(c.cmdLs()).
		AddChild(c.cmdInfo()).
		AddChild(c.cmdOpen()).
		AddChild(c.cmdGrab()).
		AddChild(c.cmdHint()).
		AddChild(c.cmdFlag()).
		AddChild(c.cmdSubmit()).
		AddChild(c.cmdLast()).
		AddChild(c.cmdTemplates()).
		AddChild(c.cmdConfig()).
		AddChild(cmdVersion(a.Version))
	return root.AddChild(root.CompletionCommand().Group("Setup"))
}

func (c *cli) shortPath(path string) string {
	if rel, ok := fsutil.Within(c.Config.BaseDir, path); ok {
		return rel
	}
	if home, err := os.UserHomeDir(); err == nil {
		if rel, ok := fsutil.Within(home, path); ok {
			return filepath.Join("~", rel)
		}
	}
	return path
}
