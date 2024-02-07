package astikit

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestCopyFile(t *testing.T) {
	// Get temp dir
	p := t.TempDir()

	// Copy file
	e := "testdata/os/f"
	g := filepath.Join(p, "f")
	err := CopyFile(context.Background(), g, e, LocalCopyFileFunc)
	if err != nil {
		t.Fatalf("expected no error, got %+v", err)
	}
	compareFile(t, e, g)

	// Move file
	e = g
	g = filepath.Join(p, "m")
	err = MoveFile(context.Background(), g, e, LocalCopyFileFunc)
	if err != nil {
		t.Fatalf("expected no error, got %+v", err)
	}
	checkFile(t, g, "0")
	_, err = os.Stat(e)
	if !os.IsNotExist(err) {
		t.Fatal("expected true, got false")
	}

	// Copy dir
	e = "testdata/os/d"
	g = filepath.Join(p, "d")
	err = CopyFile(context.Background(), g, e, LocalCopyFileFunc)
	if err != nil {
		t.Fatalf("expected no error, got %+v", err)
	}
	compareDir(t, e, g)
}
