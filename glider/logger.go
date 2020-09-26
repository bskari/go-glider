package glider

import (
	"fmt"
	"os"
	"time"
)

func ConfigureLogger(file *os.File) {
	fileLog = file
}

type MultiLogger struct {
}

var fileLog *os.File
var Logger = &MultiLogger{}

func getFileFormatString(level string) string {
	now := time.Now()
	return fmt.Sprintf("%s %4s %%s\n", now.Format("15:04:05.000"), level)
}

func (*MultiLogger) Debug(msg string) {
	format := getFileFormatString("DEBU")
	if fileLog != nil {
		fileLog.Write([]byte(fmt.Sprintf(format, msg)))
	}
	fmt.Printf(format, msg)
}
func (*MultiLogger) Debugf(msg string, args ...interface{}) {
	format := getFileFormatString("DEBU")
	newMsg := fmt.Sprintf(msg, args...)
	if fileLog != nil {
		fileLog.Write([]byte(fmt.Sprintf(format, newMsg)))
	}
	fmt.Printf(format, newMsg)
}
func (*MultiLogger) Info(msg string) {
	format := getFileFormatString("INFO")
	if fileLog != nil {
		fileLog.Write([]byte(fmt.Sprintf(format, msg)))
	}
	fmt.Printf(format, msg)
}
func (*MultiLogger) Infof(msg string, args ...interface{}) {
	format := getFileFormatString("INFO")
	newMsg := fmt.Sprintf(msg, args...)
	if fileLog != nil {
		fileLog.Write([]byte(fmt.Sprintf(format, newMsg)))
	}
	fmt.Printf(format, newMsg)
}
func (*MultiLogger) Notice(msg string) {
	format := getFileFormatString("NOTI")
	if fileLog != nil {
		fileLog.Write([]byte(fmt.Sprintf(format, msg)))
	}
	fmt.Printf(format, msg)
}
func (*MultiLogger) Noticef(msg string, args ...interface{}) {
	format := getFileFormatString("NOTI")
	newMsg := fmt.Sprintf(msg, args...)
	if fileLog != nil {
		fileLog.Write([]byte(fmt.Sprintf(format, newMsg)))
	}
	fmt.Printf(format, newMsg)
}
func (*MultiLogger) Warning(msg string) {
	format := getFileFormatString("WARN")
	if fileLog != nil {
		fileLog.Write([]byte(fmt.Sprintf(format, msg)))
	}
	fmt.Printf(format, msg)
}
func (*MultiLogger) Warningf(msg string, args ...interface{}) {
	format := getFileFormatString("WARN")
	newMsg := fmt.Sprintf(msg, args...)
	if fileLog != nil {
		fileLog.Write([]byte(fmt.Sprintf(format, newMsg)))
	}
	fmt.Printf(format, newMsg)
}
func (*MultiLogger) Error(msg string) {
	format := getFileFormatString("ERRO")
	if fileLog != nil {
		fileLog.Write([]byte(fmt.Sprintf(format, msg)))
	}
	fmt.Printf(format, msg)
}
func (*MultiLogger) Errorf(msg string, args ...interface{}) {
	format := getFileFormatString("ERRO")
	newMsg := fmt.Sprintf(msg, args...)
	if fileLog != nil {
		fileLog.Write([]byte(fmt.Sprintf(format, newMsg)))
	}
	fmt.Printf(format, newMsg)
}
func (*MultiLogger) Critical(msg string) {
	format := getFileFormatString("CRIT")
	if fileLog != nil {
		fileLog.Write([]byte(fmt.Sprintf(format, msg)))
	}
	fmt.Printf(format, msg)
}
func (*MultiLogger) Criticalf(msg string, args ...interface{}) {
	format := getFileFormatString("CRIT")
	newMsg := fmt.Sprintf(msg, args...)
	if fileLog != nil {
		fileLog.Write([]byte(fmt.Sprintf(format, newMsg)))
	}
	fmt.Printf(format, newMsg)
}
