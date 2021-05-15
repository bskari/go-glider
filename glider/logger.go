package glider

import (
	"fmt"
	"github.com/fatih/color"
	"os"
	"time"
)

func ConfigureLogger(file *os.File) {
	fileLog = file
}

type MultiLogger struct {
	warningColor *color.Color
	errorColor   *color.Color
}

var fileLog *os.File
var Logger = newLogger()

func newLogger() MultiLogger {
	return MultiLogger{
		errorColor:   color.New(color.FgRed),
		warningColor: color.New(color.FgYellow),
	}
}

func getFileFormatString(level string) string {
	now := time.Now()
	return fmt.Sprintf("%s %4s %%s\n", now.Format("15:04:05.000"), level)
}

func (*MultiLogger) Debug(msg string) {
	format := getFileFormatString("DEBU")
	if fileLog != nil {
		fileLog.Write([]byte(fmt.Sprintf(format, msg)))
	}
	//fmt.Printf(format, msg)
	//logDashboard(msg)
}
func (*MultiLogger) Debugf(msg string, args ...interface{}) {
	format := getFileFormatString("DEBU")
	newMsg := fmt.Sprintf(msg, args...)
	if fileLog != nil {
		fileLog.Write([]byte(fmt.Sprintf(format, newMsg)))
	}
	//fmt.Printf(format, newMsg)
	//logDashboard(newMsg)
}
func (*MultiLogger) Info(msg string) {
	format := getFileFormatString("INFO")
	if fileLog != nil {
		fileLog.Write([]byte(fmt.Sprintf(format, msg)))
	}
	//fmt.Printf(format, msg)
	logDashboard(msg)
}
func (*MultiLogger) Infof(msg string, args ...interface{}) {
	format := getFileFormatString("INFO")
	newMsg := fmt.Sprintf(msg, args...)
	if fileLog != nil {
		fileLog.Write([]byte(fmt.Sprintf(format, newMsg)))
	}
	//fmt.Printf(format, newMsg)
	logDashboard(newMsg)
}
func (logger *MultiLogger) Warning(msg string) {
	format := getFileFormatString("WARN")
	if fileLog != nil {
		fileLog.Write([]byte(fmt.Sprintf(format, msg)))
	}
	//logger.warningColor.Printf(format, msg)
	logDashboard(msg)
}
func (logger *MultiLogger) Warningf(msg string, args ...interface{}) {
	format := getFileFormatString("WARN")
	newMsg := fmt.Sprintf(msg, args...)
	if fileLog != nil {
		fileLog.Write([]byte(fmt.Sprintf(format, newMsg)))
	}
	//logger.warningColor.Printf(format, newMsg)
	logDashboard(newMsg)
}
func (logger *MultiLogger) Error(msg string) {
	format := getFileFormatString("ERRO")
	if fileLog != nil {
		fileLog.Write([]byte(fmt.Sprintf(format, msg)))
	}
	//logger.errorColor.Printf(format, msg)
	logDashboard(msg)
}
func (logger *MultiLogger) Errorf(msg string, args ...interface{}) {
	format := getFileFormatString("ERRO")
	newMsg := fmt.Sprintf(msg, args...)
	if fileLog != nil {
		fileLog.Write([]byte(fmt.Sprintf(format, newMsg)))
	}
	//logger.errorColor.Printf(format, newMsg)
	logDashboard(newMsg)
}
func (logger *MultiLogger) Critical(msg string) {
	format := getFileFormatString("CRIT")
	if fileLog != nil {
		fileLog.Write([]byte(fmt.Sprintf(format, msg)))
	}
	//logger.errorColor.Printf(format, msg)
	logDashboard(msg)
}
func (logger *MultiLogger) Criticalf(msg string, args ...interface{}) {
	format := getFileFormatString("CRIT")
	newMsg := fmt.Sprintf(msg, args...)
	if fileLog != nil {
		fileLog.Write([]byte(fmt.Sprintf(format, newMsg)))
	}
	//logger.errorColor.Printf(format, newMsg)
	logDashboard(newMsg)
}
