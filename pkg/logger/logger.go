package logger

import (
	"fmt"
	"os"
	"path"

	"github.com/yockii/dify_tools/pkg/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var logger *zap.Logger

// F 用于创建日志字段的简写
func F(key string, value interface{}) zap.Field {
	return zap.Any(key, value)
}

// 初始化日志
func Init() {
	logFile := config.GetString("log.filename")
	if logFile == "" {
		logFile = "logs/app.log"
	}

	// 确保日志目录存在
	logDir := path.Dir(logFile)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Printf("创建日志目录失败，无法记录日志到文件: %v", err)
		return
	}

	// 配置日志输出
	w := zapcore.AddSync(&lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    config.GetInt("log.max_size"), // MB
		MaxBackups: config.GetInt("log.max_backups"),
		MaxAge:     config.GetInt("log.max_age"),   // days
		Compress:   config.GetBool("log.compress"), // 是否压缩
	})

	// 配置编码器
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	// 创建核心
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		w,
		zap.InfoLevel,
	)

	// 创建logger
	logger = zap.New(core, zap.AddCaller())
}

// Debug 调试级别日志
func Debug(msg string, fields ...zap.Field) {
	logger.Debug(msg, fields...)
}

// Info 信息级别日志
func Info(msg string, fields ...zap.Field) {
	logger.Info(msg, fields...)
}

// Warn 警告级别日志
func Warn(msg string, fields ...zap.Field) {
	logger.Warn(msg, fields...)
}

// Error 错误级别日志
func Error(msg string, fields ...zap.Field) {
	logger.Error(msg, fields...)
}

// Fatal 致命级别日志
func Fatal(msg string, fields ...zap.Field) {
	logger.Fatal(msg, fields...)
}

// Sync 同步日志缓冲
func Sync() error {
	return logger.Sync()
}

// GetLogger 获取原始logger实例
func GetLogger() *zap.Logger {
	return logger
}
