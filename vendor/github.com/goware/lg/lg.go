package lg

import (
	"fmt"
	"io"
	stdlog "log"
	"runtime"

	"github.com/Sirupsen/logrus"
)

// lg package wraps logrus to allow us to use `lg.X` in our app code.
type (
	Fields logrus.Fields
	Level  logrus.Level
)

var (
	Logger  *logrus.Logger
	AlertFn func(level Level, msg string)
)

const (
	PanicLevel Level = iota
	FatalLevel
	ErrorLevel
	WarnLevel
	InfoLevel
	DebugLevel
)

func init() {
	Logger = logrus.New()

	// Defaults
	Logger.Level = logrus.InfoLevel
	Logger.Formatter = &logrus.TextFormatter{ForceColors: true, FullTimestamp: true}

	// Redirect standard logger
	stdlog.SetOutput(&logRedirectWriter{})
	stdlog.SetFlags(0)
}

// Proxy writer for any packages using the standard log.Println() stuff
type logRedirectWriter struct{}

func (l *logRedirectWriter) Write(p []byte) (n int, err error) {
	if len(p) > 0 {
		Logger.Infof("%s", p[:len(p)-1])
	}
	return len(p), nil
}

func StandardLogger() *logrus.Logger {
	return Logger
}

// SetOutput sets the standard logger output.
func SetOutput(out io.Writer) {
	Logger.Out = out
}

// SetFormatter sets the standard logger formatter.
func SetFormatter(formatter logrus.Formatter) {
	Logger.Formatter = formatter
}

func SetLevelString(lvl string) error {
	l, err := logrus.ParseLevel(lvl)
	if err != nil {
		return err
	}
	SetLevel(l)
	return nil
}

// SetLevel sets the standard logger level.
func SetLevel(level logrus.Level) {
	Logger.Level = level
}

// GetLevel returns the standard logger level.
func GetLevel() logrus.Level {
	return Logger.Level
}

// AddHook adds a hook to the standard logger hooks.
func AddHook(hook logrus.Hook) {
	Logger.Hooks.Add(hook)
}

// WithField creates an entry from the standard logger and adds a field to
// it. If you want multiple fields, use `WithFields`.
//
// Note that it doesn't log until you call Debug, Print, Info, Warn, Fatal
// or Panic on the Entry it returns.
func WithField(key string, value interface{}) *logrus.Entry {
	return Logger.WithField(key, value)
}

// WithFields creates an entry from the standard logger and adds multiple
// fields to it. This is simply a helper for `WithField`, invoking it
// once for each field.
//
// Note that it doesn't log until you call Debug, Print, Info, Warn, Fatal
// or Panic on the Entry it returns.
func WithFields(fields Fields) *logrus.Entry {
	return Logger.WithFields(logrus.Fields(fields))
}

// Debug logs a message at level Debug on the standard logger.
func Debug(args ...interface{}) {
	Logger.Debug(args...)
}

// Print logs a message at level Info on the standard logger.
func Print(args ...interface{}) {
	Logger.Print(args...)
}

// Info logs a message at level Info on the standard logger.
func Info(args ...interface{}) {
	Logger.Info(args...)
}

// Warn logs a message at level Warn on the standard logger.
func Warn(args ...interface{}) {
	Logger.Warn(args...)
}

// Warning logs a message at level Warn on the standard logger.
func Warning(args ...interface{}) {
	Logger.Warning(args...)
}

// Error logs a message at level Error on the standard logger.
func Error(args ...interface{}) {
	Logger.Error(args...)
}

// Alert logs a message at level Error on the standard logger and fires AlertFn().
func Alert(args ...interface{}) {
	if AlertFn != nil {
		_, file, line, _ := runtime.Caller(1)
		AlertFn(ErrorLevel, fmt.Sprintf("%s:%d ", file, line)+fmt.Sprint(args...))
	}
	Logger.Error(args...)
}

// Panic logs a message at level Panic on the standard logger and fires AlertFn().
func Panic(args ...interface{}) {
	if AlertFn != nil {
		_, file, line, _ := runtime.Caller(1)
		AlertFn(PanicLevel, fmt.Sprintf("%s:%d ", file, line)+fmt.Sprint(args...))
	}
	Logger.Panic(args...)
}

