package ctfd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/azuyamat/ctf/internal/platform"
)

const lockedType = "hidden"

type challengeJSON struct {
	ID         int    `json:"id"`
	Type       string `json:"type"`
	Name       string `json:"name"`
	Category   string `json:"category"`
	Value      int    `json:"value"`
	Solves     *int   `json:"solves"`
	SolvedByMe bool   `json:"solved_by_me"`
}

type challengeDetailJSON struct {
	challengeJSON
	Description    string     `json:"description"`
	ConnectionInfo *string    `json:"connection_info"`
	Tags           []string   `json:"tags"`
	Files          []string   `json:"files"`
	Hints          []hintJSON `json:"hints"`
	Attempts       int        `json:"attempts"`
	MaxAttempts    int        `json:"max_attempts"`
}

type hintJSON struct {
	ID      int     `json:"id"`
	Title   *string `json:"title"`
	Cost    int     `json:"cost"`
	Content *string `json:"content"`
}

func (h hintJSON) toHint() platform.Hint {
	hint := platform.Hint{ID: h.ID, Cost: h.Cost}
	if h.Title != nil {
		hint.Title = *h.Title
	}
	if h.Content != nil {
		hint.Content = *h.Content
		hint.Unlocked = true
	}
	return hint
}

type attemptJSON struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func (ch challengeJSON) toSummary() platform.Summary {
	locked := ch.Type == lockedType
	s := platform.Summary{
		ID:     ch.ID,
		Solved: ch.SolvedByMe,
		Locked: locked,
	}
	if locked {
		return s
	}
	s.Name = ch.Name
	s.Category = ch.Category
	s.Points = ch.Value
	if ch.Solves != nil {
		s.Solves = *ch.Solves
	}
	return s
}

func (ch challengeDetailJSON) toDetail(base *url.URL) platform.Detail {
	d := platform.Detail{
		Summary:     ch.toSummary(),
		URL:         challengeURL(base, ch.Name, ch.ID),
		Description: ch.Description,
		Tags:        ch.Tags,
		Attempts:    ch.Attempts,
		MaxAttempts: ch.MaxAttempts,
	}
	if ch.ConnectionInfo != nil {
		d.Connection = *ch.ConnectionInfo
	}
	for _, raw := range ch.Files {
		d.Files = append(d.Files, platform.File{ID: fileID(raw), Name: fileName(raw), URL: raw})
	}
	for _, h := range ch.Hints {
		d.Hints = append(d.Hints, h.toHint())
	}
	return d
}

func challengeURL(base *url.URL, name string, id int) string {
	u := base.JoinPath("challenges")
	u.Fragment = name + "-" + strconv.Itoa(id)
	return u.String()
}

func fileID(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	return u.Path
}

func fileName(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return path.Base(raw)
	}
	return path.Base(u.Path)
}

func (c *Client) Challenges(ctx context.Context) ([]platform.Summary, error) {
	var list []challengeJSON
	if err := c.do(ctx, http.MethodGet, "challenges", nil, &list); err != nil {
		return nil, err
	}

	summaries := make([]platform.Summary, len(list))
	for i, ch := range list {
		summaries[i] = ch.toSummary()
	}
	return summaries, nil
}

func (c *Client) Challenge(ctx context.Context, id int) (platform.Detail, error) {
	var ch challengeDetailJSON
	if err := c.do(ctx, http.MethodGet, "challenges/"+strconv.Itoa(id), nil, &ch); err != nil {
		return platform.Detail{}, err
	}
	return ch.toDetail(c.base), nil
}

func (c *Client) Submit(ctx context.Context, id int, flag string) (platform.SubmitResult, error) {
	body := map[string]any{"challenge_id": id, "submission": flag}
	var a attemptJSON
	if err := c.do(ctx, http.MethodPost, "challenges/attempt", body, &a); err != nil {
		return platform.SubmitResult{}, err
	}
	return platform.SubmitResult{Status: platform.Status(a.Status), Message: a.Message}, nil
}

func (c *Client) UnlockHint(ctx context.Context, id int) (platform.Hint, error) {
	body := map[string]any{"target": id, "type": "hints"}
	if err := c.do(ctx, http.MethodPost, "unlocks", body, nil); err != nil && !strings.Contains(err.Error(), "already unlocked") {
		return platform.Hint{}, err
	}

	var h hintJSON
	if err := c.do(ctx, http.MethodGet, "hints/"+strconv.Itoa(id), nil, &h); err != nil {
		return platform.Hint{}, err
	}
	return h.toHint(), nil
}

func (c *Client) Download(ctx context.Context, f platform.File, w io.Writer) error {
	ref, err := url.Parse(f.URL)
	if err != nil {
		return fmt.Errorf("invalid file URL %s: %w", f.URL, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base.ResolveReference(ref).String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", c.agent)

	resp, err := c.download.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized:
		return fmt.Errorf("%w: download link for %s rejected", platform.ErrUnauthorized, f.Name)
	case resp.StatusCode == http.StatusNotFound:
		return fmt.Errorf("%w: %s", platform.ErrNotFound, f.Name)
	case resp.StatusCode != http.StatusOK:
		return fmt.Errorf("download %s: %s", f.Name, resp.Status)
	}

	if _, err := io.Copy(w, resp.Body); err != nil {
		return fmt.Errorf("download %s: %w", f.Name, err)
	}
	return nil
}

type solveJSON struct {
	ChallengeID int       `json:"challenge_id"`
	Date        time.Time `json:"date"`
	User        *struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"user"`
}

func (c *Client) Solves(ctx context.Context) ([]platform.Solve, error) {
	var me userJSON
	if err := c.do(ctx, http.MethodGet, "users/me", nil, &me); err != nil {
		return nil, err
	}
	path := "users/me/solves"
	if me.TeamID != nil {
		path = "teams/me/solves"
	}

	var list []solveJSON
	if err := c.do(ctx, http.MethodGet, path, nil, &list); err != nil {
		return nil, err
	}
	solves := make([]platform.Solve, len(list))
	for i, s := range list {
		solve := platform.Solve{ChallengeID: s.ChallengeID, User: me.Name, Mine: true, Time: s.Date}
		if s.User != nil {
			solve.User = s.User.Name
			solve.Mine = s.User.ID == me.ID
		}
		solves[i] = solve
	}
	return solves, nil
}
