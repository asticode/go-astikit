package astikit

import (
	"bytes"
	"context"
	"io"
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
	err := CopyFile(context.Background(), "/path/to/dst", "testdata/ssh/f", SSHCopyFileFunc(func() (SSHSession, *Closer, error) {
		c++
		return s, NewCloser(), nil
	}))
	if err != nil {
		t.Errorf("expected no error, got %+v", err)
	}
	if e := 2; c != e {
		t.Errorf("expected %v, got %v", e, c)
	}
	if e := []string{"mkdir -p /path/to", "scp -qt /path/to"}; !reflect.DeepEqual(e, s.cmds) {
		t.Errorf("expected %+v, got %+v", e, s.cmds)
	}
	if e, g := "C0755 1 dst\n0\x00", s.buf.String(); e != g {
		t.Errorf("expected %s, got %s", e, g)
	}
}