// Fatal logs a message at level Fatal on the standard logger and fires AlertFn().
func Fatal(args ...interface{}) {
	if AlertFn != nil {
		_, file, line, _ := runtime.Caller(1)
		AlertFn(FatalLevel, fmt.Sprintf("%s:%d ", file, line)+fmt.Sprint(args...))
	}
	Logger.Fatal(args...)
}

// Debugf logs a message at level Debug on the standard logger.
func Debugf(format string, args ...interface{}) {
	Logger.Debugf(format, args...)
}

// Printf logs a message at level Info on the standard logger.
func Printf(format string, args ...interface{}) {
	Logger.Printf(format, args...)
}

// Infof logs a message at level Info on the standard logger.
func Infof(format string, args ...interface{}) {
	Logger.Infof(format, args...)
}

// Warnf logs a message at level Warn on the standard logger.
func Warnf(format string, args ...interface{}) {
	Logger.Warnf(format, args...)
}

// Warningf logs a message at level Warn on the standard logger.
func Warningf(format string, args ...interface{}) {
	Logger.Warningf(format, args...)
}

// Errorf logs a message at level Error on the standard logger.
func Errorf(format string, args ...interface{}) {
	Logger.Errorf(format, args...)
}

// Alertf logs a message at level Error on the standard logger and fires AlertFn().
func Alertf(format string, args ...interface{}) {
	if AlertFn != nil {
		_, file, line, _ := runtime.Caller(1)
		AlertFn(ErrorLevel, fmt.Sprintf("%s:%d ", file, line)+fmt.Sprintf(format, args...))
	}
	Logger.Errorf(format, args...)
}

// Panicf logs a message at level Panic on the standard logger and fires AlertFn().
func Panicf(format string, args ...interface{}) {
	if AlertFn != nil {
		_, file, line, _ := runtime.Caller(1)
		AlertFn(PanicLevel, fmt.Sprintf("%s:%d ", file, line)+fmt.Sprintf(format, args...))
	}
	Logger.Panicf(format, args...)
}

// Fatalf logs a message at level Fatal on the standard logger. and fires AlertFn()
func Fatalf(format string, args ...interface{}) {
	if AlertFn != nil {
		_, file, line, _ := runtime.Caller(1)
		AlertFn(FatalLevel, fmt.Sprintf("%s:%d ", file, line)+fmt.Sprintf(format, args...))
	}
	Logger.Fatalf(format, args...)
}

// Debugln logs a message at level Debug on the standard logger.
func Debugln(args ...interface{}) {
	Logger.Debugln(args...)
}

// Println logs a message at level Info on the standard logger.
func Println(args ...interface{}) {
	Logger.Println(args...)
}

// Infoln logs a message at level Info on the standard logger.
func Infoln(args ...interface{}) {
	Logger.Infoln(args...)
}

// Warnln logs a message at level Warn on the standard logger.
func Warnln(args ...interface{}) {
	Logger.Warnln(args...)
}

// Warningln logs a message at level Warn on the standard logger.
func Warningln(args ...interface{}) {
	Logger.Warningln(args...)
}

// Errorln logs a message at level Error on the standard logger.
func Errorln(args ...interface{}) {
	Logger.Errorln(args...)
}

// Alertln logs a message at level Error on the standard logger and fires AlertFn().
func Alertln(args ...interface{}) {
	if AlertFn != nil {
		_, file, line, _ := runtime.Caller(1)
		AlertFn(ErrorLevel, fmt.Sprintf("%s:%d ", file, line)+fmt.Sprintln(args...))
	}
	Logger.Errorln(args...)
}

// Panicln logs a message at level Panic on the standard logger and fires AlertFn().
func Panicln(args ...interface{}) {
	if AlertFn != nil {
		_, file, line, _ := runtime.Caller(1)
		AlertFn(PanicLevel, fmt.Sprintf("%s:%d ", file, line)+fmt.Sprintln(args...))
	}
	Logger.Panicln(args...)
}

// Fatalln logs a message at level Fatal on the standard logger and fires AlertFn()
func Fatalln(args ...interface{}) {
	if AlertFn != nil {
		_, file, line, _ := runtime.Caller(1)
		AlertFn(FatalLevel, fmt.Sprintf("%s:%d ", file, line)+fmt.Sprintln(args...))
	}
	Logger.Fatalln(args...)
}
