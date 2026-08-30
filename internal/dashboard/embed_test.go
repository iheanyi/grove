package dashboard

import (
	"bytes"
	"errors"
	"io/fs"
	"testing"
)

func TestEmbeddedWebMatchesStubOrFullUI(t *testing.T) {
	staticFS, err := fs.Sub(webFS, "web/build")
	if err != nil {
		t.Fatal(err)
	}

	index, err := fs.ReadFile(staticFS, "index.html")
	if err != nil {
		t.Fatalf("embedded web/build/index.html: %v", err)
	}

	appInfo, err := fs.Stat(staticFS, "_app")
	if err == nil {
		if !appInfo.IsDir() {
			t.Fatal("embedded web/build/_app is not a directory")
		}
		if bytes.Contains(index, []byte(`data-grove-embed="stub"`)) {
			t.Fatal("full dashboard embed must not use the stub index.html")
		}

		hasFile := false
		if err := fs.WalkDir(staticFS, "_app", func(_ string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() {
				hasFile = true
			}
			return nil
		}); err != nil {
			t.Fatalf("walk embedded web/build/_app: %v", err)
		}
		if !hasFile {
			t.Fatal("embedded web/build/_app must contain at least one file")
		}
		return
	}

	if !bytes.Contains(index, []byte(`data-grove-embed="stub"`)) {
		t.Fatal("stub dashboard embed must mark index.html with data-grove-embed=\"stub\"")
	}
	if !isNotExist(err) {
		t.Fatalf("stat embedded web/build/_app: %v", err)
	}
}

func isNotExist(err error) bool {
	return errors.Is(err, fs.ErrNotExist)
}
