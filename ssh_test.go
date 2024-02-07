package astikit

import (
	"bytes"
	"context"
	"io"
	"path/filepath"
	"reflect"
	"testing"
)

type mockedSSHSession struct {
	buf  *bytes.Buffer
	cmds []string
}

func newMockedSSHSession() *mockedSSHSession {
	return &mockedSSHSession{buf: &bytes.Buffer{}}
}

func (s *mockedSSHSession) Run(cmd string) error {
	s.cmds = append(s.cmds, cmd)
	return nil
}

func (s *mockedSSHSession) Start(cmd string) error {
	s.cmds = append(s.cmds, cmd)
	return nil
}

func (s *mockedSSHSession) StdinPipe() (io.WriteCloser, error) {
	return NopCloser(s.buf), nil
}

func (s *mockedSSHSession) Wait() error { return nil }

func TestSSHCopyFunc(t *testing.T) {
	var c int
	s := newMockedSSHSession()
	err := CopyFile(context.Background(), "/path/to with space/dst", "testdata/ssh/f", SSHCopyFileFunc(func() (SSHSession, *Closer, error) {
		c++
		return s, NewCloser(), nil
	}))
	if err != nil {
		t.Fatalf("expected no error, got %+v", err)
	}
	if e := 2; c != e {
		t.Fatalf("expected %v, got %v", e, c)
	}
	if e := []string{"mkdir -p " + filepath.Clean("/path/to\\ with\\ space"), "scp -qt " + filepath.Clean("/path/to\\ with\\ space")}; !reflect.DeepEqual(e, s.cmds) {
		t.Fatalf("expected %+v, got %+v", e, s.cmds)
	}
	if e1, e2, e3, g := "C0775 1 dst\n0\x00", "C0755 1 dst\n0\x00", "C0666 1 dst\n0\x00", s.buf.String(); g != e1 && g != e2 && g != e3 {
		t.Fatalf("expected %s or %s or %s, got %s", e1, e2, e3, g)
	}
}
