package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/azuyamat/ctf/internal/fsutil"
	"github.com/azuyamat/ctf/internal/ui"
	"github.com/azuyamat/gear/command"
)

func (c *cli) cmdGrab() command.Command {
	return command.NewExecutableCommand("grab", "Grabs files from the downloads directory").
		Group("Challenges").
		Flags(
			command.NewIntFlag("recent", "n", "Number of most recent downloads to preselect", 1),
		).
		Handler(c.runGrab)
}

func (c *cli) runGrab(ctx *command.Context, args command.ValidatedArgs) error {
	ch, err := c.Challenge(false)
	if err != nil {
		return err
	}

	downloads, err := c.listDownloads()
	if err != nil {
		return err
	}
	if len(downloads) == 0 {
		return errors.New("no files in downloads directory")
	}

	n := command.FlagValue[int](args, "recent")
	if n < 0 {
		return errors.New("recent count cannot be negative")
	}

	ev, _ := c.EventFor(ch)
	dst := ch.FilesDir(c.Config.Paths(ev).Files)
	p := tea.NewProgram(ui.NewSelect("Grab into "+c.shortPath(dst), downloads).SelectFirst(n))
	model, err := p.Run()
	if err != nil {
		return fmt.Errorf("run select UI: %w", err)
	}
	selected := model.(ui.SelectModel).Selected()
	if len(selected) == 0 {
		return errors.New("no files selected")
	}

	for _, choice := range selected {
		d := choice.(downloadChoice)
		if err := fsutil.Move(d.path, filepath.Join(dst, d.name)); err != nil {
			return fmt.Errorf("grab %s: %w", d.name, err)
		}
		ui.Successf("grabbed %s → %s", ui.Bold(d.name), c.shortPath(dst))
	}
	return nil
}

type downloadChoice struct {
	name    string
	path    string
	size    int64
	isDir   bool
	modTime time.Time
}

func (d downloadChoice) Label() string {
	return d.name
}

func (d downloadChoice) Detail() string {
	size := ui.Size(d.size)
	if d.isDir {
		size = "folder"
	}
	return fmt.Sprintf("%9s  %s", size, ui.Ago(d.modTime))
}

var partialDownloadExts = []string{".part", ".crdownload", ".download"}

func (c *cli) listDownloads() ([]ui.Choice, error) {
	dir := c.Config.DownloadDir
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read downloads directory: %w", err)
	}

	var downloads []downloadChoice
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") || slices.Contains(partialDownloadExts, filepath.Ext(name)) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		downloads = append(downloads, downloadChoice{
			name:    name,
			path:    filepath.Join(dir, name),
			size:    info.Size(),
			isDir:   info.IsDir(),
			modTime: info.ModTime(),
		})
	}

	slices.SortFunc(downloads, func(a, b downloadChoice) int {
		return b.modTime.Compare(a.modTime)
	})

	choices := make([]ui.Choice, len(downloads))
	for i, d := range downloads {
		choices[i] = d
	}
	return choices, nil
}
