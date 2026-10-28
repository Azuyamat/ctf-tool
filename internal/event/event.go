package event

import (
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/azuyamat/ctf/internal/challenge"
	"github.com/azuyamat/ctf/internal/fsutil"
	"github.com/azuyamat/ctf/internal/layout"
)

var eventFileName = ".event.json"

const PlatformCTFd = "ctfd"

type Platform struct {
	Kind string `json:"kind"`
	URL  string `json:"url"`
}

type Event struct {
	Name       string    `json:"name"`
	Path       string    `json:"-"`
	Platform   *Platform `json:"platform,omitempty"`
	FlagFormat string    `json:"flagFormat,omitempty"`
	ReadmeName string    `json:"readmeName,omitempty"`
	FilesDir   string    `json:"filesDir,omitempty"`
}

func New(l layout.Layout, name string) (*Event, error) {
	if err := ValidateName(name); err != nil {
		return nil, err
	}
	return &Event{
		Name: name,
		Path: l.EventDir(name),
	}, nil
}

func Load(l layout.Layout, name string) (*Event, error) {
	e, err := New(l, name)
	if err != nil {
		return nil, err
	}

	if err := fsutil.ReadJSON(e.configFilePath(), e); errors.Is(err, fs.ErrNotExist) {
		return e, nil
	} else if err != nil {
		return nil, fmt.Errorf("read event: %w", err)
	}
	e.Name = name
	for _, value := range []string{e.ReadmeName, e.FilesDir} {
		if value == "" {
			continue
		}
		if err := challenge.ValidateName(value); err != nil {
			return nil, fmt.Errorf("event %s: %w", name, err)
		}
	}
	return e, nil
}

func (e *Event) Exists() bool {
	_, err := os.Lstat(e.configFilePath())
	return err == nil
}

func (e *Event) configFilePath() string {
	return filepath.Join(e.Path, eventFileName)
}

const gitignoreName = ".gitignore"

var gitignore = challenge.LocalFileName + "\n*.part\n"

func (e *Event) Create() error {
	if err := os.MkdirAll(e.Path, 0755); err != nil {
		return err
	}
	path := filepath.Join(e.Path, gitignoreName)
	if _, err := os.Lstat(path); !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return fsutil.WriteFile(path, []byte(gitignore), 0644)
}

func (e *Event) Save() error {
	if err := fsutil.WriteJSON(e.configFilePath(), e, 0644); err != nil {
		return fmt.Errorf("write event: %w", err)
	}
	return nil
}

func (p *Platform) Host() string {
	u, err := url.Parse(p.URL)
	if err != nil || u.Host == "" {
		return p.URL
	}
	return u.Host + strings.TrimRight(u.Path, "/")
}

func (e *Event) Link(kind, rawURL string, insecure bool) error {
	u, err := NormalizeURL(rawURL, insecure)
	if err != nil {
		return err
	}
	e.Platform = &Platform{Kind: kind, URL: u}
	return nil
}

func (e *Event) Unlink() bool {
	linked := e.Platform != nil
	e.Platform = nil
	return linked
}

func (e *Event) SetFlagFormat(pattern string) error {
	if _, err := regexp.Compile(pattern); err != nil {
		return fmt.Errorf("invalid flag format: %w", err)
	}
	e.FlagFormat = pattern
	return nil
}

func (e *Event) MatchesFlagFormat(flag string) bool {
	if e.FlagFormat == "" {
		return true
	}
	re, err := regexp.Compile("^(?:" + e.FlagFormat + ")$")
	return err != nil || re.MatchString(flag)
}

func NormalizeURL(rawURL string, insecure bool) (string, error) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || u.Host == "" {
		return "", fmt.Errorf("invalid URL: %s", rawURL)
	}
	switch u.Scheme {
	case "https":
	case "http":
		if !insecure {
			return "", fmt.Errorf("refusing plain http URL without --insecure: %s", rawURL)
		}
	default:
		return "", fmt.Errorf("URL must start with https://: %s", rawURL)
	}

	u.Path = strings.TrimRight(u.Path, "/")
	u.Path = strings.TrimSuffix(u.Path, "/api/v1")
	u.Path = strings.TrimRight(u.Path, "/")
	u.RawPath, u.RawQuery, u.Fragment = "", "", ""
	return u.String(), nil
}

func ValidateName(name string) error {
	if !fsutil.ValidName(name) {
		return fmt.Errorf("invalid event name: %s", name)
	}
	if layout.IsReserved(name) {
		return fmt.Errorf("event name is reserved: %s", name)
	}
	return nil
}
