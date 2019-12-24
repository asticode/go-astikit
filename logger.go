package astikit

type Logger interface {
	Debug(v ...interface{})
	Debugf(format string, v ...interface{})
	Error(v ...interface{})
	Errorf(format string, v ...interface{})
	Info(v ...interface{})
	Infof(format string, v ...interface{})
}

type nopLogger struct{}

func newNopLogger() *nopLogger { return &nopLogger{} }

func (l *nopLogger) Debug(v ...interface{})                 {}
func (l *nopLogger) Debugf(format string, v ...interface{}) {}
func (l *nopLogger) Error(v ...interface{})                 {}
func (l *nopLogger) Errorf(format string, v ...interface{}) {}
func (l *nopLogger) Info(v ...interface{})                  {}
func (l *nopLogger) Infof(format string, v ...interface{})  {}
