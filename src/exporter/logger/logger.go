package logger

import (
	"log/syslog"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log *zap.Logger

func init() {

	// File path for backup logging
	// Create the log file if it does not exist, or open it in append mode if it does

	// Setting up network syslog writer
	syslogServer := "10.1.0.2:514" // Adjust as necessary
	syslogWriter, err := syslog.Dial("udp", syslogServer, syslog.LOG_LOCAL0, "shidai")
	if err != nil {
		panic("Failed to dial syslog: " + err.Error())
	}

	// Setting up encoders for each output
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	plaintextEncoder := zapcore.NewConsoleEncoder(encoderConfig)

	var combinedCore zapcore.Core

	// fmt.Println("LOGGER: is running without file writer")
	stdoutCore := zapcore.NewCore(plaintextEncoder, zapcore.AddSync(os.Stdout), zap.NewAtomicLevelAt(zapcore.DebugLevel))
	syslogCore := zapcore.NewCore(plaintextEncoder, zapcore.AddSync(syslogWriter), zap.NewAtomicLevelAt(zapcore.DebugLevel))
	combinedCore = zapcore.NewTee(syslogCore, stdoutCore)

	// Create the logger with the combined cores
	log = zap.New(combinedCore, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

}

func GetLogger() *zap.Logger { return log }
