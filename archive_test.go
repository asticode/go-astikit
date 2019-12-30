package astikit

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestZip(t *testing.T) {
	// Create temp dir
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("creating temp dir failed: %w", err)
	}

	// Make sure to delete temp dir
	defer os.RemoveAll(dir)

	// With internal path
	i := "testdata/archive"
	f := filepath.Join(dir, "with-internal", "f.zip/root")
	err = Zip(context.Background(), f, i)
	if err != nil {
		t.Errorf("expected no error, got %+v", err)
	}
	d := filepath.Join(dir, "with-internal", "d")
	err = Unzip(context.Background(), d, f)
	if err != nil {
		t.Errorf("expected no error, got %+v", err)
	}
	compareDir(t, i, d)

	// Without internal path
	i = "testdata/archive"
	f = filepath.Join(dir, "without-internal", "f.zip")
	err = Zip(context.Background(), f, i)
	if err != nil {
		t.Errorf("expected no error, got %+v", err)
	}
	d = filepath.Join(dir, "without-internal", "d")
	err = Unzip(context.Background(), d, f)
	if err != nil {
		t.Errorf("expected no error, got %+v", err)
	}
	compareDir(t, i, d)
}
