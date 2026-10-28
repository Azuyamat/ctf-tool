package platform

import (
	"context"
	"errors"
	"io"
	"time"
)

type Platform interface {
	Me(ctx context.Context) (User, error)
	Challenges(ctx context.Context) ([]Summary, error)
	Challenge(ctx context.Context, id int) (Detail, error)
	Submit(ctx context.Context, id int, flag string) (SubmitResult, error)
	Download(ctx context.Context, f File, w io.Writer) error
	UnlockHint(ctx context.Context, id int) (Hint, error)
	Solves(ctx context.Context) ([]Solve, error)
}

type Auth struct {
	Token   string
	Session string
}

func (a Auth) Secret() string {
	if a.Token != "" {
		return a.Token
	}
	return a.Session
}

type Solve struct {
	ChallengeID int
	User        string
	Mine        bool
	Time        time.Time
}

type User struct {
	ID    int
	Name  string
	Team  string
	Score int
}

type Summary struct {
	ID       int
	Name     string
	Category string
	Points   int
	Solves   int
	Solved   bool
	Locked   bool
}

type Detail struct {
	Summary
	URL         string
	Description string
	Connection  string
	Tags        []string
	Files       []File
	Hints       []Hint
	Attempts    int
	MaxAttempts int
}

type Hint struct {
	ID       int
	Title    string
	Cost     int
	Content  string
	Unlocked bool
}

type File struct {
	ID   string
	Name string
	URL  string
}

type Status string

const (
	StatusCorrect       Status = "correct"
	StatusIncorrect     Status = "incorrect"
	StatusAlreadySolved Status = "already_solved"
	StatusRateLimited   Status = "ratelimited"
	StatusPaused        Status = "paused"
)

type SubmitResult struct {
	Status  Status
	Message string
}

var (
	ErrUnauthorized   = errors.New("unauthorized")
	ErrTokensDisabled = errors.New("API tokens don't work on this instance, use ctf login --session")
	ErrSessionExpired = errors.New("session expired, run ctf login --session")
	ErrNotFound       = errors.New("not found")
	ErrRateLimited    = errors.New("rate limited")
	ErrNotStarted     = errors.New("ctf is not running")
)
