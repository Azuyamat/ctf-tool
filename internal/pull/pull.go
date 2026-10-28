package pull

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"unicode"

	"github.com/azuyamat/ctf/internal/challenge"
	"github.com/azuyamat/ctf/internal/platform"
	"github.com/azuyamat/ctf/internal/template"
)

const parallelFetches = 4

type Kind int

const (
	New Kind = iota
	Linked
	Changed
	Unchanged
	Gone
	Failed
)

type Options struct {
	Root       string
	Event      string
	Template   *template.Template
	Paths      challenge.Paths
	Categories []string
	DryRun     bool
	FreeHints  bool
	NoFiles    bool
	StatusOnly bool
}

type Result struct {
	Kind    Kind
	Path    string
	Points  int
	Changes []string
	Files   []FileResult
	Err     error
}

type Report struct {
	Results   []Result
	Solves    []SolveResult
	SolvesErr error
	Locked    int
}

func (r *Report) CountFiles(kind FileKind) (int, int64) {
	n, total := 0, int64(0)
	for _, res := range r.Results {
		for _, f := range res.Files {
			if f.Kind == kind {
				n++
				total += f.Bytes
			}
		}
	}
	return n, total
}

func (r *Report) Count(kind Kind) int {
	n := 0
	for _, res := range r.Results {
		if res.Kind == kind {
			n++
		}
	}
	return n
}

type puller struct {
	p        platform.Platform
	opts     Options
	byID     map[int]*challenge.Challenge
	byPath   map[string]*challenge.Challenge
	assigned map[string]bool
}

func newPuller(p platform.Platform, opts Options, locals []*challenge.Challenge) *puller {
	pl := &puller{
		p:        p,
		opts:     opts,
		byID:     make(map[int]*challenge.Challenge),
		byPath:   make(map[string]*challenge.Challenge),
		assigned: make(map[string]bool),
	}
	for _, ch := range locals {
		pl.byPath[filepath.ToSlash(ch.Name)] = ch
		if ch.Remote != nil {
			pl.byID[ch.Remote.ID] = ch
		}
	}
	return pl
}

func Run(ctx context.Context, p platform.Platform, opts Options) (*Report, error) {
	locals, err := challenge.ListChallenges(opts.Root)
	if err != nil {
		return nil, err
	}
	pl := newPuller(p, opts, locals)
	report := &Report{}

	if opts.StatusOnly {
		report.Solves, err = pl.solves(ctx, nil)
		return report, err
	}

	summaries, err := p.Challenges(ctx)
	if err != nil {
		return nil, err
	}
	onServer := make(map[int]bool)
	var visible []platform.Summary
	for _, s := range summaries {
		onServer[s.ID] = true
		if s.Locked {
			report.Locked++
			continue
		}
		if pl.wanted(s.Category) {
			visible = append(visible, s)
		}
	}
	slices.SortFunc(visible, func(a, b platform.Summary) int { return a.ID - b.ID })

	details, errs := pl.fetchAll(ctx, visible)
	for i, s := range visible {
		if errs[i] != nil {
			report.Results = append(report.Results, Result{Kind: Failed, Path: s.Category + "/" + s.Name, Err: errs[i]})
			continue
		}
		res, ch := pl.apply(details[i])
		if ch != nil && res.Kind != Failed && !opts.NoFiles {
			res.Files = pl.download(ctx, ch, details[i].Files)
		}
		report.Results = append(report.Results, res)
	}

	for _, ch := range locals {
		if ch.Remote != nil && !onServer[ch.Remote.ID] && pl.wanted(ch.Remote.Category) {
			report.Results = append(report.Results, Result{Kind: Gone, Path: filepath.ToSlash(ch.Name)})
		}
	}
	report.Solves, report.SolvesErr = pl.solves(ctx, onServer)
	return report, nil
}

func (pl *puller) wanted(category string) bool {
	if len(pl.opts.Categories) == 0 {
		return true
	}
	for _, want := range pl.opts.Categories {
		if strings.EqualFold(want, category) || slug(want) == slug(category) {
			return true
		}
	}
	return false
}

func (pl *puller) fetchAll(ctx context.Context, list []platform.Summary) ([]platform.Detail, []error) {
	details := make([]platform.Detail, len(list))
	errs := make([]error, len(list))
	sem := make(chan struct{}, parallelFetches)
	var wg sync.WaitGroup
	for i, s := range list {
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()
			details[i], errs[i] = pl.fetch(ctx, s.ID)
		})
	}
	wg.Wait()
	return details, errs
}

func (pl *puller) fetch(ctx context.Context, id int) (platform.Detail, error) {
	d, err := pl.p.Challenge(ctx, id)
	if err != nil || !pl.opts.FreeHints || pl.opts.DryRun {
		return d, err
	}
	for i, h := range d.Hints {
		if h.Cost != 0 || h.Unlocked {
			continue
		}
		unlocked, err := pl.p.UnlockHint(ctx, h.ID)
		if err != nil {
			return d, fmt.Errorf("unlock free hint %d: %w", i+1, err)
		}
		d.Hints[i] = unlocked
	}
	return d, nil
}

