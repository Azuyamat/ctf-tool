package layout

import "path/filepath"

const (
	TemplatesDir       = "templates"
	EventTemplatesRoot = "event-templates"
)

type Layout struct {
	Base string
}

func (l Layout) EventDir(event string) string {
	return filepath.Join(l.Base, event)
}

func (l Layout) GlobalTemplatesDir() string {
	return filepath.Join(l.Base, TemplatesDir)
}

func (l Layout) EventTemplatesDir(event string) string {
	return filepath.Join(l.EventDir(event), TemplatesDir)
}

// GlobalEventTemplatesDir holds templates scaffolded onto an event's own folder
// when it is created, as opposed to the challenge templates copied into each new
// challenge.
func (l Layout) GlobalEventTemplatesDir() string {
	return filepath.Join(l.Base, EventTemplatesRoot)
}

func IsReserved(name string) bool {
	return name == TemplatesDir || name == EventTemplatesRoot
}
