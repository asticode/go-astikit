package astikit

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func fileContent(t *testing.T, path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected no error, got %+v", err)
	}
	return string(b)
}

func checkFile(t *testing.T, p string, e string) {
	if g := fileContent(t, p); e != g {
		t.Fatalf("expected %s, got %s", e, g)
	}
}

func compareFile(t *testing.T, expectedPath, gotPath string) {
	if e, g := fileContent(t, expectedPath), fileContent(t, gotPath); e != g {
		t.Fatalf("expected %s, got %s", e, g)
	}
}

func dirContent(t *testing.T, dir string) (o map[string]string) {
	// Make sure to clean dir path so that we get consistent path separator with filepath.Walk
	dir = filepath.Clean(dir)

	// Walk
	o = make(map[string]string)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, e error) (err error) {
		// Check error
		if e != nil {
			return e
		}

		// Don't process dirs
		if info.IsDir() {
			return
		}

		// Read
		var b []byte
		if b, err = os.ReadFile(path); err != nil {
			return
		}

		// Add to map
		o[strings.TrimPrefix(path, dir)] = string(b)
		return
	})
	if err != nil {
		t.Fatalf("expected no error, got %+v", err)
	}
	return
}

func checkDir(t *testing.T, p string, e map[string]string) {
	for k, v := range e {
		delete(e, k)
		e[filepath.Clean(k)] = v
	}
	if g := dirContent(t, p); !reflect.DeepEqual(e, g) {
		t.Fatalf("expected %s, got %s", e, g)
	}
}

func compareDir(t *testing.T, ePath, gPath string) {
	if e, g := dirContent(t, ePath), dirContent(t, gPath); !reflect.DeepEqual(e, g) {
		t.Fatalf("expected %+v, got %+v", e, g)
	}
}
