package pull

import (
	"context"
	"path/filepath"
	"slices"
	"time"

	"github.com/azuyamat/ctf/internal/challenge"
	"github.com/azuyamat/ctf/internal/platform"
)

type SolveKind int

const (
	SolvedRemotely SolveKind = iota
	LocalOnly
	SolveFailed
)

type SolveResult struct {
	Kind   SolveKind
	Path   string
	Mine   bool
	Solver string
	Err    error
}

func (r *Report) CountSolves(kind SolveKind) int {
	n := 0
	for _, s := range r.Solves {
		if s.Kind == kind {
			n++
		}
	}
	return n
}

func (pl *puller) solves(ctx context.Context, onServer map[int]bool) ([]SolveResult, error) {
	list, err := pl.p.Solves(ctx)
	if err != nil {
		return nil, err
	}
	byID := make(map[int]platform.Solve, len(list))
	for _, s := range list {
		byID[s.ChallengeID] = s
	}

	ids := make([]int, 0, len(pl.byID))
	for id := range pl.byID {
		ids = append(ids, id)
	}
	slices.Sort(ids)

	var results []SolveResult
	for _, id := range ids {
		ch := pl.byID[id]
		if !pl.wanted(ch.Category()) || (onServer != nil && !onServer[id]) {
			continue
		}
		if res, ok := pl.applySolve(ch, byID); ok {
			results = append(results, res)
		}
	}
	return results, nil
}

func (pl *puller) applySolve(ch *challenge.Challenge, solves map[int]platform.Solve) (SolveResult, bool) {
	res := SolveResult{Path: filepath.ToSlash(ch.Name)}
	s, solved := solves[ch.Remote.ID]
	if !solved {
		if !ch.IsSolved() {
			return res, false
		}
		res.Kind = LocalOnly
		return res, true
	}

	if ch.IsSolved() && ch.Solver == s.User {
		return res, false
	}

	report := !ch.IsSolved()
	at := s.Time
	if at.IsZero() {
		at = time.Now()
	}
	ch.MarkSolved(s.User, at)
	res.Kind, res.Mine, res.Solver = SolvedRemotely, s.Mine, s.User
	if !pl.opts.DryRun {
		if err := ch.Save(); err != nil {
			res.Kind, res.Err = SolveFailed, err
			return res, true
		}
	}
	return res, report
}
