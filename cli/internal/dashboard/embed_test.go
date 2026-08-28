package dashboard

import (
	"io/fs"
	"testing"
)

func TestEmbeddedWebHasIndex(t *testing.T) {
	staticFS, err := fs.Sub(webFS, "web/build")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fs.Stat(staticFS, "index.html"); err != nil {
		t.Fatalf("embedded web/build/index.html: %v", err)
	}
}
