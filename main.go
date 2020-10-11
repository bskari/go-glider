package main

import (
	"flag"
	"fmt"
	"github.com/bskari/go-glider/glider"
	"github.com/stianeikeland/go-rpio/v4"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"
)

// The pins are numbered 1-40, starting on the inside next to the SD card, then
// going to its twin, then down.
// GPS connections:
// 1 = 3.3V, connect to GPS VCC
// 6 = ground, connect to GPS ground
// 8 = TXD, connect to GPS RXD
// 10 = RXD, connect to GPS TXD
// Accelerometer, LSM303DLHC connections:
// 2 = 5.5V, connect to accelerometer VIN
// 9 = ground, connect to accelerometer ground
// 5 = SCL, connect to accelerometer SCL
// 3 = SDA, connect to accelerometer SDA
// Servo connections:
// 32 (BCM 12) = left servo yellow wire
// 33 (BCM 13) = right servo yellow wire
// 34 = ground, connect to AA black
// servo orange = +, connect to AA red
// servo brown = -, connect to AA black
// Button connections:
// 18 (BCM 24) = connect to button
// 20 = ground, connect to button

func main() {
	if os.Getuid() != 0 {
		fmt.Println("Must run as root")
		return
	}

	dumpSensorsPtr := flag.Bool("dump", false, "Dump the sensor data")
	glidePtr := flag.Bool("glide", false, "Run the glide test")
	servoPtr := flag.Bool("servo", false, "Run the servo test")
	flag.Parse()

	os.Mkdir("logs", 0655)

	err := rpio.Open()
	if err != nil {
		fmt.Printf("Failed to initialize RPIO: %v\n", err)
		return
	}
	defer rpio.Close()

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
	} else if *servoPtr {
		testServos()
	} else {
		flag.PrintDefaults()
	}
}

func runGlide() {
	telemetry, err := glider.NewTelemetry()
	if err != nil {
		panic(fmt.Sprintf("Couldn't initialize telemetry: %v", err))
	}

	// Wait for the GPS to get a lock, so we can set the clock
	timeSet := false
	glider.Logger.Info("Waiting for timestamp from GPS")
	for i := 0; i < 10; i++ {
		time.Sleep(time.Millisecond * 100)
		glider.ToggleLed()
		time.Sleep(time.Millisecond * 900)
		glider.ToggleLed()
		// Parse a queued up message
		_, err := telemetry.ParseQueuedMessage()
		if err != nil {
			glider.Logger.Errorf("Unable to parse GPS message: %v", err)
			break
		}
		// 1601261144 = September 27 2020
		if telemetry.GetTimestamp() > 1601261144 {
			break
		}
	}
	timestamp := telemetry.GetTimestamp()
	if timestamp > 1601261144 {
		now := time.Unix(timestamp, 0)
		formatted := now.Format("Jan 2 15:04:05 2006 -0700 MST")
		glider.Logger.Infof("Setting timestamp to %v (%v)", timestamp, formatted)
		glider.Logger.Debugf("Running `date +%%s -s @%v", fmt.Sprintf("@%v", timestamp))
		command := exec.Command("date", "+%s", "-s", fmt.Sprintf("@%v", timestamp))
		err := command.Run()
		if err != nil {
			glider.Logger.Errorf("Unable to set time: %v", err)
		}
		timeSet = true
	} else {
		now := time.Unix(timestamp, 0)
		formatted := now.Format("Jan 2 15:04:05 2006 -0700 MST")
		glider.Logger.Warningf("Bad timestamp received from GPS: %v (%v)", timestamp, formatted)
	}

	logName := getLogName(timeSet)
	fileLog, err := os.OpenFile(logName, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer fileLog.Close()
	glider.ConfigureLogger(fileLog)
	glider.Logger.Info("Starting Pilot")
	pilot, err := glider.NewPilot()
	if err != nil {
		glider.Logger.Errorf("Couldn't create Pilot: %v", err)
	}
	pilot.RunGlideTestForever()
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
			glider.Logger.Errorf("Unable to list directory contents: %v", err)
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
