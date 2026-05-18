package dash

import (
	"testing"
)

func TestFS(t *testing.T) {
	_, err := FS.ReadFile("dist/.gitkeep")
	if err != nil {
		t.Fatalf("expected .gitkeep to be embedded, got error: %v", err)
	}
	_, err = FS.ReadFile("dist/index.html")
	if err != nil {
		t.Fatalf("expected index.html to be embedded, got error: %v", err)
	}
}
