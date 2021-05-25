package loopia

import (
	"os"
	"strings"
	"sync"

	"go.uber.org/zap"
)

func init() {
	if strings.Contains(os.Getenv("DEBUG"), "libdns-loopia") {
		defaultLogger, _ = newDefaultDevelopmentLog()
	} else {
		defaultLogger, _ = newDefaultProductionLog()
	}
}

func newDefaultProductionLog() (*zap.SugaredLogger, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}
	return logger.Sugar(), err

}

func newDefaultDevelopmentLog() (*zap.SugaredLogger, error) {
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.FunctionKey = "F"
	logger, err := config.Build()
	if err != nil {
		return nil, err
	}
	return logger.Sugar(), err
}

func Log() *zap.SugaredLogger {
	defaultLoggerMu.RLock()
	defer defaultLoggerMu.RUnlock()
	return defaultLogger
}

var (
	defaultLogger   *zap.SugaredLogger
	defaultLoggerMu sync.RWMutex
)
