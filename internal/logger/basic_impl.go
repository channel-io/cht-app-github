package logger

import (
	"log"

	"go.uber.org/zap"

	"github.com/channel-io/cht-app-github/internal/config"
)

type BasicImpl struct {
	logger *zap.SugaredLogger
}

func NewBasicLogger(e *config.Config) *BasicImpl {
	return newLogger("channel-io/cht-app-github", e.Log.Level)
}

func newLogger(name string, loglevel string) *BasicImpl {
	loggerConfig := zap.NewProductionConfig()
	logLevel, err := zap.ParseAtomicLevel(loglevel)
	if err != nil {
		log.Fatal("Failed to create new logger, parsing log level", err)
	}
	loggerConfig.Level = logLevel
	logger, err := loggerConfig.Build()
	if err != nil {
		log.Fatal("Failed to create new logger, building zap logger", err)
	}
	sugaredLogger := logger.Named(name).Sugar()
	return &BasicImpl{
		logger: sugaredLogger,
	}
}

func (l *BasicImpl) Error(args ...interface{}) {
	l.logger.Error(args)
}

func (l *BasicImpl) Errorw(format string, args ...interface{}) {
	l.logger.Errorw(format, args...)
}

func (l *BasicImpl) Warn(args ...interface{}) {
	l.logger.Warn(args...)
}

func (l *BasicImpl) Warnw(format string, args ...interface{}) {
	l.logger.Warnw(format, args...)
}

func (l *BasicImpl) Info(args ...interface{}) {
	l.logger.Info(args...)
}

func (l *BasicImpl) Infow(format string, args ...interface{}) {
	l.logger.Infow(format, args...)
}

func (l *BasicImpl) Debug(args ...interface{}) {
	l.logger.Debug(args...)
}

func (l *BasicImpl) Debugw(format string, args ...interface{}) {
	l.logger.Debugw(format, args...)
}
