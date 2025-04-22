package logger

import (
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// Logger 是全局日志实例
	Logger *zap.Logger
	once   sync.Once
)

// LogConfig 日志配置
type LogConfig struct {
	Level      string // 日志级别: debug, info, warn, error, dpanic, panic, fatal
	Encoding   string // 编码方式: json, console
	OutputPath string // 输出路径: stdout, stderr, /path/to/file
	ErrorPath  string // 错误日志路径
}

// NewDefaultConfig 返回默认配置
func NewDefaultConfig() LogConfig {
	return LogConfig{
		Level:      "info",
		Encoding:   "console",
		OutputPath: "stdout",
		ErrorPath:  "stderr",
	}
}

// InitLogger 初始化日志
func InitLogger(config LogConfig) *zap.Logger {
	once.Do(func() {
		// 设置日志级别
		var level zapcore.Level
		switch config.Level {
		case "debug":
			level = zap.DebugLevel
		case "info":
			level = zap.InfoLevel
		case "warn":
			level = zap.WarnLevel
		case "error":
			level = zap.ErrorLevel
		case "dpanic":
			level = zap.DPanicLevel
		case "panic":
			level = zap.PanicLevel
		case "fatal":
			level = zap.FatalLevel
		default:
			level = zap.InfoLevel
		}

		// 配置编码器
		encoderConfig := zapcore.EncoderConfig{
			TimeKey:        "time",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}

		// 配置输出
		var outputPaths, errorOutputPaths []string
		if config.OutputPath == "" {
			outputPaths = []string{"stdout"}
		} else {
			outputPaths = []string{config.OutputPath}
		}

		if config.ErrorPath == "" {
			errorOutputPaths = []string{"stderr"}
		} else {
			errorOutputPaths = []string{config.ErrorPath}
		}

		// 创建Zap配置
		zapConfig := zap.Config{
			Level:             zap.NewAtomicLevelAt(level),
			Development:       false,
			DisableCaller:     false,
			DisableStacktrace: false,
			Sampling: &zap.SamplingConfig{
				Initial:    100,
				Thereafter: 100,
			},
			Encoding:         config.Encoding,
			EncoderConfig:    encoderConfig,
			OutputPaths:      outputPaths,
			ErrorOutputPaths: errorOutputPaths,
		}

		var err error
		Logger, err = zapConfig.Build(zap.AddCallerSkip(1))
		if err != nil {
			panic("无法初始化日志: " + err.Error())
		}
	})

	return Logger
}

// GetLogger 获取日志实例
func GetLogger() *zap.Logger {
	if Logger == nil {
		return InitLogger(NewDefaultConfig())
	}
	return Logger
}

// Debug 记录debug级别日志
func Debug(msg string, fields ...zap.Field) {
	GetLogger().Debug(msg, fields...)
}

// Info 记录info级别日志
func Info(msg string, fields ...zap.Field) {
	GetLogger().Info(msg, fields...)
}

// Warn 记录warn级别日志
func Warn(msg string, fields ...zap.Field) {
	GetLogger().Warn(msg, fields...)
}

// Error 记录error级别日志
func Error(msg string, fields ...zap.Field) {
	GetLogger().Error(msg, fields...)
}

// DPanic 记录dpanic级别日志
func DPanic(msg string, fields ...zap.Field) {
	GetLogger().DPanic(msg, fields...)
}

// Panic 记录panic级别日志
func Panic(msg string, fields ...zap.Field) {
	GetLogger().Panic(msg, fields...)
}

// Fatal 记录fatal级别日志
func Fatal(msg string, fields ...zap.Field) {
	GetLogger().Fatal(msg, fields...)
}

// Sync 执行所有缓冲日志操作
func Sync() {
	if Logger != nil {
		Logger.Sync()
	}
}

// WithRequestID 创建带有请求ID的日志字段
func WithRequestID(requestID string) zap.Field {
	return zap.String("request_id", requestID)
}

// WithUserID 创建带有用户ID的日志字段
func WithUserID(userID uint64) zap.Field {
	return zap.Uint64("user_id", userID)
}

// WithError 创建带有错误信息的日志字段
func WithError(err error) zap.Field {
	return zap.Error(err)
}

// WithDuration 创建带有执行时间的日志字段
func WithDuration(duration time.Duration) zap.Field {
	return zap.Duration("duration", duration)
}
