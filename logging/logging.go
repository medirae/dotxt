package logging

import (
	"dotxt/utils"
	"io"
	"os"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	Logger        *zap.SugaredLogger
	ConsoleLogger *zap.SugaredLogger

	FileLevel  int
	fileHandle *os.File
	fileCore   *zapcore.Core
	fileEnc    zapcore.Encoder

	ConsoleLevel int
	consoleCore  *zapcore.Core
	consoleEnc   = zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
)

func init() {
	initFileEnc()
	InitConsole(false)
	Initialize()
}

func InitConsole(quiet bool) {
	var out io.Writer = os.Stdout
	if quiet {
		out = io.Discard
	}
	consoleCore = utils.MkPtr(zapcore.NewCore(
		consoleEnc,
		zapcore.Lock(zapcore.AddSync(out)),
		zapcore.Level(ConsoleLevel),
	))
}

func initFileEnc() {
	fileEncCfg := zap.NewProductionEncoderConfig()
	fileEncCfg.TimeKey = "ts"
	fileEncCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	fileEncCfg.EncodeLevel = zapcore.CapitalLevelEncoder
	fileEnc = zapcore.NewConsoleEncoder(fileEncCfg)
}

func InitFile(logpath string) error {
	f, err := os.OpenFile(logpath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	fileHandle = f

	FileLevel = viper.GetInt("logging.file-level")
	fileCore = utils.MkPtr(zapcore.NewCore(
		fileEnc,
		zapcore.AddSync(io.Writer(f)),
		zapcore.Level(FileLevel),
	))
	return nil
}

func Initialize() {
	var core zapcore.Core
	if fileCore != nil {
		core = zapcore.NewTee(*consoleCore, *fileCore)
	} else {
		core = zapcore.NewTee(*consoleCore)
	}
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.PanicLevel))
	Logger = logger.Sugar()
	ConsoleLogger = zap.New(*consoleCore, zap.AddCaller(), zap.AddStacktrace(zapcore.PanicLevel)).Sugar()
}

func Close() error {
	if fileHandle != nil {
		return fileHandle.Close()
	}
	return nil
}
