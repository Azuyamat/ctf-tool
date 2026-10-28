package ctfd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"

	"github.com/azuyamat/ctf/internal/platform"
)

var (
	formNonce  = regexp.MustCompile(`name="nonce"[^>]*value="([^"]+)"`)
	csrfNonce  = regexp.MustCompile(`['"]csrfNonce['"]\s*:\s*"([^"]+)"`)
	alertBlock = regexp.MustCompile(`(?s)class="alert[^"]*alert-danger[^"]*"[^>]*>(.*?)</div>`)
	htmlTag    = regexp.MustCompile(`<[^>]+>`)
)

func (c *Client) csrf(ctx context.Context) (string, error) {
	c.csrfOnce.Do(func() {
		page, status, err := c.page(ctx, c.http, "challenges", func(req *http.Request) { c.authorize(req) })
		switch {
		case err != nil:
			c.csrfErr = err
		case status >= 300 && status < 400:
			c.csrfErr = platform.ErrSessionExpired
		default:
			c.csrfToken, c.csrfErr = scrape(csrfNonce, page, "CSRF nonce")
		}
	})
	return c.csrfToken, c.csrfErr
}

func (c *Client) page(ctx context.Context, client *http.Client, path string, prepare func(*http.Request)) (string, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base.JoinPath(path).String(), nil)
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("User-Agent", c.agent)
	if prepare != nil {
		prepare(req)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return "", 0, fmt.Errorf("read %s: %w", path, err)
	}
	return string(body), resp.StatusCode, nil
}

func scrape(re *regexp.Regexp, page, what string) (string, error) {
	m := re.FindStringSubmatch(page)
	if m == nil {
		return "", fmt.Errorf("no %s found on the page, is this a CTFd instance?", what)
	}
	return m[1], nil
}

func Login(ctx context.Context, baseURL, name, password, version string) (string, error) {
	c, err := New(baseURL, platform.Auth{}, version)
	if err != nil {
		return "", err
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		return "", err
	}
	client := apiClient(jar)

	page, status, err := c.page(ctx, client, "login", nil)
	if err != nil {
		return "", err
	}
	if status != http.StatusOK {
		return "", fmt.Errorf("GET /login: %d %s", status, http.StatusText(status))
	}
	nonce, err := scrape(formNonce, page, "login form nonce")
	if err != nil {
		return "", err
	}

	form := url.Values{"name": {name}, "password": {password}, "nonce": {nonce}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base.JoinPath("login").String(), strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", c.agent)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))

	if resp.StatusCode < 300 || resp.StatusCode >= 400 {
		return "", loginFailure(string(body), resp.Status)
	}
	if loc := resp.Header.Get("Location"); strings.Contains(loc, "/login") {
		return "", errors.New("login was rejected")
	}
	for _, cookie := range jar.Cookies(c.base) {
		if cookie.Name == sessionCookie {
			return cookie.Value, nil
		}
	}
	return "", errors.New("logged in but no session cookie was set")
}

func loginFailure(page, status string) error {
	m := alertBlock.FindStringSubmatch(page)
	if m == nil {
		return fmt.Errorf("login failed: %s", status)
	}
	msg := strings.Join(strings.Fields(htmlTag.ReplaceAllString(m[1], " ")), " ")
	msg = strings.TrimSpace(strings.TrimPrefix(msg, "×"))
	if msg == "" {
		return fmt.Errorf("login failed: %s", status)
	}
	return fmt.Errorf("login failed: %s", msg)
}
