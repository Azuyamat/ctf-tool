package challenge

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/azuyamat/ctf/internal/connection"
	"github.com/azuyamat/ctf/internal/fsutil"
	"github.com/azuyamat/ctf/internal/layout"
	"github.com/azuyamat/ctf/internal/platform"
	"github.com/azuyamat/ctf/internal/template"
)

const FileName = ".ctf.json"

type Challenge struct {
	Name        string            `json:"name"`
	Path        string            `json:"-"`
	Flag        string            `json:"flag,omitempty"`
	Created     time.Time         `json:"created"`
	Solved      time.Time         `json:"solved,omitzero"`
	SolvedBy    string            `json:"solvedBy,omitempty"`
	Solver      string            `json:"solver,omitempty"`
	Connection  string            `json:"connection,omitempty"`
	Remote      *Remote           `json:"remote,omitempty"`
	Downloads   map[string]string `json:"downloads,omitempty"`
	Submissions []Submission      `json:"submissions,omitempty"`
}

type Submission struct {
	Time   time.Time `json:"time"`
	Flag   string    `json:"flag"`
	Status string    `json:"status"`
}

func (c *Challenge) Record(flag, status string) {
	c.Submissions = append(c.Submissions, Submission{Time: time.Now(), Flag: flag, Status: status})
}

func (c *Challenge) WasRejected(flag string) bool {
	for _, s := range c.Submissions {
		if s.Flag == flag && s.Status == string(platform.StatusIncorrect) {
			return true
		}
	}
	return false
}

const (
	SolvedByMe    = "me"
	SolvedByTeam  = "team"
	SolvedByLocal = "local"
)

func (c *Challenge) MarkSolved(by, solver string, at time.Time) {
	if c.Solved.IsZero() {
		c.Solved = at
	}
	c.SolvedBy = by
	c.Solver = solver
}

func (c *Challenge) SolvedOnServer() bool {
	return c.SolvedBy == SolvedByMe || c.SolvedBy == SolvedByTeam
}

const DistDir = "dist"

func (c *Challenge) FilesDir() string {
	dist := filepath.Join(c.Path, DistDir)
	if info, err := os.Stat(dist); err == nil && info.IsDir() {
		return dist
	}
	return c.Path
}

