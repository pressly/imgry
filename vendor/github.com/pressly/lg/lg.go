package lg

import (
	"fmt"
	"runtime"

	"github.com/sirupsen/logrus"
)

var (
	DefaultLogger *logrus.Logger
	AlertFn       func(level logrus.Level, msg string)
)

func WithField(key string, value interface{}) *logrus.Entry {
	if DefaultLogger == nil {
		panic("lg: DefaultLogger is nil")
	}
	return DefaultLogger.WithField(key, value)
}

func WithFields(fields logrus.Fields) *logrus.Entry {
	if DefaultLogger == nil {
		panic("lg: DefaultLogger is nil")
	}
	return DefaultLogger.WithFields(fields)
}

func WithError(err error) *logrus.Entry {
	if DefaultLogger == nil {
		panic("lg: DefaultLogger is nil")
	}
	return DefaultLogger.WithError(err)
}

func Debugf(format string, args ...interface{}) {
	if DefaultLogger == nil {
		panic("lg: DefaultLogger is nil")
	}
	DefaultLogger.Debugf(format, args...)
}

func Infof(format string, args ...interface{}) {
	if DefaultLogger == nil {
		panic("lg: DefaultLogger is nil")
	}
	DefaultLogger.Infof(format, args...)
}

func Printf(format string, args ...interface{}) {
	if DefaultLogger == nil {
		panic("lg: DefaultLogger is nil")
	}
	DefaultLogger.Printf(format, args...)
}

func Warnf(format string, args ...interface{}) {
	if DefaultLogger == nil {
		panic("lg: DefaultLogger is nil")
	}
	DefaultLogger.Warnf(format, args...)
}

func Errorf(format string, args ...interface{}) {
	if DefaultLogger == nil {
		panic("lg: DefaultLogger is nil")
	}
	DefaultLogger.Errorf(format, args...)
}

func Alertf(format string, args ...interface{}) {
	if AlertFn != nil {
		_, file, line, _ := runtime.Caller(1)
		AlertFn(logrus.ErrorLevel, fmt.Sprintf("%s:%d ", file, line)+fmt.Sprintf(format, args...))
	}
	Errorf(format, args...)
}

func Fatalf(format string, args ...interface{}) {
	if DefaultLogger == nil {
		panic("lg: DefaultLogger is nil")
	}
	if AlertFn != nil {
		_, file, line, _ := runtime.Caller(1)
		AlertFn(logrus.FatalLevel, fmt.Sprintf("%s:%d ", file, line)+fmt.Sprintf(format, args...))
	}
	DefaultLogger.Fatalf(format, args...)
}

func Panicf(format string, args ...interface{}) {
	if DefaultLogger == nil {
		panic("lg: DefaultLogger is nil")
	}
	if AlertFn != nil {
		_, file, line, _ := runtime.Caller(1)
		AlertFn(logrus.PanicLevel, fmt.Sprintf("%s:%d ", file, line)+fmt.Sprintf(format, args...))
	}
	DefaultLogger.Panicf(format, args...)
}

func Debug(args ...interface{}) {
	if DefaultLogger == nil {
		panic("lg: DefaultLogger is nil")
	}
	DefaultLogger.Debug(args...)
}

func Info(args ...interface{}) {
	if DefaultLogger == nil {
		panic("lg: DefaultLogger is nil")
	}
	DefaultLogger.Info(args...)
}

func Print(args ...interface{}) {
	if DefaultLogger == nil {
		panic("lg: DefaultLogger is nil")
	}
	DefaultLogger.Print(args...)
}

func Warn(args ...interface{}) {
	if DefaultLogger == nil {
		panic("lg: DefaultLogger is nil")
	}
	DefaultLogger.Warn(args...)
}

func Error(args ...interface{}) {
	if DefaultLogger == nil {
		panic("lg: DefaultLogger is nil")
	}
	DefaultLogger.Error(args...)
}

func Alert(args ...interface{}) {
	if AlertFn != nil {
		_, file, line, _ := runtime.Caller(1)
		AlertFn(logrus.ErrorLevel, fmt.Sprintf("%s:%d ", file, line)+fmt.Sprint(args...))
	}
	Error(args...)
}

func Fatal(args ...interface{}) {
	if DefaultLogger == nil {
		panic("lg: DefaultLogger is nil")
	}
	if AlertFn != nil {
		_, file, line, _ := runtime.Caller(1)
		AlertFn(logrus.FatalLevel, fmt.Sprintf("%s:%d ", file, line)+fmt.Sprint(args...))
	}
	DefaultLogger.Fatal(args...)
}

func Panic(args ...interface{}) {
	if DefaultLogger == nil {
		panic("lg: DefaultLogger is nil")
	}
	if AlertFn != nil {
		_, file, line, _ := runtime.Caller(1)
		AlertFn(logrus.PanicLevel, fmt.Sprintf("%s:%d ", file, line)+fmt.Sprint(args...))
	}
	DefaultLogger.Panic(args...)
}

func Debugln(args ...interface{}) {
	if DefaultLogger == nil {
		panic("lg: DefaultLogger is nil")
	}
	DefaultLogger.Debugln(args...)
}

func Infoln(args ...interface{}) {
	if DefaultLogger == nil {
		panic("lg: DefaultLogger is nil")
	}
	DefaultLogger.Infoln(args...)
}

func Println(args ...interface{}) {
	if DefaultLogger == nil {
		panic("lg: DefaultLogger is nil")
	}
	DefaultLogger.Println(args...)
}

func Warnln(args ...interface{}) {
	if DefaultLogger == nil {
		panic("lg: DefaultLogger is nil")
	}
	DefaultLogger.Warnln(args...)
}

func Errorln(args ...interface{}) {
	if DefaultLogger == nil {
		panic("lg: DefaultLogger is nil")
	}
	DefaultLogger.Errorln(args...)
}

func Alertln(args ...interface{}) {
	if AlertFn != nil {
		_, file, line, _ := runtime.Caller(1)
		AlertFn(logrus.ErrorLevel, fmt.Sprintf("%s:%d ", file, line)+fmt.Sprintln(args...))
	}
	Errorln(args...)
}

func Fatalln(args ...interface{}) {
	if DefaultLogger == nil {
		panic("lg: DefaultLogger is nil")
	}
	if AlertFn != nil {
		_, file, line, _ := runtime.Caller(1)
		AlertFn(logrus.FatalLevel, fmt.Sprintf("%s:%d ", file, line)+fmt.Sprintln(args...))
	}
	DefaultLogger.Fatalln(args...)
}

func Panicln(args ...interface{}) {
	if DefaultLogger == nil {
		panic("lg: DefaultLogger is nil")
	}
	if AlertFn != nil {
		_, file, line, _ := runtime.Caller(1)
		AlertFn(logrus.PanicLevel, fmt.Sprintf("%s:%d ", file, line)+fmt.Sprintln(args...))
	}
	DefaultLogger.Panicln(args...)
}
