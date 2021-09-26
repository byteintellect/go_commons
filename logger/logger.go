package logger

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"time"
)

func createDirectoryIfNotExists() error {
	path, err := os.Getwd()
	if err != nil {
		return err
	}
	if _, err := os.Stat(fmt.Sprintf("%s/logs", path)); os.IsNotExist(err) {
		_ = os.Mkdir("logs", os.ModePerm)
	}
	return nil
}

func getLogWriter(appName string) (zapcore.WriteSyncer, error) {
	path, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	timeNow := time.Now().Format("20060102150405")
	logFile, err := os.OpenFile(fmt.Sprintf("%v/logs/%v-%v.log", path, appName, timeNow), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return zapcore.AddSync(logFile), nil
}

func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.TimeEncoder(func(t time.Time, encoder zapcore.PrimitiveArrayEncoder) {
		encoder.AppendString(t.UTC().Format(time.RFC3339))
	})
	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	return zapcore.NewConsoleEncoder(encoderConfig)
}

func getLogLevel() zapcore.LevelEnabler {
	logLevel, found := os.LookupEnv("LOG_LEVEL")
	if !found {
		return zap.InfoLevel
	}
	switch logLevel {
	case "INFO":
		return zap.InfoLevel
	case "DEBUG":
		return zap.DebugLevel
	case "WARN":
		return zap.WarnLevel
	case "ERROR":
		return zap.ErrorLevel
	case "DPANIC":
		return zap.DPanicLevel
	case "PANIC":
		return zap.PanicLevel
	case "FATAL":
		return zap.FatalLevel
	default:
		return zap.InfoLevel
	}
}

func InitLogger() (*zap.Logger, error) {
	if err := createDirectoryIfNotExists(); err != nil {
		return nil, err
	}
	writerSync, err := getLogWriter(os.Getenv("APP_NAME"))
	if err != nil {
		return nil, err
	}
	encoder := getEncoder()
	core := zapcore.NewTee(
		zapcore.NewCore(encoder, writerSync, getLogLevel()),
		zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), getLogLevel()),
	)
	logger := zap.New(core, zap.AddCaller())
	zap.ReplaceGlobals(logger)
	return logger, nil
}
