package pull

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/azuyamat/ctf/internal/challenge"
	"github.com/azuyamat/ctf/internal/fsutil"
	"github.com/azuyamat/ctf/internal/platform"
)

const partSuffix = ".part"

type FileKind int

const (
	Planned FileKind = iota
	Fetched
	Conflict
	FileFailed
)

type FileResult struct {
	Kind  FileKind
	Path  string
	Bytes int64
	Err   error
}

func (pl *puller) download(ctx context.Context, ch *challenge.Challenge, files []platform.File) []FileResult {
	var results []FileResult
	changed := false
	for _, f := range files {
		res, saved := pl.downloadOne(ctx, ch, f)
		if res != nil {
			results = append(results, *res)
		}
		if saved != "" {
			if ch.Downloads == nil {
				ch.Downloads = make(map[string]string)
			}
			ch.Downloads[f.ID] = saved
			changed = true
		}
	}

	if changed {
		if err := ch.Save(); err != nil {
			results = append(results, FileResult{Kind: FileFailed, Path: pl.rel(ch.Path), Err: err})
		}
	}
	return results
}

func (pl *puller) downloadOne(ctx context.Context, ch *challenge.Challenge, f platform.File) (*FileResult, string) {
	if !fsutil.ValidName(f.Name) {
		return &FileResult{Kind: FileFailed, Path: pl.rel(ch.Path), Err: fmt.Errorf("unsafe file name %q", f.Name)}, ""
	}
	if saved, ok := ch.Downloads[f.ID]; ok && exists(filepath.Join(ch.Path, saved)) {
		return nil, ""
	}

	dir := pl.filesDir(ch)
	if dir == ch.Path && ownedByPull(f.Name) {
		dir = filepath.Join(ch.Path, challenge.FilesDirName)
	}
	target := filepath.Join(dir, f.Name)
	if pl.opts.DryRun {
		return &FileResult{Kind: Planned, Path: pl.rel(target)}, ""
	}

	part := target + partSuffix
	n, err := pl.fetchFile(ctx, f, part)
	if err != nil {
		os.Remove(part)
		return &FileResult{Kind: FileFailed, Path: pl.rel(target), Err: err}, ""
	}

	kind := Fetched
	for i := 1; exists(target); i++ {
		same, err := sameContent(part, target)
		if err != nil {
			os.Remove(part)
			return &FileResult{Kind: FileFailed, Path: pl.rel(target), Err: err}, ""
		}
		if same {
			os.Remove(part)
			return nil, pl.relTo(ch, target)
		}
		kind = Conflict
		target = filepath.Join(dir, f.Name+".new")
		if i > 1 {
			target += fmt.Sprint(i)
		}
	}

	if err := os.Rename(part, target); err != nil {
		os.Remove(part)
		return &FileResult{Kind: FileFailed, Path: pl.rel(target), Err: err}, ""
	}
	return &FileResult{Kind: kind, Path: pl.rel(target), Bytes: n}, pl.relTo(ch, target)
}

func (pl *puller) filesDir(ch *challenge.Challenge) string {
	if exists(ch.Path) || pl.opts.Template == nil {
		return ch.FilesDir()
	}
	if info, err := os.Stat(filepath.Join(pl.opts.Template.Path, challenge.FilesDirName)); err == nil && info.IsDir() {
		return filepath.Join(ch.Path, challenge.FilesDirName)
	}
	return ch.Path
}

func (pl *puller) relTo(ch *challenge.Challenge, path string) string {
	rel, err := filepath.Rel(ch.Path, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return !errors.Is(err, fs.ErrNotExist)
}

func (pl *puller) fetchFile(ctx context.Context, f platform.File, path string) (int64, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return 0, err
	}
	out, err := os.Create(path)
	if err != nil {
		return 0, err
	}
	counter := &countingWriter{w: out}
	err = pl.p.Download(ctx, f, counter)
	return counter.n, errors.Join(err, out.Close())
}

func (pl *puller) rel(path string) string {
	if rel, ok := fsutil.Within(pl.opts.Root, path); ok {
		return filepath.ToSlash(rel)
	}
	return path
}

func ownedByPull(name string) bool {
	return name == challenge.ReadmeName || name == challenge.FileName
}

func sameContent(a, b string) (bool, error) {
	ha, sa, err := hashFile(a)
	if err != nil {
		return false, err
	}
	hb, sb, err := hashFile(b)
	if err != nil {
		return false, err
	}
	return sa == sb && bytes.Equal(ha, hb), nil
}

func hashFile(path string) ([]byte, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return nil, 0, err
	}
	return h.Sum(nil), n, nil
}

type countingWriter struct {
	w io.Writer
	n int64
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += int64(n)
	return n, err
}
