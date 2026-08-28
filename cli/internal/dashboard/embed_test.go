package dashboard

import (
	"io/fs"
	"testing"
)

func TestEmbeddedWebHasIndex(t *testing.T) {
	staticFS, err := fs.Sub(embeddedWeb, embeddedWebRoot)
	if err != nil {
		t.Fatalf("fs.Sub(embeddedWeb, %q): %v", embeddedWebRoot, err)
	}

	info, err := fs.Stat(staticFS, "index.html")
	if err != nil {
		t.Fatalf("index.html missing from embedded web (%s): %v", embeddedWebRoot, err)
	}
	if info.IsDir() {
		t.Fatal("index.html is a directory")
	}
}
