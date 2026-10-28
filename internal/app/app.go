package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/azuyamat/ctf/internal/challenge"
	"github.com/azuyamat/ctf/internal/config"
	"github.com/azuyamat/ctf/internal/credentials"
	"github.com/azuyamat/ctf/internal/event"
	"github.com/azuyamat/ctf/internal/fsutil"
	"github.com/azuyamat/ctf/internal/layout"
	"github.com/azuyamat/ctf/internal/platform"
	"github.com/azuyamat/ctf/internal/platform/ctfd"
	"github.com/azuyamat/ctf/internal/state"
)

const TokenEnv = "CTF_TOKEN"

type App struct {
	Config  *config.Config
	State   *state.State
	Version string
	loaded  state.State
}

func Load(version string) (*App, error) {
	cfg, err := config.LoadOrCreateConfig()
	if err != nil {
		return nil, err
	}
	st, err := state.Load()
	if err != nil {
		return nil, err
	}
	return &App{Config: cfg, State: st, Version: version, loaded: *st}, nil
}

func (a *App) Save() error {
	if *a.State == a.loaded {
		return nil
	}
	if err := a.State.Save(); err != nil {
		return err
	}
	a.loaded = *a.State
	return nil
}

func (a *App) Layout() layout.Layout {
	return layout.Layout{Base: a.Config.BaseDir}
}

func (a *App) ChallengeRoot() string {
	return a.Layout().EventDir(a.State.Event)
}

func (a *App) Challenge(fallback bool) (*challenge.Challenge, error) {
	var ch *challenge.Challenge
	var err error
	if fallback {
		ch, err = challenge.LoadCurrent(a.Config.BaseDir, a.State.LastChallenge)
	} else {
		ch, err = challenge.LoadFromCwd(a.Config.BaseDir)
	}
	if err != nil {
		return nil, fmt.Errorf("load challenge: %w", err)
	}
	return ch, nil
}

func (a *App) Event() (*event.Event, error) {
	if a.State.Event == "" {
		return nil, errors.New("no event set, run ctf event <name>")
	}
	return event.Load(a.Layout(), a.State.Event)
}

func (a *App) LinkedEvent() (*event.Event, error) {
	return requireLinked(a.Event())
}

func (a *App) EventFor(ch *challenge.Challenge) (*event.Event, error) {
	rel, ok := fsutil.Within(a.Config.BaseDir, ch.Path)
	if !ok {
		return nil, fmt.Errorf("%s is outside %s", ch.Path, a.Config.BaseDir)
	}
	name, _, found := strings.Cut(filepath.ToSlash(rel), "/")
	if !found {
		return nil, fmt.Errorf("%s isn't inside an event", ch.Path)
	}
	return event.Load(a.Layout(), name)
}

func (a *App) LinkedEventFor(ch *challenge.Challenge) (*event.Event, error) {
	return requireLinked(a.EventFor(ch))
}

func requireLinked(ev *event.Event, err error) (*event.Event, error) {
	if err != nil {
		return nil, err
	}
	if ev.Platform == nil {
		return nil, fmt.Errorf("event %s is not linked, run ctf event %s --ctfd <url>", ev.Name, ev.Name)
	}
	return ev, nil
}

func (a *App) Auth(ev *event.Event) (platform.Auth, error) {
	if token := os.Getenv(TokenEnv); token != "" {
		return platform.Auth{Token: token}, nil
	}
	creds, err := credentials.Load()
	if err != nil {
		return platform.Auth{}, err
	}
	if auth := creds.Auth(ev.Platform.URL); auth.Secret() != "" {
		return auth, nil
	}
	return platform.Auth{}, fmt.Errorf("not logged in to %s, run ctf login", ev.Platform.Host())
}

func (a *App) Connect(ev *event.Event) (platform.Platform, error) {
	auth, err := a.Auth(ev)
	if err != nil {
		return nil, err
	}
	return a.Platform(ev, auth)
}

func (a *App) Platform(ev *event.Event, auth platform.Auth) (platform.Platform, error) {
	switch ev.Platform.Kind {
	case event.PlatformCTFd:
		client, err := ctfd.New(ev.Platform.URL, auth, a.Version)
		if err != nil {
			return nil, err
		}
		return client, nil
	default:
		return nil, fmt.Errorf("unsupported platform: %s", ev.Platform.Kind)
	}
}
