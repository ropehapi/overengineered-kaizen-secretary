package logger

import (
	"os"

	"go.uber.org/zap"
)

func Init() {
	var logger *zap.Logger
	var err error

	if os.Getenv("APP_ENV") == "production" {
		logger, err = zap.NewProduction()
	} else {
		logger, err = zap.NewDevelopment()
	}
	if err != nil {
		panic("failed to initialize zap logger: " + err.Error())
	}
	zap.ReplaceGlobals(logger)
}
