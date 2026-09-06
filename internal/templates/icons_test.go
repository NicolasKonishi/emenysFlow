package templates

import (
	"strings"
	"testing"
)

func TestIconSVGKnownNames(t *testing.T) {
	names := []string{
		"mark", "home", "events", "layouts", "models", "catalog", "inventory",
		"rules", "settings", "checklists", "plus", "search", "arrow", "back",
		"logout", "close", "warning", "check", "return", "refresh", "pin",
	}
	for _, name := range names {
		markup := string(iconSVG(name))
		if markup == "" {
			t.Fatalf("icon %q should return markup", name)
		}
		if name == "mark" {
			if !strings.Contains(markup, "emenys-mark-ink.png") {
				t.Fatal("mark should use the extracted brand logo")
			}
			continue
		}
		if !strings.Contains(markup, "<svg") {
			t.Fatalf("icon %q should return svg markup", name)
		}
	}
}

func TestIconSVGUnknownName(t *testing.T) {
	if iconSVG("not-an-icon") != "" {
		t.Fatal("unknown icon should be empty")
	}
}
