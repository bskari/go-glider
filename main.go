package main

import (
	"flag"
	"fmt"
	"github.com/bskari/go-glider/glider"
	"github.com/op/go-logging"
	"os"
	"os/exec"
	"time"
)

func main() {
	dumpSensorsPtr := flag.Bool("dump", false, "Dump the sensor data")
	glidePtr := flag.Bool("glide", false, "Run the glide test")
	flag.Parse()

	if *dumpSensorsPtr {
		dumpSensors()
	} else if *glidePtr {
		telemetry, err := glider.NewTelemetry()
		if err != nil {
			panic(fmt.Sprintf("Couldn't initialize telemetry: %v", err))
		}

		// Wait for the GPS to get a lock, so we can set the clock
		fmt.Printf("Waiting for timestamp from GPS")
		for i := 0; i < 10; i++ {
			time.Sleep(time.Millisecond * 100)
			glider.ToggleLed()
			time.Sleep(time.Millisecond * 900)
			glider.ToggleLed()
			// Parse any queued up messages
			for j := 0; j < 10; j++ {
				telemetry.GetPosition()
			}
			if telemetry.GetTimestamp() != 0 {
				break
			}
		}
		if telemetry.GetTimestamp() != 0 {
			command := exec.Command("date", "+%s", "-s", fmt.Sprintf("@%v", telemetry.GetTimestamp()))
			err := command.Run()
			if err != nil {
				fmt.Printf("Unable to set time: %v", err)
			}
		} else {
			fmt.Printf("No timestamp received from GPS")
		}

		logger := createLogger()
		logger.Info("Calling Glide")
		glider.Glide(logger)
	} else {
		fmt.Println("Nothing to do")
	}
}

func createLogger() *logging.Logger {
	now := time.Now()
	logName := fmt.Sprintf("%d-%d-%d-%d-%d-%d-glider.log", now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())
	file, err := os.OpenFile(logName, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		panic("Couldn't open log file")
	}
	defer file.Close()

	logger := logging.MustGetLogger("glider")
	stdoutBackend := logging.NewLogBackend(os.Stdout, "", 0)
	fileBackend := logging.NewLogBackend(file, "", 0)

	// For messages written to the file we want to add some additional
	// information to the output, including the used log level and the
	// name of the function.
	logFormat := logging.MustStringFormatter(
		`%{color}%{time:15:04:05.000} %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`,
	)
	fileFormatter := logging.NewBackendFormatter(fileBackend, logFormat)

	// Only info and up should be sent to backend
	stdoutLeveled := logging.AddModuleLevel(stdoutBackend)
	stdoutLeveled.SetLevel(logging.ERROR, "")

	// Set the backends to be used
	logging.SetBackend(stdoutLeveled, fileFormatter)

	return logger
}
