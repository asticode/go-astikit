package astikit

import (
	"testing"
)

func TestLoggerLevel(t *testing.T) {
	var l LoggerLevel
	for _, v := range []struct {
		l LoggerLevel
		s string
	}{
		{
			l: LoggerLevelDebug,
			s: "debug",
		},
		{
			l: LoggerLevelError,
			s: "error",
		},
		{
			l: LoggerLevelFatal,
			s: "fatal",
		},
		{
			l: LoggerLevelInfo,
			s: "info",
		},
		{
			l: LoggerLevelWarn,
			s: "warn",
		},
	} {
		if e, g := v.s, v.l.String(); e != g {
			t.Errorf("expected %s, got %s", e, g)
		}
		b, err := v.l.MarshalText()
		if err != nil {
			t.Errorf("expected no error, got %s", err)
		}
		if e, g := v.s, string(b); e != g {
			t.Errorf("expected %s, got %s", e, g)
		}
		if e, g := v.l, LoggerLevelFromString(v.s); e != g {
			t.Errorf("expected %s, got %s", e, g)
		}
		err = l.UnmarshalText([]byte(v.s))
		if err != nil {
			t.Errorf("expected no error, got %s", err)
		}
		if e, g := v.l, l; e != g {
			t.Errorf("expected %s, got %s", e, g)
		}
	}
}
