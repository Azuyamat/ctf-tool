package commands

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/azuyamat/ctf/internal/challenge"
	"github.com/azuyamat/ctf/internal/config"
	"github.com/azuyamat/ctf/internal/ui"
	"github.com/azuyamat/gear/command"
)

func cmdGrab() command.Command {
	return command.NewExecutableCommand("grab", "Grabs files from the downloads directory").
		Flags(
			command.NewIntFlag("recent", "n", "Number of most recent downloads to preselect", 1),
		).
		Handler(runGrab)
}

func runGrab(ctx *command.Context, args command.ValidatedArgs) error {
	ch, err := challenge.LoadFromCwd(config.Current.BaseDir)
	if err != nil {
		return fmt.Errorf("load challenge: %w", err)
	}

	downloads, err := listDownloads()
	if err != nil {
		return err
	}
	if len(downloads) == 0 {
		return errors.New("no files in downloads directory")
	}

	n := args.FlagInt("recent")
	if n < 0 {
		return errors.New("recent count cannot be negative")
	}

	p := tea.NewProgram(ui.NewSelect(downloads).SelectFirst(n))
	model, err := p.Run()
	if err != nil {
		return fmt.Errorf("run select UI: %w", err)
	}
	selected := model.(ui.SelectModel).Selected()
	if len(selected) == 0 {
		return errors.New("no files selected")
	}

	dst := grabDir(ch)
	for _, choice := range selected {
		d := choice.(downloadChoice)
		if err := move(d.path, filepath.Join(dst, d.name)); err != nil {
			return fmt.Errorf("grab %s: %w", d.name, err)
		}
		fmt.Printf("grabbed %s into %s\n", d.name, dst)
	}
	return nil
}

func grabDir(ch *challenge.Challenge) string {
	dist := filepath.Join(ch.Path, "dist")
	if info, err := os.Stat(dist); err == nil && info.IsDir() {
		return dist
	}
	return ch.Path
}

func move(src, dst string) error {
	if _, err := os.Lstat(dst); err == nil {
		return fmt.Errorf("%s already exists", dst)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	err := os.Rename(src, dst)
	if !errors.Is(err, syscall.EXDEV) {
		return err
	}

	if err := copyPath(src, dst); err != nil {
		os.RemoveAll(dst)
		return err
	}
	return os.RemoveAll(src)
}

func copyPath(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return os.CopyFS(dst, os.DirFS(src))
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

type downloadChoice struct {
	name    string
	path    string
	modTime time.Time
}

func (d downloadChoice) String() string {
	return d.name
}

var partialDownloadExts = []string{".part", ".crdownload", ".download"}

func listDownloads() ([]ui.Choice, error) {
	dir := config.Current.DownloadDir
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
