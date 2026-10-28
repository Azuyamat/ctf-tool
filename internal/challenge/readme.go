package challenge

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/azuyamat/ctf/internal/fsutil"
)

const DefaultReadmeName = "README.md"

var ReadmeName = DefaultReadmeName

func (h Hint) Label(n int) string {
	if h.Title != "" {
		return h.Title
	}
	return fmt.Sprintf("Hint %d", n)
}

func (h Hint) CostLabel() string {
	if h.Cost == 0 {
		return "free"
	}
	return fmt.Sprintf("%d pts", h.Cost)
}

func (c *Challenge) Readme() string {
	r := c.Remote
	var b strings.Builder

	fmt.Fprintf(&b, "# %s\n\n", r.Name)
	meta := []string{"**" + r.Category + "**"}
	if r.Points > 0 {
		meta = append(meta, fmt.Sprintf("%d pts", r.Points))
	}
	if len(r.Tags) > 0 {
		meta = append(meta, "tags: "+strings.Join(r.Tags, ", "))
	}
	if r.MaxAttempts > 0 {
		meta = append(meta, fmt.Sprintf("%d attempts", r.MaxAttempts))
	}
	b.WriteString(strings.Join(meta, " · ") + "\n")
	if r.URL != "" {
		fmt.Fprintf(&b, "\n[Open on the CTF site](%s)\n", r.URL)
	}

	if c.Connection != "" {
		fmt.Fprintf(&b, "\n```\n%s\n```\n", c.Connection)
	}
	if desc := strings.TrimSpace(r.Description); desc != "" {
		fmt.Fprintf(&b, "\n%s\n", desc)
	}

	if len(r.Files) > 0 {
		b.WriteString("\n## Files\n\n")
		for _, name := range r.Files {
			fmt.Fprintf(&b, "- `%s`\n", name)
		}
	}

	if len(r.Hints) > 0 {
		b.WriteString("\n## Hints\n\n")
		for i, h := range r.Hints {
			n := i + 1
			if !h.Unlocked {
				fmt.Fprintf(&b, "%d. **%s** (%s): locked, run `ctf hint %d`\n", n, h.Label(n), h.CostLabel(), n)
				continue
			}
			fmt.Fprintf(&b, "%d. **%s** (%s)\n\n", n, h.Label(n), h.CostLabel())
			for _, line := range strings.Split(strings.TrimSpace(h.Content), "\n") {
				fmt.Fprintf(&b, "   %s\n", line)
			}
			b.WriteString("\n")
		}
	}
	return strings.TrimRight(b.String(), "\n") + "\n"
}

func (c *Challenge) WriteReadme() (bool, error) {
	path := filepath.Join(c.Path, ReadmeName)
	data := []byte(c.Readme())

	existing, err := os.ReadFile(path)
	if err == nil && bytes.Equal(existing, data) {
		return false, nil
	}
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return false, err
	}
	if err := fsutil.WriteFile(path, data, 0644); err != nil {
		return false, fmt.Errorf("write %s: %w", ReadmeName, err)
	}
	return true, nil
}
