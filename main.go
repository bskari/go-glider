package main

import (
	"flag"
	"fmt"
	"github.com/bskari/go-glider/glider"
	"os"
	"os/exec"
	"strings"
	"time"
	"io/ioutil"
)

func main() {
	dumpSensorsPtr := flag.Bool("dump", false, "Dump the sensor data")
	glidePtr := flag.Bool("glide", false, "Run the glide test")
	flag.Parse()

	os.mkdir("logs", 0644)

	if *dumpSensorsPtr {
		dumpSensors()
	} else if *glidePtr {
		logName := getLogName(true)
		fileLog, err := os.OpenFile(logName, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		defer fileLog.Close()
		glider.ConfigureLogger(fileLog)

		runGlide()
	} else {
		fmt.Println("Nothing to do")
	}
}

func runGlide() {
	telemetry, err := glider.NewTelemetry()
	if err != nil {
		panic(fmt.Sprintf("Couldn't initialize telemetry: %v", err))
	}

	// Wait for the GPS to get a lock, so we can set the clock
	timeSet := false
	fmt.Println("Waiting for timestamp from GPS")
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
			fmt.Printf("Unable to set time: %v\n", err)
		}
		timeSet = true
	} else {
		fmt.Println("No timestamp received from GPS")
	}

	logName := getLogName(timeSet)
	fileLog, err := os.OpenFile(logName, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer fileLog.Close()
	glider.ConfigureLogger(fileLog)
	glider.GetLogger().Info("Calling Glide")
	glider.Glide()
}

func getLogName(timeSet bool) string {
	logName := "1.log"
	if timeSet {
		now := time.Now()
		logName = fmt.Sprintf("%04d-%02d-%02d-%02d-%02d-%02d-glider.log", now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())
	} else {
		// Just list the files in numerical order
		entries, err := ioutil.ReadDir(".")
		if err != nil {
			fmt.Printf("Unable to list directory contents: %v\n", err)
		} else {
			for fileNumber := 1; fileNumber < 100; fileNumber++ {
				logName = fmt.Sprintf("%d.log", fileNumber)
				alreadyExists := false
				for _, entry := range entries {
					if strings.HasSuffix(entry.Name(), logName) {
						alreadyExists = false
						break
					}
				}
				if !alreadyExists {
					break
				}
			}
		}
	}
	return "logs/" + logName
}
