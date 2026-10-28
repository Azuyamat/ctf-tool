package ctfd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/azuyamat/ctf/internal/platform"
)

const maxResponseSize = 10 << 20

var _ platform.Platform = (*Client)(nil)

const sessionCookie = "session"

type Client struct {
	base     *url.URL
	auth     platform.Auth
	http     *http.Client
	download *http.Client
	agent    string

	csrfOnce  sync.Once
	csrfToken string
	csrfErr   error
}

func New(baseURL string, auth platform.Auth, version string) (*Client, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	return &Client{
		base:     base,
		auth:     auth,
		http:     apiClient(nil),
		download: &http.Client{},
		agent:    "ctf/" + version,
	}, nil
}

func apiClient(jar http.CookieJar) *http.Client {
	return &http.Client{
		Timeout: 15 * time.Second,
		Jar:     jar,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func (c *Client) usesSession() bool {
	return c.auth.Token == "" && c.auth.Session != ""
}

func (c *Client) authorize(req *http.Request) {
	if c.usesSession() {
		req.AddCookie(&http.Cookie{Name: sessionCookie, Value: c.auth.Session})
		return
	}
	req.Header.Set("Authorization", "Token "+c.auth.Token)
}

type envelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Errors  json.RawMessage `json:"errors"`
	Message string          `json:"message"`
}

func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		reader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.base.JoinPath("api", "v1", path).String(), reader)
	if err != nil {
		return err
	}
	c.authorize(req)
	if c.usesSession() && method != http.MethodGet {
		csrf, err := c.csrf(ctx)
		if err != nil {
			return err
		}
		req.Header.Set("CSRF-Token", csrf)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.agent)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		if c.usesSession() {
			return platform.ErrSessionExpired
		}
		return fmt.Errorf("%w: redirected to %s", platform.ErrTokensDisabled, resp.Header.Get("Location"))
	}

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	var env envelope
	if !isJSON(resp.Header.Get("Content-Type")) || json.Unmarshal(raw, &env) != nil {
		if resp.StatusCode < 300 && c.usesSession() {
			return platform.ErrSessionExpired
		}
		if resp.StatusCode < 300 {
			return fmt.Errorf("%w: %s %s returned a non-JSON page", platform.ErrTokensDisabled, method, path)
		}
		return fmt.Errorf("%s %s: %s", method, path, resp.Status)
	}

	if resp.StatusCode >= 400 && !env.Success {
		err := statusError(resp, env.message())
		if c.usesSession() && errors.Is(err, platform.ErrUnauthorized) {
			return platform.ErrSessionExpired
		}
		return err
	}
	if !env.Success {
		return fmt.Errorf("%s %s: %s", method, path, env.message())
	}

	if out == nil {
		return nil
	}
	if err := json.Unmarshal(env.Data, out); err != nil {
		return fmt.Errorf("decode %s %s: %w", method, path, err)
	}
	return nil
}

func statusError(resp *http.Response, msg string) error {
	switch resp.StatusCode {
	case http.StatusBadRequest:
		return errors.New(msg)
	case http.StatusUnauthorized:
		return fmt.Errorf("%w: %s", platform.ErrUnauthorized, msg)
	case http.StatusForbidden:
		if strings.Contains(msg, "has not started") || strings.Contains(msg, "has ended") {
			return fmt.Errorf("%w: %s", platform.ErrNotStarted, msg)
		}
		return fmt.Errorf("%w: %s", platform.ErrUnauthorized, msg)
	case http.StatusNotFound:
		return platform.ErrNotFound
	case http.StatusTooManyRequests:
		return fmt.Errorf("%w: %s", platform.ErrRateLimited, msg)
	default:
		return fmt.Errorf("%s: %s", resp.Status, msg)
	}
}

func (e envelope) message() string {
	if e.Message != "" {
		return e.Message
	}
	if msg := flattenErrors(e.Errors); msg != "" {
		return msg
	}
	return "request failed"
}

func flattenErrors(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	var list []string
	if json.Unmarshal(raw, &list) == nil {
		return strings.Join(list, "; ")
	}

	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil {
		return string(raw)
	}
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var parts []string
	for _, key := range keys {
		var s string
		if json.Unmarshal(fields[key], &s) == nil {
			parts = append(parts, s)
			continue
		}
		var ss []string
		if json.Unmarshal(fields[key], &ss) == nil {
			parts = append(parts, ss...)
			continue
		}
		parts = append(parts, key+": "+string(fields[key]))
	}
	return strings.Join(parts, "; ")
}

func isJSON(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	return err == nil && mediaType == "application/json"
}

type userJSON struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	TeamID *int   `json:"team_id"`
	Score  int    `json:"score"`
}

type teamJSON struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Score int    `json:"score"`
}

func (u userJSON) toUser(t *teamJSON) platform.User {
	user := platform.User{
		ID:    u.ID,
		Name:  u.Name,
		Score: u.Score,
	}
	if t != nil {
		user.Team = t.Name
		user.Score = t.Score
	}
	return user
}

func (c *Client) Me(ctx context.Context) (platform.User, error) {
	var u userJSON
	if err := c.do(ctx, http.MethodGet, "users/me", nil, &u); err != nil {
		return platform.User{}, err
	}
	if u.TeamID == nil {
		return u.toUser(nil), nil
	}

	var t teamJSON
	if err := c.do(ctx, http.MethodGet, "teams/me", nil, &t); err != nil {
		return platform.User{}, err
	}
	return u.toUser(&t), nil
}
