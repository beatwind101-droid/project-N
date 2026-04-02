package logging

import (
	"io"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-hclog"
)

// Config 日志配置
type Config struct {
	Level  string
	Format string
	File   string
}

// Logger 日志接口
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Fatal(msg string, args ...interface{})
	Named(name string) Logger
	With(args ...interface{}) Logger
}

// HCLoggerAdapter hclog适配器
type HCLoggerAdapter struct {
	logger hclog.Logger
}

// NewHCLoggerAdapter 创建hclog适配器
func NewHCLoggerAdapter(logger hclog.Logger) *HCLoggerAdapter {
	return &HCLoggerAdapter{
		logger: logger,
	}
}

// Debug 调试日志
func (l *HCLoggerAdapter) Debug(msg string, args ...interface{}) {
	l.logger.Debug(msg, args...)
}

// Info 信息日志
func (l *HCLoggerAdapter) Info(msg string, args ...interface{}) {
	l.logger.Info(msg, args...)
}

// Warn 警告日志
func (l *HCLoggerAdapter) Warn(msg string, args ...interface{}) {
	l.logger.Warn(msg, args...)
}

// Error 错误日志
func (l *HCLoggerAdapter) Error(msg string, args ...interface{}) {
	l.logger.Error(msg, args...)
}

// Fatal 致命错误日志 (hclog 没有 Fatal 方法，使用 Error+os.Exit)
func (l *HCLoggerAdapter) Fatal(msg string, args ...interface{}) {
	l.logger.Error(msg, args...)
	os.Exit(1)
}

// Named 创建命名日志
func (l *HCLoggerAdapter) Named(name string) Logger {
	return NewHCLoggerAdapter(l.logger.Named(name))
}

// With 创建带上下文的日志
func (l *HCLoggerAdapter) With(args ...interface{}) Logger {
	return NewHCLoggerAdapter(l.logger.With(args...))
}

// NewLogger 创建新的日志记录器
func NewLogger(config Config) (Logger, error) {
	var output io.Writer = os.Stdout

	if config.File != "" {
		dir := filepath.Dir(config.File)
		if err := os.MkdirAll(dir, 0700); err != nil {
			return nil, err
		}

		file, err := os.OpenFile(config.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			return nil, err
		}

		output = io.MultiWriter(os.Stdout, file)
	}

	level := hclog.LevelFromString(config.Level)
	if level == hclog.NoLevel {
		level = hclog.Info
	}

	var logger hclog.Logger
	if config.Format == "json" {
		logger = hclog.New(&hclog.LoggerOptions{
			Name:   "toolkit",
			Level:  level,
			Output: output,
			Color:  hclog.ColorOff,
		})
	} else {
		logger = hclog.New(&hclog.LoggerOptions{
			Name:   "toolkit",
			Level:  level,
			Output: output,
			Color:  hclog.AutoColor,
		})
	}

	return NewHCLoggerAdapter(logger), nil
}

// DefaultLogger 创建默认日志记录器
func DefaultLogger() Logger {
	logger := hclog.New(&hclog.LoggerOptions{
		Name:  "toolkit",
		Level: hclog.Info,
		Color: hclog.AutoColor,
	})
	return NewHCLoggerAdapter(logger)
}
