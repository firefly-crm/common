package logger

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"time"
)

type (
	// Logger interface contains a subset of the logrus methods that we use
	Logger interface {
		// Debug Utility for debug log messages
		Debugf(arg0 interface{}, args ...interface{})

		// Errorf Utility for error log messages
		Errorf(arg0 interface{}, args ...interface{})

		// Info Utility for info log messages
		Infof(arg0 interface{}, args ...interface{})

		// Trace Utility for trace log messages
		Tracef(arg0 interface{}, args ...interface{})

		// Warn Utility for error log messages
		Warnf(arg0 interface{}, args ...interface{})

		// Including custom field into structured log
		WithField(key string, value interface{}) Logger
	}

	logrusWrapper struct {
		log *logrus.Entry
	}
)

func init() {
	logrus.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
	})
	logrus.SetLevel(logrus.TraceLevel)
}

var DefaultLogger = New(logrus.DebugLevel, os.Stdout)

func New(level logrus.Level, out io.Writer) Logger {
	l := logrus.StandardLogger()
	l.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
	})
	l.SetLevel(level)
	l.SetOutput(out)

	return logrusWrapper{
		log: logrus.NewEntry(l),
	}
}

func (lw logrusWrapper) Infof(arg0 interface{}, args ...interface{}) {
	switch first := arg0.(type) {
	case string:
		lw.log.Infof(first, lw.securedSlice(args)...)
	default:
		allArgs := append([]interface{}{arg0}, args...)
		fmt.Printf("infof slice: %v\n", allArgs)
		lw.log.Info(lw.securedSlice(allArgs)...)
	}
}

func (lw logrusWrapper) Tracef(arg0 interface{}, args ...interface{}) {
	switch first := arg0.(type) {
	case string:
		lw.log.Tracef(first, lw.securedSlice(args)...)
	default:
		allArgs := append([]interface{}{arg0}, args...)
		lw.log.Trace(lw.securedSlice(allArgs)...)
	}
}

func (lw logrusWrapper) Debugf(arg0 interface{}, args ...interface{}) {
	switch first := arg0.(type) {
	case string:
		lw.log.Debugf(first, lw.securedSlice(args)...)
	default:
		allArgs := append([]interface{}{arg0}, args...)
		lw.log.Debug(lw.securedSlice(allArgs)...)
	}
}

func (lw logrusWrapper) Warnf(arg0 interface{}, args ...interface{}) {
	switch first := arg0.(type) {
	case string:
		args = lw.securedSlice(args)
		lw.log.Warnf(first, args...)
	default:
		allArgs := append([]interface{}{arg0}, args...)
		allArgs = lw.securedSlice(allArgs)
		lw.log.Warn(allArgs...)
	}
}

func (lw logrusWrapper) Errorf(arg0 interface{}, args ...interface{}) {
	switch first := arg0.(type) {
	case string:
		lw.log.Errorf(first, lw.securedSlice(args)...)
	default:
		allArgs := append([]interface{}{arg0}, args...)
		allArgs = lw.securedSlice(allArgs)
		lw.log.Error(allArgs...)
	}
}

func (lw logrusWrapper) WithField(key string, value interface{}) Logger {
	if value == nil {
		return lw
	}

	if b, ok := value.([]byte); ok {
		return logrusWrapper{
			log: lw.log.WithField(key, string(b)),
		}
	}

	newWrapper := lw
	newWrapper.log = lw.log.WithField(key, stringMarshaller{value})
	return newWrapper
}

func (lw logrusWrapper) securedSlice(s []interface{}) []interface{} {
	ss := make([]interface{}, len(s))
	for i := range s {
		switch s[i].(type) {
		case string, []byte, int, int64, uint, uint8, float32, float64, rune, error:
			ss[i] = s[i]
		default:
			//ss[i] = sensitive.NewValue(s[i], lw.cfg.HidePackages)
		}
	}

	return ss
}

type stringMarshaller struct {
	v interface{}
}

func (sm stringMarshaller) MarshalJSON() (b []byte, err error) {
	b, err = json.Marshal(sm.v)
	if err != nil {
		return nil, err
	}

	if len(b) > 0 && (b[0] == '{' || b[0] == '[' || string(b) == "null") {
		return []byte(fmt.Sprintf("%q", b)), nil
	}

	return b, nil
}

// Crashf writes log message and crash the application
func Crashf(message string, args ...interface{}) {
	logrus.StandardLogger().Panicf(message, args...)
}
