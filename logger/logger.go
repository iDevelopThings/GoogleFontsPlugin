package logger

import (
	config "github.com/go-ozzo/ozzo-config"
	log "github.com/go-ozzo/ozzo-log"
)

var Logger = log.NewLogger()

func Emergency(format string, a ...any) { Logger.Emergency(format, a...) }
func Alert(format string, a ...any)     { Logger.Alert(format, a...) }
func Critical(format string, a ...any)  { Logger.Critical(format, a...) }
func Error(format string, a ...any)     { Logger.Error(format, a...) }
func Warning(format string, a ...any)   { Logger.Warning(format, a...) }
func Notice(format string, a ...any)    { Logger.Notice(format, a...) }
func Info(format string, a ...any)      { Logger.Info(format, a...) }
func Debug(format string, a ...any)     { Logger.Debug(format, a...) }

func Init(c *config.Config) {
	c.Register("ConsoleTarget", log.NewConsoleTarget)
	c.Register("FileTarget", log.NewFileTarget)

	if err := c.Configure(Logger, "Logger"); err != nil {
		panic(err)
	}

	if err := Logger.Open(); err != nil {
		panic(err)
	}

	Debug("Logger initialized")
}
