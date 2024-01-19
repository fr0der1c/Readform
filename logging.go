package main

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var logger *logrus.Logger

var onceLogger sync.Once

func initLogger() {
	onceLogger.Do(func() {
		logger = logrus.New()
		logger.SetReportCaller(true)
		logger.SetFormatter(new(MyFormatter))
		logger.SetOutput(os.Stdout)
		logger.SetLevel(logrus.DebugLevel)
	})

}

type MyFormatter struct{}

func (f *MyFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	file := ""
	if entry.HasCaller() {
		file = fmt.Sprintf("%s:%d", filepath.Base(entry.Caller.File), entry.Caller.Line)
	}
	msg := fmt.Sprintf("%s %s (%s) %s\n", strings.ToUpper(entry.Level.String()), timestamp, file, entry.Message)

	switch entry.Level {
	case logrus.WarnLevel:
		msg = color.YellowString(msg)
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		msg = color.RedString(msg)
	default:
		msg = color.BlueString(msg)
	}

	return []byte(msg), nil
}
