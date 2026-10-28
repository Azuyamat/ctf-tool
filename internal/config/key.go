package config

import "github.com/azuyamat/ctf/internal/event"

type Key struct {
	Name        string
	Description string
	Get         func(*Config) string
	Set         func(*Config, string) error
	GetEvent    func(*event.Event) string
	SetEvent    func(*event.Event, string) error
}

func (k Key) EventScoped() bool {
	return k.GetEvent != nil
}
