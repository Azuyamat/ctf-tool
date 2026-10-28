package layout

import "path/filepath"

const TemplatesDir = "templates"

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

func IsReserved(name string) bool {
	return name == TemplatesDir
}
