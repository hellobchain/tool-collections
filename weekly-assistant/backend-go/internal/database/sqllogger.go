package database

import (
	"context"
	"time"

	"github.com/hellobchain/wswlog/wlogging"
	"gorm.io/gorm/logger"
)

type sqlLogger struct {
	logger   *wlogging.WswLogger
	LogLevel logger.LogLevel
}

func (l *sqlLogger) LogMode(logLevel logger.LogLevel) logger.Interface {
	newSqlLogger := *l
	newSqlLogger.LogLevel = logLevel
	return &newSqlLogger
}
func (l *sqlLogger) Info(ctx context.Context, s string, i ...interface{}) {
	if l.LogLevel >= logger.Info {
		l.logger.Infof(s, i...)
	}
}
func (l *sqlLogger) Warn(ctx context.Context, s string, i ...interface{}) {
	if l.LogLevel >= logger.Warn {
		l.logger.Warnf(s, i...)
	}
}
func (l *sqlLogger) Error(ctx context.Context, s string, i ...interface{}) {
	if l.LogLevel >= logger.Error {
		l.logger.Errorf(s, i...)
	}
}

func (l *sqlLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= logger.Silent {
		return
	}
	sql, rows := fc()
	l.logger.Infof("SQL: %s [rows: %d] [time: %v]", sql, rows, time.Since(begin))
}

func NewSqlLogger(logger *wlogging.WswLogger, logLevel logger.LogLevel) *sqlLogger {
	sqlLogger := &sqlLogger{logger: logger, LogLevel: logLevel}
	return sqlLogger
}
