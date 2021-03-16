package cache

import "github.com/sirupsen/logrus"

type logger struct {
	*logrus.Entry
}

func (l *logger) Errorf(format string, v ...interface{}) {
	if l.Entry == nil {
		return
	}

	l.Entry.Errorf(format, v...)
}

// Infof logs an Debug message to the logger specified in opts.
func (l *logger) Infof(format string, v ...interface{}) {
	if l.Entry == nil {
		return
	}

	l.Entry.Debugf(format, v...)
}

// Warningf logs a WARNING message to the logger specified in opts.
func (l *logger) Warningf(format string, v ...interface{}) {
	if l.Entry == nil {
		return
	}

	l.Entry.Warningf(format, v...)
}

// Debugf logs a DEBUG message to the logger specified in opts.
func (l *logger) Debugf(format string, v ...interface{}) {
	if l.Entry == nil {
		return
	}

	l.Entry.Debugf(format, v...)
}
