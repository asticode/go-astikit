package astikit

// StdLogger represents a standard logger
type StdLogger interface {
	Print(v ...interface{})
	Printf(format string, v ...interface{})
}

// SeverityLogger represents a severity logger
type SeverityLogger interface {
	Debug(v ...interface{})
	Debugf(format string, v ...interface{})
	Error(v ...interface{})
	Errorf(format string, v ...interface{})
	Info(v ...interface{})
	Infof(format string, v ...interface{})
}

type severityLogger struct {
	debug, error, info    func(v ...interface{})
	debugf, errorf, infof func(format string, v ...interface{})
}

func newSeverityLogger() *severityLogger {
	return &severityLogger{
		debug:  func(v ...interface{}) {},
		debugf: func(format string, v ...interface{}) {},
		error:  func(v ...interface{}) {},
		errorf: func(format string, v ...interface{}) {},
		info:   func(v ...interface{}) {},
		infof:  func(format string, v ...interface{}) {},
	}
}

func (l *severityLogger) Debug(v ...interface{})                 { l.debug(v...) }
func (l *severityLogger) Debugf(format string, v ...interface{}) { l.debugf(format, v...) }
func (l *severityLogger) Error(v ...interface{})                 { l.error(v...) }
func (l *severityLogger) Errorf(format string, v ...interface{}) { l.errorf(format, v...) }
func (l *severityLogger) Info(v ...interface{})                  { l.info(v...) }
func (l *severityLogger) Infof(format string, v ...interface{})  { l.infof(format, v...) }

// AdaptStdLogger transforms an StdLogger into a SeverityLogger
func AdaptStdLogger(i StdLogger) SeverityLogger {
	l := newSeverityLogger()
	if i != nil {
		if v, ok := i.(SeverityLogger); ok {
			l.debug = v.Debug
			l.debugf = v.Debugf
			l.error = v.Error
			l.errorf = v.Errorf
			l.info = v.Info
			l.infof = v.Infof
		} else {
			l.debug = i.Print
			l.debugf = i.Printf
			l.error = i.Print
			l.errorf = i.Printf
			l.info = i.Print
			l.infof = i.Printf
		}
	} else {
		l.debug = func(v ...interface{}) {}
		l.debugf = func(format string, v ...interface{}) {}
		l.error = func(v ...interface{}) {}
		l.errorf = func(format string, v ...interface{}) {}
		l.info = func(v ...interface{}) {}
		l.infof = func(format string, v ...interface{}) {}
	}
	return l
}