func (pl *puller) apply(d platform.Detail) (Result, *challenge.Challenge) {
	if ch, ok := pl.byID[d.ID]; ok {
		return pl.update(ch, d, Changed), ch
	}

	rel := folderPath(d.Category, d.Name, d.ID)
	if ch, ok := pl.byPath[rel]; ok && ch.Remote == nil && !pl.assigned[rel] {
		pl.assigned[rel] = true
		return pl.update(ch, d, Linked), ch
	}
	if pl.taken(rel) && !pl.adoptable(rel) {
		rel += fmt.Sprintf("-%d", d.ID)
	}
	if pl.adoptable(rel) {
		pl.assigned[rel] = true
		return pl.adopt(rel, d)
	}
	pl.assigned[rel] = true
	return pl.create(rel, d)
}

func (pl *puller) adoptable(rel string) bool {
	if pl.assigned[rel] || pl.byPath[rel] != nil {
		return false
	}
	info, err := os.Stat(filepath.Join(pl.opts.Root, filepath.FromSlash(rel)))
	return err == nil && info.IsDir()
}

func (pl *puller) adopt(rel string, d platform.Detail) (Result, *challenge.Challenge) {
	res := Result{Kind: Linked, Path: rel, Points: d.Points}
	ch, err := challenge.New(pl.opts.Root, rel)
	if err != nil {
		return failed(res, err), nil
	}
	ch.Connection = d.Connection
	ch.Remote = challenge.RemoteFrom(d)
	ch.Local.Attempts = d.Attempts
	pl.byID[d.ID] = ch
	if pl.opts.DryRun {
		return res, ch
	}

	if err := ch.Save(); err != nil {
		return failed(res, err), nil
	}
	if _, err := ch.WriteReadme(pl.opts.Paths.Readme); err != nil {
		return failed(res, err), nil
	}
	return res, ch
}

func (pl *puller) taken(rel string) bool {
	if pl.assigned[rel] {
		return true
	}
	_, err := os.Lstat(filepath.Join(pl.opts.Root, filepath.FromSlash(rel)))
	return !errors.Is(err, fs.ErrNotExist)
}

func (pl *puller) create(rel string, d platform.Detail) (Result, *challenge.Challenge) {
	res := Result{Kind: New, Path: rel, Points: d.Points}
	ch, err := challenge.New(pl.opts.Root, rel)
	if err != nil {
		return failed(res, err), nil
	}
	ch.Connection = d.Connection
	ch.Remote = challenge.RemoteFrom(d)
	ch.Local.Attempts = d.Attempts
	pl.byID[d.ID] = ch
	if pl.opts.DryRun {
		return res, ch
	}

	if err := ch.Create(pl.opts.Template, pl.opts.Event); err != nil {
		return failed(res, err), nil
	}
	if _, err := ch.WriteReadme(pl.opts.Paths.Readme); err != nil {
		return failed(res, err), nil
	}
	return res, ch
}

func (pl *puller) update(ch *challenge.Challenge, d platform.Detail, kind Kind) Result {
	res := Result{Kind: kind, Path: filepath.ToSlash(ch.Name), Points: d.Points}
	next := challenge.RemoteFrom(d)
	connection := ch.Connection
	if d.Connection != "" {
		connection = d.Connection
	}

	if kind == Changed {
		res.Changes = changes(ch.Remote, next, ch.Connection, connection)
		if len(res.Changes) == 0 {
			res.Kind = Unchanged
		}
	}
	if pl.opts.DryRun {
		return res
	}

	ch.Remote = next
	ch.Connection = connection
	ch.Local.Attempts = d.Attempts
	if err := ch.Save(); err != nil {
		return failed(res, err)
	}
	if _, err := ch.WriteReadme(pl.opts.Paths.Readme); err != nil {
		return failed(res, err)
	}
	return res
}

func changes(old, next *challenge.Remote, oldConn, nextConn string) []string {
	var out []string
	if old.Name != next.Name {
		out = append(out, "renamed from "+old.Name)
	}
	if old.Category != next.Category {
		out = append(out, "category")
	}
	if old.Description != next.Description {
		out = append(out, "description")
	}
	if oldConn != nextConn {
		out = append(out, "connection")
	}
	if !slices.Equal(old.Files, next.Files) {
		out = append(out, "files")
	}
	if !slices.Equal(hintKeys(old.Hints), hintKeys(next.Hints)) {
		out = append(out, "hints")
	}
	return out
}

func hintKeys(hints []challenge.Hint) []string {
	keys := make([]string, len(hints))
	for i, h := range hints {
		keys[i] = fmt.Sprintf("%d|%s|%d|%t|%s", h.ID, h.Title, h.Cost, h.Unlocked, h.Content)
	}
	return keys
}

func failed(res Result, err error) Result {
	res.Kind = Failed
	res.Err = err
	return res
}

func folderPath(category, name string, id int) string {
	cat := slug(category)
	if cat == "" {
		cat = "uncategorized"
	}
	n := slug(name)
	if n == "" {
		n = fmt.Sprintf("challenge-%d", id)
	}
	return cat + "/" + n
}

func slug(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(s) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' || r == '_':
			if dash && b.Len() > 0 {
				b.WriteByte('-')
			}
			dash = false
			b.WriteRune(r)
		default:
			dash = true
		}
	}
	return strings.Trim(b.String(), ".")
}
