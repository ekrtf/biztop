package gui

import "testing"

func TestEmbeddedAssets(t *testing.T) {
	for _, name := range []string{"index.html", "style.css", "js/app.js", "js/mission.js"} {
		data, err := FS.ReadFile(name)
		if err != nil {
			t.Fatalf("FS.ReadFile(%q) error = %v", name, err)
		}
		if len(data) == 0 {
			t.Fatalf("FS.ReadFile(%q) returned empty file", name)
		}
	}
}
