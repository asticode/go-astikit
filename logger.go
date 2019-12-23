package astikit

type Logger interface {
	Debugf(format string, v ...interface{})
	Error(v ...interface{})
	Info(v ...interface{})
	Infof(format string, v ...interface{})
}

type nopLogger struct{}

func newNopLogger() *nopLogger { return &nopLogger{} }

func (l *nopLogger) Debugf(format string, v ...interface{}) {}
func (l *nopLogger) Error(v ...interface{})                 {}
func (l *nopLogger) Info(v ...interface{})                  {}
func (l *nopLogger) Infof(format string, v ...interface{})  {}
