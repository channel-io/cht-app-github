package logger

type Logger interface {
	Error(args ...interface{})
	Errorw(msg string, args ...interface{})
	Warn(args ...interface{})
	Warnw(msg string, args ...interface{})
	Info(args ...interface{})
	Infow(msg string, args ...interface{})
	Debug(args ...interface{})
	Debugw(msg string, args ...interface{})
}