type Remote struct {
	ID          int      `json:"id"`
	Name        string   `json:"name,omitempty"`
	URL         string   `json:"url,omitempty"`
	Category    string   `json:"category,omitempty"`
	Description string   `json:"description,omitempty"`
	Points      int      `json:"points,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Files       []string `json:"files,omitempty"`
	Hints       []Hint   `json:"hints,omitempty"`
	Attempts    int      `json:"attempts,omitempty"`
	MaxAttempts int      `json:"maxAttempts,omitempty"`
}

type Hint struct {
	ID       int    `json:"id"`
	Title    string `json:"title,omitempty"`
	Cost     int    `json:"cost"`
	Unlocked bool   `json:"unlocked,omitempty"`
	Content  string `json:"content,omitempty"`
}

func RemoteFrom(d platform.Detail) *Remote {
	r := &Remote{
		ID:          d.ID,
		Name:        d.Name,
		URL:         d.URL,
		Category:    d.Category,
		Description: d.Description,
		Points:      d.Points,
		Tags:        d.Tags,
		Hints:       HintsFrom(d.Hints),
		Attempts:    d.Attempts,
		MaxAttempts: d.MaxAttempts,
	}
	for _, f := range d.Files {
		r.Files = append(r.Files, f.Name)
	}
	return r
}

func HintsFrom(hints []platform.Hint) []Hint {
	var out []Hint
	for _, h := range hints {
		out = append(out, Hint{
			ID:       h.ID,
			Title:    h.Title,
			Cost:     h.Cost,
			Unlocked: h.Unlocked,
			Content:  h.Content,
		})
	}
	return out
}

func New(root, name string) (*Challenge, error) {
	root = filepath.Clean(root)
	rel, ok := fsutil.Within(root, filepath.Join(root, name))
	if !ok {
		return nil, fmt.Errorf("invalid challenge name: %s", name)
	}
	if slices.ContainsFunc(strings.Split(rel, string(filepath.Separator)), layout.IsReserved) {
		return nil, fmt.Errorf("challenge name is reserved: %s", name)
	}

	return &Challenge{
		Name:    rel,
		Path:    filepath.Join(root, rel),
		Created: time.Now(),
	}, nil
}

// Create makes the challenge folder, copying tmpl into it when non-nil.
func (c *Challenge) Create(tmpl *template.Template, event string) error {
	if err := os.MkdirAll(filepath.Dir(c.Path), 0755); err != nil {
		return err
	}

	if err := os.Mkdir(c.Path, 0755); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return fmt.Errorf("challenge already exists: %s", c.Name)
		}
		return err
	}

	if tmpl != nil {
		if err := tmpl.CopyTo(c.Path, c.templateData(event)); err != nil {
			os.RemoveAll(c.Path)
			return fmt.Errorf("copy template %s: %w", tmpl.Name, err)
		}
	}

	// Saved after the template copy so it wins over any marker the template contains.
	if err := c.Save(); err != nil {
		os.RemoveAll(c.Path)
		return err
	}
	return nil
}

// SetFlag records the flag. The solve time is kept from the first time a flag was set.
func (c *Challenge) SetFlag(flag string) {
	c.Flag = flag
	if c.Solved.IsZero() {
		c.Solved = time.Now()
	}
}

func (c *Challenge) IsSolved() bool {
	return !c.Solved.IsZero()
}

func (c *Challenge) Category() string {
	if c.Remote != nil && c.Remote.Category != "" {
		return c.Remote.Category
	}
	category, _, ok := strings.Cut(filepath.ToSlash(c.Name), "/")
	if !ok {
		return ""
	}
	return category
}

func (c *Challenge) templateData(event string) template.Data {
	conn := connection.Parse(c.Connection)
	data := template.Data{
		Name:       c.Name,
		Event:      event,
		Category:   c.Category(),
		Connection: c.Connection,
		Host:       conn.Host,
		Port:       conn.Port,
		URL:        conn.URL,
	}
	if c.Remote != nil {
		data.Points = c.Remote.Points
	}
	return data
}

func (c *Challenge) configFilePath() string {
	return filepath.Join(c.Path, FileName)
}

func (c *Challenge) Load() error {
	if err := fsutil.ReadJSON(c.configFilePath(), c); err != nil {
		return fmt.Errorf("read challenge: %w", err)
	}
	return nil
}

func (c *Challenge) Save() error {
	if err := fsutil.WriteJSON(c.configFilePath(), c, 0644); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}
	return nil
}

var ErrNotFound = errors.New("not inside a challenge")

// LoadFromPath finds the challenge containing path by walking up to base.
// It is bounded by base rather than Root so it works across events.
func LoadFromPath(base, path string) (*Challenge, error) {
	base = filepath.Clean(base)
	path = filepath.Clean(path)
	if _, ok := fsutil.Within(base, path); !ok {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, path)
	}

	for p := path; p != base; p = filepath.Dir(p) {
		challenge := &Challenge{Path: p}
		if err := challenge.Load(); err == nil {
			return challenge, nil
		} else if !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("load challenge config: %w", err)
		}
	}

	return nil, fmt.Errorf("%w: %s", ErrNotFound, path)
}

// LoadFromCwd loads the challenge containing the working directory.
func LoadFromCwd(base string) (*Challenge, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get current dir: %w", err)
	}
	return LoadFromPath(base, cwd)
}

// LoadCurrent loads the challenge containing the working directory, falling
// back to the last created challenge.
func LoadCurrent(base, last string) (*Challenge, error) {
	ch, err := LoadFromCwd(base)
	if !errors.Is(err, ErrNotFound) || last == "" {
		return ch, err
	}
	return LoadFromPath(base, last)
}

// ListChallenges returns the challenges under root. A missing root has none.
func ListChallenges(root string) ([]*Challenge, error) {
	var challenges []*Challenge
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if path == root && errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if path != root && layout.IsReserved(d.Name()) {
			return filepath.SkipDir
		}
		challenge := &Challenge{Path: path}
		if err := challenge.Load(); err == nil {
			challenges = append(challenges, challenge)
			return filepath.SkipDir
		} else if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("load challenge config: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk challenges: %w", err)
	}

	return challenges, nil
}
