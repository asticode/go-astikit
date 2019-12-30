package astikit

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func checkFile(t *testing.T, p string, e string) {
	b, err := ioutil.ReadFile(p)
	if err != nil {
		t.Errorf("expected no error, got %+v", err)
	}
	if g := string(b); e != g {
		t.Errorf("expected %s, got %s", e, g)
	}
}

func TestCopyFile(t *testing.T) {
	// Create temporary dir
	p, err := ioutil.TempDir("", "")
	if err != nil {
		t.Errorf("expected no error, got %+v", err)
		return
	}

	// Make sure the dir is deleted
	defer func() {
		if err = os.RemoveAll(p); err != nil {
			t.Errorf("expected no error, got %+v", err)
			return
		}
	}()

	// Copy file
	err = CopyFile(context.Background(), filepath.Join(p, "f"), "testdata/os/f", LocalCopyFileFunc)
	if err != nil {
		t.Errorf("expected no error, got %+v", err)
	}
	checkFile(t, filepath.Join(p, "f"), "0")

	// Copy dir
	err = CopyFile(context.Background(), filepath.Join(p, "d"), "testdata/os/d", LocalCopyFileFunc)
	if err != nil {
		t.Errorf("expected no error, got %+v", err)
	}
	checkFile(t, filepath.Join(p, "d", "f1"), "1")
	checkFile(t, filepath.Join(p, "d", "d1", "f11"), "2")
	checkFile(t, filepath.Join(p, "d", "d2", "f21"), "3")
	checkFile(t, filepath.Join(p, "d", "d2", "d21", "f211"), "4")
}
