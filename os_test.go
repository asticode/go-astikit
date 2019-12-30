package astikit

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

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
	e := "testdata/os/f"
	g := filepath.Join(p, "f")
	err = CopyFile(context.Background(), g, e, LocalCopyFileFunc)
	if err != nil {
		t.Errorf("expected no error, got %+v", err)
	}
	compareFile(t, e, g)

	// Move file
	e = g
	g = filepath.Join(p, "m")
	err = MoveFile(context.Background(), g, e, LocalCopyFileFunc)
	if err != nil {
		t.Errorf("expected no error, got %+v", err)
	}
	checkFile(t, g, "0")
	_, err = os.Stat(e)
	if !os.IsNotExist(err) {
		t.Error("expected true, got false")
	}

	// Copy dir
	e = "testdata/os/d"
	g = filepath.Join(p, "d")
	err = CopyFile(context.Background(), g, e, LocalCopyFileFunc)
	if err != nil {
		t.Errorf("expected no error, got %+v", err)
	}
	compareDir(t, e, g)
}
