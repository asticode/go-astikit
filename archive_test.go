package astikit

import (
	"context"
	"path/filepath"
	"testing"
)

func TestZip(t *testing.T) {
	// Get temp dir
	dir := t.TempDir()

	// With internal path
	i := "testdata/archive"
	f := filepath.Join(dir, "with-internal", "f.zip/root")
	err := Zip(context.Background(), f, i)
	if err != nil {
		t.Fatalf("expected no error, got %+v", err)
	}
	d := filepath.Join(dir, "with-internal", "d")
	err = Unzip(context.Background(), d, filepath.Join(dir, "with-internal", "f.zip/invalid"))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	err = Unzip(context.Background(), d, f)
	if err != nil {
		t.Fatalf("expected no error, got %+v", err)
	}
	compareDir(t, i, d)

	// Without internal path
	f = filepath.Join(dir, "without-internal", "f.zip")
	err = Zip(context.Background(), f, i)
	if err != nil {
		t.Fatalf("expected no error, got %+v", err)
	}
	d = filepath.Join(dir, "without-internal", "d")
	err = Unzip(context.Background(), d, f)
	if err != nil {
		t.Fatalf("expected no error, got %+v", err)
	}
	compareDir(t, i, d)
}
